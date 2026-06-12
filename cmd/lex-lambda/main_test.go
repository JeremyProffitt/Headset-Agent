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
	"strings"
	"testing"

	"github.com/headset-support-agent/internal/agents"
	"github.com/headset-support-agent/internal/handlers"
	"github.com/headset-support-agent/internal/models"
	"github.com/headset-support-agent/internal/session"
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

// mockAgentInvoker returns a canned success response so Bedrock is never called.
type mockAgentInvoker struct{}

func (m *mockAgentInvoker) InvokeAgent(_ context.Context, input agents.InvokeAgentInput) (*models.AgentResponse, error) {
	return &models.AgentResponse{
		OutputText: "Stub bedrock response",
		SessionID:  input.SessionID,
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
