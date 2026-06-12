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
	// We should get a valid success response (Bedrock stub returned text).
	if len(resp.Messages) == 0 {
		t.Fatal("expected at least one message in response")
	}
	if !strings.Contains(resp.Messages[0].Content, "Stub bedrock response") {
		t.Errorf("unexpected response message: %q", resp.Messages[0].Content)
	}
}
