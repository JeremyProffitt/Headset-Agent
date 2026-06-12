package main

// Handler-level integration tests for session-store wiring (B-02) and
// frustration-counter accumulation (B-06).
//
// Mockability approach:
//   The three package-level vars that touch AWS (sessStore, agentClient,
//   personaLoader) are declared as interfaces (sessionStorer, agentInvoker,
//   personaLoaderI respectively). Tests replace them with in-process stubs
//   before calling handleLexRequest, then restore the originals via t.Cleanup.
//   No DynamoDB, Bedrock, or SSM calls are made during testing.
//   The agentConfig global is also reset so SSM is never consulted.

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/headset-support-agent/internal/agents"
	"github.com/headset-support-agent/internal/handlers"
	"github.com/headset-support-agent/internal/models"
	"github.com/headset-support-agent/internal/session"
	"github.com/headset-support-agent/internal/triage"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

// mockSessionStore is an in-memory sessionStorer. It stores sessions by ID
// so successive handleLexRequest calls within one test share state, exactly
// as they would in production with a real DynamoDB table.
type mockSessionStore struct {
	sessions map[string]*models.Session
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{sessions: make(map[string]*models.Session)}
}

func (m *mockSessionStore) Load(_ context.Context, sessionID string) (*models.Session, error) {
	if s, ok := m.sessions[sessionID]; ok {
		// Return a shallow copy so the handler's mutations don't alias our stored copy.
		copy := *s
		attrsCopy := make(map[string]string, len(s.Attributes))
		for k, v := range s.Attributes {
			attrsCopy[k] = v
		}
		copy.Attributes = attrsCopy
		return &copy, nil
	}
	return &models.Session{
		SessionID:  sessionID,
		Attributes: make(map[string]string),
	}, nil
}

func (m *mockSessionStore) Save(_ context.Context, sess *models.Session) error {
	// Store a deep copy so later turns can Load the saved state.
	copy := *sess
	attrsCopy := make(map[string]string, len(sess.Attributes))
	for k, v := range sess.Attributes {
		attrsCopy[k] = v
	}
	copy.Attributes = attrsCopy
	m.sessions[sess.SessionID] = &copy
	return nil
}

// mockAgentInvoker returns canned success responses so Bedrock is never called.
// ragText/ragErr control the RetrieveAndGenerate (A-08) path; lastRAGRequest
// captures the most recent request for assertions.
type mockAgentInvoker struct {
	ragText        string
	ragErr         error
	lastRAGRequest *agents.RetrieveAndGenerateRequest
}

func (m *mockAgentInvoker) InvokeAgent(_ context.Context, input agents.InvokeAgentInput) (*models.AgentResponse, error) {
	return &models.AgentResponse{
		OutputText: "Stub bedrock response",
		SessionID:  input.SessionID,
	}, nil
}

func (m *mockAgentInvoker) RetrieveAndGenerate(_ context.Context, req agents.RetrieveAndGenerateRequest) (*agents.KBAnswer, error) {
	m.lastRAGRequest = &req
	if m.ragErr != nil {
		return nil, m.ragErr
	}
	text := m.ragText
	if text == "" {
		text = "Stub KB-grounded answer"
	}
	return &agents.KBAnswer{
		Text:      text,
		Citations: []string{"s3://kb-bucket/trees/preflight-checklist.md"},
	}, nil
}

// mockPersonaLoader returns the default persona without hitting DynamoDB.
type mockPersonaLoader struct{}

func (m *mockPersonaLoader) Load(_ context.Context, _ string) (*models.Persona, error) {
	return &models.Persona{
		PersonaID:   "tangerine",
		DisplayName: "Tangerine Test",
		VoiceConfig: models.VoiceConfig{
			Prosody: models.Prosody{Rate: "100%", Pitch: "medium"},
		},
		Phrases: models.Phrases{
			Escalation: []string{},
		},
	}, nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// setupMocks replaces the three AWS-backed globals with in-memory stubs and
// resets agentConfig so SSM is never consulted. It returns the mock store so
// callers can inspect session state, and registers a t.Cleanup to restore
// everything at the end of the test.
func setupMocks(t *testing.T) *mockSessionStore {
	t.Helper()

	store := newMockSessionStore()

	origSessStore := sessStore
	origAgentClient := agentClient
	origPersonaLoader := personaLoader
	origLoaded := agentConfig.loaded
	origAgentID := agentConfig.agentID
	origAgentAlias := agentConfig.agentAlias

	sessStore = store
	agentClient = &mockAgentInvoker{}
	personaLoader = &mockPersonaLoader{}

	// Default to the legacy (no knowledge base) path; tests that exercise the
	// A-08 KB retrieval path override KB_ID themselves. t.Setenv restores the
	// original value on cleanup.
	t.Setenv("KB_ID", "")

	// Provide a non-empty, non-PLACEHOLDER agent config so the handler
	// reaches the Bedrock invocation path (or escalation, whichever fires).
	agentConfig.Lock()
	agentConfig.agentID = "test-agent-id"
	agentConfig.agentAlias = "test-alias-id"
	agentConfig.loaded = true
	agentConfig.Unlock()

	t.Cleanup(func() {
		sessStore = origSessStore
		agentClient = origAgentClient
		personaLoader = origPersonaLoader
		agentConfig.Lock()
		agentConfig.loaded = origLoaded
		agentConfig.agentID = origAgentID
		agentConfig.agentAlias = origAgentAlias
		agentConfig.Unlock()
	})

	return store
}

// makeLexEvent builds a minimal LexV2Event for the given session and transcript.
func makeLexEvent(sessionID, transcript string) LexV2Event {
	return LexV2Event{
		SessionID:       sessionID,
		InputTranscript: transcript,
		SessionState: SessionState{
			SessionAttributes: map[string]string{},
		},
	}
}

// ---------------------------------------------------------------------------
// B-02 basic wiring: session is loaded and saved each turn
// ---------------------------------------------------------------------------

func TestHandleLexRequest_SessionAttributesPersistedAcrossTurns(t *testing.T) {
	store := setupMocks(t)
	ctx := context.Background()
	sid := "sess-persist-test"

	// Turn 1: benign input — no escalation, Bedrock stub fires.
	_, err := handleLexRequest(ctx, makeLexEvent(sid, "my headset is quiet"))
	if err != nil {
		t.Fatalf("turn 1: unexpected error: %v", err)
	}

	// The session should now exist in the store.
	stored, loadErr := store.Load(ctx, sid)
	if loadErr != nil {
		t.Fatalf("load after turn 1 failed: %v", loadErr)
	}
	if stored.SessionID != sid {
		t.Errorf("expected session_id=%q, got %q", sid, stored.SessionID)
	}
}

// ---------------------------------------------------------------------------
// B-06: frustration counter accumulates across 3 turns → escalation fires
// ---------------------------------------------------------------------------

// TestHandleLexRequest_FrustrationAccumulatesAcross3Turns is the definitive
// handler-level test that proves the B-06 fix is working end-to-end:
//
//	Turn 1: transcript contains one frustration phrase ("this is ridiculous").
//	        frustrationCount read from session = 0. DetectEscalation returns
//	        FrustrationDelta=1, ShouldEscalate=false (0+1 < 3). Handler persists
//	        frustration_count=1.
//
//	Turn 2: transcript contains one frustration phrase ("doesn't work").
//	        frustrationCount read from session = 1. DetectEscalation returns
//	        FrustrationDelta=1, ShouldEscalate=false (1+1 < 3). Handler persists
//	        frustration_count=2.
//
//	Turn 3: transcript contains one frustration phrase ("still not working").
//	        frustrationCount read from session = 2. DetectEscalation returns
//	        FrustrationDelta=1, ShouldEscalate=true (2+1 >= 3). Handler returns
//	        escalation response with reason=user_frustrated.
func TestHandleLexRequest_FrustrationAccumulatesAcross3Turns(t *testing.T) {
	store := setupMocks(t)
	ctx := context.Background()
	sid := "sess-frustration-test"

	turns := []struct {
		transcript      string
		wantEscalate    bool
		wantFrustration int // expected persisted frustration_count after this turn
	}{
		{"this is ridiculous", false, 1},
		{"doesn't work", false, 2},
		{"still not working", true, 3},
	}

	for i, tc := range turns {
		turnNum := i + 1
		resp, err := handleLexRequest(ctx, makeLexEvent(sid, tc.transcript))
		if err != nil {
			t.Fatalf("turn %d: unexpected error: %v", turnNum, err)
		}

		// Check escalation flag in response session attributes.
		sa := resp.SessionState.SessionAttributes
		escalated := sa["escalation_requested"] == "true"
		if escalated != tc.wantEscalate {
			t.Errorf("turn %d: escalation_requested=%v, want %v", turnNum, escalated, tc.wantEscalate)
		}

		if tc.wantEscalate {
			if sa["escalation_reason"] != "user_frustrated" {
				t.Errorf("turn %d: expected escalation_reason=user_frustrated, got %q",
					turnNum, sa["escalation_reason"])
			}
		}

		// Verify persisted frustration_count in the store.
		stored, loadErr := store.Load(ctx, sid)
		if loadErr != nil {
			t.Fatalf("turn %d: load failed: %v", turnNum, loadErr)
		}
		got := session.GetFrustrationCount(stored)
		if got != tc.wantFrustration {
			t.Errorf("turn %d: persisted frustration_count=%d, want %d", turnNum, got, tc.wantFrustration)
		}
	}
}

// ---------------------------------------------------------------------------
// B-02: merge precedence — Lex turn attrs win over stored attrs on collision
// ---------------------------------------------------------------------------

func TestHandleLexRequest_LexAttrsWinOnCollision(t *testing.T) {
	store := setupMocks(t)
	ctx := context.Background()
	sid := "sess-merge-test"

	// Pre-seed the session store with a persona_id.
	seedSess := &models.Session{
		SessionID:  sid,
		Attributes: map[string]string{"persona_id": "joseph"},
	}
	if err := store.Save(ctx, seedSess); err != nil {
		t.Fatalf("seed save failed: %v", err)
	}

	// Turn arrives with persona_id=jennifer in the Lex sessionAttributes.
	event := makeLexEvent(sid, "my headset has no sound")
	event.SessionState.SessionAttributes["persona_id"] = "jennifer"

	_, err := handleLexRequest(ctx, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The session store should now reflect jennifer (Lex attr won the collision).
	stored, loadErr := store.Load(ctx, sid)
	if loadErr != nil {
		t.Fatalf("load failed: %v", loadErr)
	}
	if stored.Attributes["persona_id"] != "jennifer" {
		t.Errorf("expected persona_id=jennifer (Lex attr wins), got %q",
			stored.Attributes["persona_id"])
	}
}

// ---------------------------------------------------------------------------
// B-02: graceful degrade — Load error does not fail the turn
// ---------------------------------------------------------------------------

// errorSessionStore always returns an error on Load, simulating a DynamoDB outage.
type errorSessionStore struct{}

func (e *errorSessionStore) Load(_ context.Context, sessionID string) (*models.Session, error) {
	return nil, &mockLoadError{sessionID: sessionID}
}

func (e *errorSessionStore) Save(_ context.Context, _ *models.Session) error {
	return nil // Save succeeds (no-op)
}

type mockLoadError struct{ sessionID string }

func (m *mockLoadError) Error() string { return "simulated DynamoDB failure for " + m.sessionID }

func TestHandleLexRequest_SessionLoadErrorDoesNotFailTurn(t *testing.T) {
	setupMocks(t) // sets up agentClient, personaLoader, agentConfig
	sessStore = &errorSessionStore{}

	ctx := context.Background()
	resp, err := handleLexRequest(ctx, makeLexEvent("sess-err", "my headset crackles"))
	if err != nil {
		t.Fatalf("expected no error despite Load failure, got: %v", err)
	}
	// We should get a valid response. ("my headset crackles" classifies as
	// distorted_audio, so B-05 triage handles the turn on a fresh session —
	// the load failure degrades gracefully to an empty session either way.)
	if len(resp.Messages) == 0 {
		t.Fatal("expected at least one message in response")
	}
	if strings.TrimSpace(resp.Messages[0].Content) == "" {
		t.Error("expected a non-empty response message")
	}
}

// ---------------------------------------------------------------------------
// B-05: scripted multi-turn triage flow (mock store + mock agent + real engine)
// ---------------------------------------------------------------------------

// TestHandleLexRequest_TriageMicFlowResolves walks the mic symptom end to end:
// symptom classified → pre-flight presented → six "no" answers hand off to
// Tree 2 → two failed fix steps increment failed_steps and record
// attempted_steps → a diagnostic "yes" branches → "yes that fixed it" resolves
// and Closes the dialog.
func TestHandleLexRequest_TriageMicFlowResolves(t *testing.T) {
	store := setupMocks(t)
	ctx := context.Background()
	sid := "sess-triage-mic"

	turns := []struct {
		transcript  string
		wantTree    string
		wantStep    string
		wantFailed  int
		wantClose   bool
		wantContent string // substring expected in the spoken message
	}{
		// Turn 1: classify mic_not_working, start pre-flight.
		{"they can't hear me on calls", "preflight", "preflight.s1", 0, false, "USB port"},
		// Turns 2-6: pre-flight items fail one by one (no fix-step counting).
		{"no, that didn't help", "preflight", "preflight.s2", 0, false, "selected device"},
		{"no, that didn't help", "preflight", "preflight.s3", 0, false, "hardware mute"},
		{"no, that didn't help", "preflight", "preflight.s4", 0, false, "volume"},
		{"no, that didn't help", "preflight", "preflight.s5", 0, false, "one app"},
		{"no, that didn't help", "preflight", "preflight.s6", 0, false, "restart"},
		// Turn 7: pre-flight exhausted → $symptom routes into Tree 2.
		{"no, that didn't help", "tree2", "tree2.s1", 0, false, "hardware mute"},
		// Turn 8: tree2.s1 is a fix step → failed_steps=1, advance to s2.
		{"no it didn't work", "tree2", "tree2.s2", 1, false, "microphone"},
		// Turn 9: tree2.s2 fails too → failed_steps=2, advance to s3.
		{"nope, no change", "tree2", "tree2.s3", 2, false, "level bar"},
		// Turn 10: diagnostic step — "yes the bar moves" goes to the
		// softphone/permissions path (no failed_steps increment).
		{"yes the bar moves", "tree2", "tree2.s4", 2, false, "microphone permission"},
		// Turn 11: the fix worked → resolved terminal, Close.
		{"yes that fixed it", "tree2", "tree2.s4", 2, true, "all set"},
	}

	for i, tc := range turns {
		turnNum := i + 1
		resp, err := handleLexRequest(ctx, makeLexEvent(sid, tc.transcript))
		if err != nil {
			t.Fatalf("turn %d: unexpected error: %v", turnNum, err)
		}
		if len(resp.Messages) == 0 {
			t.Fatalf("turn %d: expected a message", turnNum)
		}
		if !strings.Contains(resp.Messages[0].Content, tc.wantContent) {
			t.Errorf("turn %d: message %q does not contain %q",
				turnNum, resp.Messages[0].Content, tc.wantContent)
		}
		wantDialog := "ElicitIntent"
		if tc.wantClose {
			wantDialog = "Close"
		}
		if resp.SessionState.DialogAction.Type != wantDialog {
			t.Errorf("turn %d: dialog action %q, want %q",
				turnNum, resp.SessionState.DialogAction.Type, wantDialog)
		}

		stored, loadErr := store.Load(ctx, sid)
		if loadErr != nil {
			t.Fatalf("turn %d: load failed: %v", turnNum, loadErr)
		}
		if got := session.GetCurrentTree(stored); got != tc.wantTree {
			t.Errorf("turn %d: current_tree=%q, want %q", turnNum, got, tc.wantTree)
		}
		if got := session.GetCurrentStep(stored); got != tc.wantStep {
			t.Errorf("turn %d: current_step=%q, want %q", turnNum, got, tc.wantStep)
		}
		if got := session.GetFailedSteps(stored); got != tc.wantFailed {
			t.Errorf("turn %d: failed_steps=%d, want %d", turnNum, got, tc.wantFailed)
		}
	}

	// Final state: symptom recorded, resolution flagged with disposition, and
	// attempted_steps carries the full trail (6 pre-flight + tree2 s1/s2/s3/s4).
	stored, _ := store.Load(ctx, sid)
	if got := stored.Attributes["symptom"]; got != "mic_not_working" {
		t.Errorf("symptom=%q, want mic_not_working", got)
	}
	if stored.Attributes["resolved"] != "true" {
		t.Errorf("resolved=%q, want true", stored.Attributes["resolved"])
	}
	if got := stored.Attributes["disposition"]; got != "contained_resolved" {
		t.Errorf("disposition=%q, want contained_resolved", got)
	}
	attempted := session.GetAttemptedSteps(stored)
	if len(attempted) != 10 {
		t.Errorf("attempted_steps has %d entries, want 10: %v", len(attempted), attempted)
	}
	for _, want := range []string{"preflight.s1", "preflight.s6", "tree2.s1", "tree2.s2", "tree2.s4"} {
		found := false
		for _, s := range attempted {
			if s == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("attempted_steps missing %q: %v", want, attempted)
		}
	}
}

// TestHandleLexRequest_TriageRMATerminalEscalates drives Tree 3 to its RMA
// terminal and verifies the engine terminal flows through the
// BuildEscalationResponse path: escalation_requested/reason/priority for the
// Connect transfer (OPS-2), plus triage_tree/triage_step handoff context.
func TestHandleLexRequest_TriageRMATerminalEscalates(t *testing.T) {
	store := setupMocks(t)
	ctx := context.Background()
	sid := "sess-triage-rma"

	transcripts := []string{
		"my headset is not detected", // classify not_detected → preflight.s1
		"no", "no", "no", "no", "no", // preflight s1→s6
		"no", // preflight.s6 fails → routes into tree3.s1
		"no", // tree3.s1 → s2
		"no", // tree3.s2 → s3
		"no", // tree3.s3 "works on another machine?" NO → RMA terminal
	}

	var resp handlers.LexV2Response
	var err error
	for i, tr := range transcripts {
		resp, err = handleLexRequest(ctx, makeLexEvent(sid, tr))
		if err != nil {
			t.Fatalf("turn %d: unexpected error: %v", i+1, err)
		}
	}

	sa := resp.SessionState.SessionAttributes
	if sa["escalation_requested"] != "true" {
		t.Errorf("escalation_requested=%q, want true", sa["escalation_requested"])
	}
	if sa["escalation_reason"] != "hardware_fault" {
		t.Errorf("escalation_reason=%q, want hardware_fault", sa["escalation_reason"])
	}
	if sa["escalation_priority"] != "high" {
		t.Errorf("escalation_priority=%q, want high", sa["escalation_priority"])
	}
	if sa["triage_tree"] != "tree3" {
		t.Errorf("triage_tree=%q, want tree3", sa["triage_tree"])
	}
	if sa["triage_step"] != "tree3.s3" {
		t.Errorf("triage_step=%q, want tree3.s3", sa["triage_step"])
	}
	if sa["attempted_steps"] == "" {
		t.Error("attempted_steps should be carried in the escalation attributes")
	}
	if resp.SessionState.DialogAction.Type != "Close" {
		t.Errorf("dialog action %q, want Close", resp.SessionState.DialogAction.Type)
	}
	stored, _ := store.Load(ctx, sid)
	if stored.Attributes["escalated"] != "true" {
		t.Errorf("persisted escalated=%q, want true", stored.Attributes["escalated"])
	}
	if stored.Attributes["escalation_reason"] != "hardware_fault" {
		t.Errorf("persisted escalation_reason=%q, want hardware_fault", stored.Attributes["escalation_reason"])
	}
}

// TestHandleLexRequest_TriageOffScriptRepromptsOnceThenNo verifies the unclear
// loop: an off-script reply re-presents the same step once; a second
// off-script reply is treated as didnt_work and the flow advances.
func TestHandleLexRequest_TriageOffScriptRepromptsOnceThenNo(t *testing.T) {
	store := setupMocks(t)
	ctx := context.Background()
	sid := "sess-triage-unclear"

	// Turn 1: start the flow.
	if _, err := handleLexRequest(ctx, makeLexEvent(sid, "i can't hear anything in my headset")); err != nil {
		t.Fatalf("turn 1: %v", err)
	}
	stored, _ := store.Load(ctx, sid)
	if got := session.GetCurrentStep(stored); got != "preflight.s1" {
		t.Fatalf("turn 1: current_step=%q, want preflight.s1", got)
	}

	// Turn 2: off-script → re-prompt, same step, streak=1.
	resp, err := handleLexRequest(ctx, makeLexEvent(sid, "banana sandwich"))
	if err != nil {
		t.Fatalf("turn 2: %v", err)
	}
	if !strings.Contains(resp.Messages[0].Content, "yes or no") {
		t.Errorf("turn 2: expected a re-prompt, got %q", resp.Messages[0].Content)
	}
	stored, _ = store.Load(ctx, sid)
	if got := session.GetCurrentStep(stored); got != "preflight.s1" {
		t.Errorf("turn 2: current_step=%q, want preflight.s1 (re-prompt)", got)
	}
	if got := session.GetUnclearStreak(stored); got != 1 {
		t.Errorf("turn 2: unclear_streak=%d, want 1", got)
	}

	// Turn 3: off-script again → treated as didnt_work → advances to s2.
	if _, err := handleLexRequest(ctx, makeLexEvent(sid, "purple monkey dishwasher")); err != nil {
		t.Fatalf("turn 3: %v", err)
	}
	stored, _ = store.Load(ctx, sid)
	if got := session.GetCurrentStep(stored); got != "preflight.s2" {
		t.Errorf("turn 3: current_step=%q, want preflight.s2 (advance after 2nd unclear)", got)
	}
	if got := session.GetUnclearStreak(stored); got != 0 {
		t.Errorf("turn 3: unclear_streak=%d, want 0", got)
	}
	attempted := session.GetAttemptedSteps(stored)
	if len(attempted) != 1 || attempted[0] != "preflight.s1" {
		t.Errorf("turn 3: attempted_steps=%v, want [preflight.s1]", attempted)
	}
}

// TestHandleLexRequest_TriageFreeFormQuestionUsesBedrock verifies a side
// question routes to the Bedrock agent and the engine re-presents the same
// step without moving counters (FreeFormHandled).
func TestHandleLexRequest_TriageFreeFormQuestionUsesBedrock(t *testing.T) {
	store := setupMocks(t)
	ctx := context.Background()
	sid := "sess-triage-freeform"

	if _, err := handleLexRequest(ctx, makeLexEvent(sid, "there's no sound in my headset")); err != nil {
		t.Fatalf("turn 1: %v", err)
	}

	resp, err := handleLexRequest(ctx, makeLexEvent(sid, "what's a usb hub?"))
	if err != nil {
		t.Fatalf("turn 2: %v", err)
	}
	if !strings.Contains(resp.Messages[0].Content, "Stub bedrock response") {
		t.Errorf("expected Bedrock answer in message, got %q", resp.Messages[0].Content)
	}

	stored, _ := store.Load(ctx, sid)
	if got := session.GetCurrentStep(stored); got != "preflight.s1" {
		t.Errorf("current_step=%q, want preflight.s1 (free-form must not advance)", got)
	}
	if got := session.GetUnclearStreak(stored); got != 0 {
		t.Errorf("unclear_streak=%d, want 0 (free-form moves no counters)", got)
	}
}

// ---------------------------------------------------------------------------
// A-08: /chat answers from the knowledge base via RetrieveAndGenerate
// ---------------------------------------------------------------------------

// makeChatRequest builds a minimal API Gateway /chat POST request.
func makeChatRequest(sessionID, transcript string) events.APIGatewayV2HTTPRequest {
	body, _ := json.Marshal(ChatRequest{
		SessionID:       sessionID,
		InputTranscript: transcript,
	})
	req := events.APIGatewayV2HTTPRequest{
		RawPath: "/chat",
		Body:    string(body),
	}
	req.RequestContext.HTTP.Method = "POST"
	return req
}

// chatMessages parses the ChatResponse body of a handleAPIRequest result.
func chatMessages(t *testing.T, resp events.APIGatewayV2HTTPResponse) []ChatMessage {
	t.Helper()
	var chatResp ChatResponse
	if err := json.Unmarshal([]byte(resp.Body), &chatResp); err != nil {
		t.Fatalf("failed to parse chat response body %q: %v", resp.Body, err)
	}
	return chatResp.Messages
}

// TestHandleAPIRequest_ChatReturnsKBGroundedAnswer is the A-08 acceptance
// test: with KB_ID set, /chat answers from the knowledge base via
// RetrieveAndGenerate — NOT the "trouble connecting" fallback and NOT the
// supervisor agent.
func TestHandleAPIRequest_ChatReturnsKBGroundedAnswer(t *testing.T) {
	setupMocks(t)
	t.Setenv("KB_ID", "KBTEST123")
	t.Setenv("BEDROCK_MODEL_SUPERVISOR", "us.anthropic.claude-3-5-sonnet-20241022-v2:0")

	mock := &mockAgentInvoker{ragText: "First, check the headset is plugged into a direct USB port — that fixes most no-sound issues."}
	agentClient = mock

	resp, err := handleAPIRequest(context.Background(), makeChatRequest("sess-chat-kb", "my headset has no sound"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d, want 200", resp.StatusCode)
	}

	msgs := chatMessages(t, resp)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if !strings.Contains(msgs[0].Content, "direct USB port") {
		t.Errorf("expected the KB-grounded answer, got %q", msgs[0].Content)
	}
	if strings.Contains(msgs[0].Content, "trouble connecting") {
		t.Errorf("got the fallback message instead of a KB answer: %q", msgs[0].Content)
	}

	// The RAG request must carry the KB id, model, query, and persona.
	if mock.lastRAGRequest == nil {
		t.Fatal("RetrieveAndGenerate was not called")
	}
	if mock.lastRAGRequest.KnowledgeBaseID != "KBTEST123" {
		t.Errorf("kb id=%q, want KBTEST123", mock.lastRAGRequest.KnowledgeBaseID)
	}
	if mock.lastRAGRequest.ModelID != "us.anthropic.claude-3-5-sonnet-20241022-v2:0" {
		t.Errorf("model=%q", mock.lastRAGRequest.ModelID)
	}
	if mock.lastRAGRequest.Query != "my headset has no sound" {
		t.Errorf("query=%q", mock.lastRAGRequest.Query)
	}
	if mock.lastRAGRequest.Persona == nil || mock.lastRAGRequest.Persona.DisplayName != "Tangerine Test" {
		t.Error("expected the loaded persona to be passed to RetrieveAndGenerate")
	}
}

// TestHandleAPIRequest_ChatFallsBackGracefullyOnRAGError verifies the
// guardrail: a RetrieveAndGenerate failure returns the graceful fallback
// message with HTTP 200 (never a 5xx to the website).
func TestHandleAPIRequest_ChatFallsBackGracefullyOnRAGError(t *testing.T) {
	setupMocks(t)
	t.Setenv("KB_ID", "KBTEST123")
	agentClient = &mockAgentInvoker{ragErr: errors.New("kb exploded")}

	resp, err := handleAPIRequest(context.Background(), makeChatRequest("sess-chat-kb-err", "no sound"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d, want 200", resp.StatusCode)
	}
	msgs := chatMessages(t, resp)
	if len(msgs) != 1 || !strings.Contains(msgs[0].Content, "trouble connecting") {
		t.Errorf("expected graceful fallback message, got %v", msgs)
	}
}

// TestHandleAPIRequest_ChatPaymentGuardrailStillFires verifies the payment
// solicitation refusal runs BEFORE any KB retrieval.
func TestHandleAPIRequest_ChatPaymentGuardrailStillFires(t *testing.T) {
	setupMocks(t)
	t.Setenv("KB_ID", "KBTEST123")
	mock := &mockAgentInvoker{}
	agentClient = mock

	resp, err := handleAPIRequest(context.Background(),
		makeChatRequest("sess-chat-pay", "my card number is 4111 1111 1111 1111"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := chatMessages(t, resp)
	if len(msgs) != 1 || !strings.Contains(msgs[0].Content, "can't take any payment") {
		t.Errorf("expected payment refusal, got %v", msgs)
	}
	if mock.lastRAGRequest != nil {
		t.Error("RetrieveAndGenerate must not be called on the payment-refusal path")
	}
}

// TestHandleAPIRequest_ChatUsesAgentWhenKBUnset verifies InvokeAgent remains
// the fallback path only when KB_ID is empty.
func TestHandleAPIRequest_ChatUsesAgentWhenKBUnset(t *testing.T) {
	setupMocks(t) // KB_ID set to "" by setupMocks
	mock := &mockAgentInvoker{}
	agentClient = mock

	resp, err := handleAPIRequest(context.Background(), makeChatRequest("sess-chat-agent", "hello there"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := chatMessages(t, resp)
	if len(msgs) != 1 || !strings.Contains(msgs[0].Content, "Stub bedrock response") {
		t.Errorf("expected supervisor-agent answer when KB_ID is unset, got %v", msgs)
	}
	if mock.lastRAGRequest != nil {
		t.Error("RetrieveAndGenerate must not be called when KB_ID is empty")
	}
}

// TestHandleLexRequest_TriageFreeFormUsesKBWhenConfigured verifies the triage
// free-form hook answers from the knowledge base when KB_ID is set, and that
// the step's KB doc is passed as retrieval context.
func TestHandleLexRequest_TriageFreeFormUsesKBWhenConfigured(t *testing.T) {
	store := setupMocks(t)
	t.Setenv("KB_ID", "KBTEST123")
	mock := &mockAgentInvoker{ragText: "A USB hub is a splitter that adds extra USB ports."}
	agentClient = mock

	ctx := context.Background()
	sid := "sess-triage-freeform-kb"

	if _, err := handleLexRequest(ctx, makeLexEvent(sid, "there's no sound in my headset")); err != nil {
		t.Fatalf("turn 1: %v", err)
	}

	resp, err := handleLexRequest(ctx, makeLexEvent(sid, "what's a usb hub?"))
	if err != nil {
		t.Fatalf("turn 2: %v", err)
	}
	if !strings.Contains(resp.Messages[0].Content, "A USB hub is a splitter") {
		t.Errorf("expected KB-grounded free-form answer, got %q", resp.Messages[0].Content)
	}
	if mock.lastRAGRequest == nil {
		t.Fatal("RetrieveAndGenerate was not called for the free-form question")
	}
	if !strings.Contains(mock.lastRAGRequest.Query, "preflight-checklist") {
		t.Errorf("expected the step's KB doc in the retrieval query, got %q", mock.lastRAGRequest.Query)
	}

	// Free-form must not advance the flow.
	stored, _ := store.Load(ctx, sid)
	if got := session.GetCurrentStep(stored); got != "preflight.s1" {
		t.Errorf("current_step=%q, want preflight.s1 (free-form must not advance)", got)
	}
}

// ---------------------------------------------------------------------------
// B-07: Lex slots → triage classification + KB metadata filters
// ---------------------------------------------------------------------------

// makeLexEventWithSlots builds a Lex event whose current intent carries the
// given slots, shaped exactly as Lex V2 delivers them (value.resolvedValues +
// interpretedValue). The slot names are the bot's slot names (issue_type,
// connection_type, headset_brand).
func makeLexEventWithSlots(sessionID, transcript string, slots map[string]string) LexV2Event {
	ev := makeLexEvent(sessionID, transcript)
	slotMap := make(map[string]interface{}, len(slots))
	for name, val := range slots {
		slotMap[name] = map[string]interface{}{
			"value": map[string]interface{}{
				"originalValue":    val,
				"interpretedValue": val,
				"resolvedValues":   []interface{}{val},
			},
		}
	}
	ev.SessionState.Intent = &IntentResult{Name: "TroubleshootIntent", Slots: slotMap}
	return ev
}

func TestResolvedSlots_ReadsAndNormalizes(t *testing.T) {
	ev := makeLexEventWithSlots("s", "my jabra usb mic is dead", map[string]string{
		"issue_type":      "mic_not_working",
		"connection_type": "USB", // upper-cased; normalizer lowercases
		"headset_brand":   "jabra",
	})
	got := ev.resolvedSlots()
	want := triage.Slots{ConnectionType: "usb", Brand: "jabra", IssueType: "mic_not_working"}
	if got != want {
		t.Errorf("resolvedSlots()=%+v, want %+v", got, want)
	}
}

func TestResolvedSlots_DropsOutOfVocabValues(t *testing.T) {
	// A brand/connection outside the frozen vocabulary is dropped (never reaches
	// the retrieval filter); issue_type is passed through for the classifier.
	ev := makeLexEventWithSlots("s", "x", map[string]string{
		"connection_type": "carrier_pigeon",
		"headset_brand":   "acme",
		"issue_type":      "no_audio_output",
	})
	got := ev.resolvedSlots()
	if got.ConnectionType != "" {
		t.Errorf("ConnectionType=%q, want empty (out of vocab)", got.ConnectionType)
	}
	if got.Brand != "" {
		t.Errorf("Brand=%q, want empty (out of vocab)", got.Brand)
	}
	if got.IssueType != "no_audio_output" {
		t.Errorf("IssueType=%q, want no_audio_output", got.IssueType)
	}
}

func TestResolvedSlots_FallsBackToInterpretations(t *testing.T) {
	ev := makeLexEvent("s", "x")
	ev.Interpretations = []Interpretation{{
		Intent: IntentResult{Name: "TroubleshootIntent", Slots: map[string]interface{}{
			"connection_type": map[string]interface{}{
				"value": map[string]interface{}{"resolvedValues": []interface{}{"bluetooth"}},
			},
		}},
	}}
	if got := ev.resolvedSlots(); got.ConnectionType != "bluetooth" {
		t.Errorf("ConnectionType=%q, want bluetooth (from interpretations)", got.ConnectionType)
	}
}

func TestMergeSlots_PersistsAndDoesNotClobber(t *testing.T) {
	sess := &models.Session{SessionID: "s", Attributes: map[string]string{}}
	attrs := map[string]string{}

	// Turn 1: connection captured.
	eff := mergeSlots(sess, attrs, triage.Slots{ConnectionType: "usb"})
	if eff.ConnectionType != "usb" {
		t.Fatalf("turn1 effective connection=%q, want usb", eff.ConnectionType)
	}
	if attrs[session.KeyConnectionType] != "usb" {
		t.Errorf("turn1 mirror attr=%q, want usb", attrs[session.KeyConnectionType])
	}

	// Turn 2: brand captured, connection NOT restated — must not be clobbered.
	eff = mergeSlots(sess, attrs, triage.Slots{Brand: "poly"})
	if eff.ConnectionType != "usb" {
		t.Errorf("turn2 effective connection=%q, want usb (sticky)", eff.ConnectionType)
	}
	if eff.Brand != "poly" {
		t.Errorf("turn2 effective brand=%q, want poly", eff.Brand)
	}
}

func TestKBFilters_AnyOfWithAny(t *testing.T) {
	sess := &models.Session{SessionID: "s", Attributes: map[string]string{}}
	if f := kbFilters(sess); f != nil {
		t.Errorf("no facts → nil filters, got %v", f)
	}
	session.SetString(sess, session.KeyConnectionType, "usb")
	session.SetString(sess, session.KeyBrand, "jabra")
	f := kbFilters(sess)
	if !reflect.DeepEqual(f["connection_type"], []string{"any", "usb"}) {
		t.Errorf("connection_type filter=%v, want [any usb]", f["connection_type"])
	}
	if !reflect.DeepEqual(f["brand"], []string{"any", "jabra"}) {
		t.Errorf("brand filter=%v, want [any jabra]", f["brand"])
	}
}

// TestHandleLexRequest_IssueTypeSlotResolvesSymptom proves the issue_type slot
// drives deterministic classification even when the free text has no symptom
// keywords ("please help me" matches nothing).
func TestHandleLexRequest_IssueTypeSlotResolvesSymptom(t *testing.T) {
	store := setupMocks(t)
	ctx := context.Background()
	sid := "sess-issue-slot"

	ev := makeLexEventWithSlots(sid, "please help me", map[string]string{
		"issue_type": "not_detected",
	})
	if _, err := handleLexRequest(ctx, ev); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored, _ := store.Load(ctx, sid)
	if got := session.GetString(stored, session.KeySymptom); got != "not_detected" {
		t.Errorf("symptom=%q, want not_detected (resolved from issue_type slot)", got)
	}
	if got := session.GetCurrentTree(stored); got != "preflight" {
		t.Errorf("current_tree=%q, want preflight", got)
	}
}

// TestHandleLexRequest_SlotsFlowIntoRetrievalFilters proves connection_type and
// brand slots are persisted and passed as IN-with-"any" filters on a KB turn.
func TestHandleLexRequest_SlotsFlowIntoRetrievalFilters(t *testing.T) {
	store := setupMocks(t)
	mock := &mockAgentInvoker{ragText: "Some general guidance."}
	agentClient = mock
	t.Setenv("KB_ID", "kb-123")
	ctx := context.Background()
	sid := "sess-slot-filter"

	// "tell me about my headset" classifies to nothing → generic KB turn fires.
	ev := makeLexEventWithSlots(sid, "tell me about my headset", map[string]string{
		"connection_type": "usb",
		"headset_brand":   "jabra",
	})
	if _, err := handleLexRequest(ctx, ev); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.lastRAGRequest == nil {
		t.Fatal("expected a RetrieveAndGenerate call on the generic KB turn")
	}
	f := mock.lastRAGRequest.FilterAnyOf
	if !reflect.DeepEqual(f["connection_type"], []string{"any", "usb"}) {
		t.Errorf("connection_type filter=%v, want [any usb]", f["connection_type"])
	}
	if !reflect.DeepEqual(f["brand"], []string{"any", "jabra"}) {
		t.Errorf("brand filter=%v, want [any jabra]", f["brand"])
	}
	// Persisted for later turns.
	stored, _ := store.Load(ctx, sid)
	if got := session.GetString(stored, session.KeyConnectionType); got != "usb" {
		t.Errorf("persisted connection_type=%q, want usb", got)
	}
}
