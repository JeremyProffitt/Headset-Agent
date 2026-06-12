package handlers

import (
	"testing"

	"github.com/headset-support-agent/internal/models"
)

// ---------------------------------------------------------------------------
// DetectEscalation tests
// ---------------------------------------------------------------------------

func TestDetectEscalation_EscapeKeywordsTriggerUserRequested(t *testing.T) {
	// Every keyword in EscapeKeywords must trigger user_requested / high.
	for _, kw := range EscapeKeywords {
		t.Run(kw, func(t *testing.T) {
			decision := DetectEscalation("I want to talk to a "+kw+" please", 0, 0)
			if !decision.ShouldEscalate {
				t.Errorf("keyword %q: expected ShouldEscalate=true", kw)
			}
			if decision.Reason != "user_requested" {
				t.Errorf("keyword %q: expected reason=user_requested, got %q", kw, decision.Reason)
			}
			if decision.Priority != "high" {
				t.Errorf("keyword %q: expected priority=high, got %q", kw, decision.Priority)
			}
		})
	}
}

func TestDetectEscalation_EscapeKeywordCaseInsensitive(t *testing.T) {
	decision := DetectEscalation("I need a HUMAN right now", 0, 0)
	if !decision.ShouldEscalate || decision.Reason != "user_requested" {
		t.Errorf("expected user_requested escalation for uppercase keyword, got %+v", decision)
	}
}

// TestDetectEscalation_SingleFrustrationPhraseDoesNotEscalate confirms that a
// single frustration phrase with frustrationCount=0 does NOT trigger escalation
// (threshold requires totalFrustration >= 3).
func TestDetectEscalation_SingleFrustrationPhraseDoesNotEscalate(t *testing.T) {
	decision := DetectEscalation("this is ridiculous", 0, 0)
	if decision.ShouldEscalate {
		t.Errorf("single frustration phrase with count=0 should NOT escalate, got %+v", decision)
	}
}

// TestDetectEscalation_CounterBugNeverIncrements documents the known counter bug.
//
// BUG captured per WS-G-01; fixed by B-06 which will persist the counter.
// Update this test when B-06 lands.
//
// The Lex handler reads frustration_count from session attributes but NEVER
// persists/increments it back after each turn. As a result, three separate
// conversation turns each containing ONE frustration phrase will each call
// DetectEscalation with frustrationCount=0 and currentFrustration=1.
// totalFrustration == 1, which is < 3, so escalation on frustration alone
// NEVER triggers. This test asserts the CURRENT (buggy) behaviour.
func TestDetectEscalation_CounterBugNeverIncrements(t *testing.T) {
	// Simulate 3 separate turns, each with one frustration phrase.
	// Because the counter is never persisted, each call receives frustrationCount=0.
	frustrationPhrases := []string{
		"this is ridiculous",
		"doesn't work",
		"still not working",
	}

	for i, phrase := range frustrationPhrases {
		// Each call mimics what the Lex handler actually does:
		// it passes the stale (never-incremented) frustrationCount=0.
		decision := DetectEscalation(phrase, 0 /*never incremented*/, 0)

		// BUG: totalFrustration is always 1 (0 + 1), never reaches 3.
		if decision.ShouldEscalate {
			t.Errorf(
				"turn %d: expected NO escalation because counter is never persisted (bug), "+
					"but got ShouldEscalate=true reason=%q",
				i+1, decision.Reason,
			)
		}
		if decision.Reason == "user_frustrated" {
			t.Errorf(
				"turn %d: should not have reason=user_frustrated due to counter bug, got %q",
				i+1, decision.Reason,
			)
		}
	}
}

// TestDetectEscalation_FrustrationCountPassedInReachesThreshold shows that if
// the caller DOES pass in a pre-accumulated frustrationCount >= 3 the escalation
// fires. (This is the intended behaviour once B-06 persists the counter.)
func TestDetectEscalation_FrustrationCountPassedInReachesThreshold(t *testing.T) {
	// frustrationCount=3, currentFrustration from transcript=0 → total=3 → escalate
	decision := DetectEscalation("hello", 3, 0)
	if !decision.ShouldEscalate {
		t.Errorf("expected ShouldEscalate=true with frustrationCount=3, got %+v", decision)
	}
	if decision.Reason != "user_frustrated" {
		t.Errorf("expected reason=user_frustrated, got %q", decision.Reason)
	}
	if decision.Priority != "medium" {
		t.Errorf("expected priority=medium, got %q", decision.Priority)
	}
}

func TestDetectEscalation_FrustrationCombinedCountPlusCurrent(t *testing.T) {
	// frustrationCount=2 + 1 frustration phrase in transcript → total=3 → escalate
	decision := DetectEscalation("this is ridiculous", 2, 0)
	if !decision.ShouldEscalate {
		t.Errorf("expected escalation when combined frustration=3, got %+v", decision)
	}
	if decision.Reason != "user_frustrated" {
		t.Errorf("expected user_frustrated, got %q", decision.Reason)
	}
}

func TestDetectEscalation_TwoFrustrationPhrasesDoNotEscalate(t *testing.T) {
	// 0 + 2 = 2, still below threshold of 3
	decision := DetectEscalation("this is ridiculous and doesn't work", 0, 0)
	if decision.ShouldEscalate {
		t.Errorf("two frustration phrases with count=0 (total=2) should NOT escalate, got %+v", decision)
	}
}

func TestDetectEscalation_FailedStepsThreshold(t *testing.T) {
	tests := []struct {
		failedSteps  int
		wantEscalate bool
		wantReason   string
	}{
		{4, false, ""},
		{5, true, "troubleshooting_exhausted"},
		{10, true, "troubleshooting_exhausted"},
	}

	for _, tc := range tests {
		decision := DetectEscalation("nothing is working", 0, tc.failedSteps)
		if decision.ShouldEscalate != tc.wantEscalate {
			t.Errorf("failedSteps=%d: expected ShouldEscalate=%v, got %v",
				tc.failedSteps, tc.wantEscalate, decision.ShouldEscalate)
		}
		if tc.wantEscalate && decision.Reason != tc.wantReason {
			t.Errorf("failedSteps=%d: expected reason=%q, got %q",
				tc.failedSteps, tc.wantReason, decision.Reason)
		}
		if tc.wantEscalate && decision.Priority != "medium" {
			t.Errorf("failedSteps=%d: expected priority=medium, got %q",
				tc.failedSteps, decision.Priority)
		}
	}
}

func TestDetectEscalation_NoTrigger(t *testing.T) {
	decision := DetectEscalation("my headset seems a bit quiet", 0, 0)
	if decision.ShouldEscalate {
		t.Errorf("expected no escalation for benign input, got %+v", decision)
	}
	if decision.Reason != "" || decision.Priority != "" {
		t.Errorf("expected empty reason/priority, got reason=%q priority=%q",
			decision.Reason, decision.Priority)
	}
}

// ---------------------------------------------------------------------------
// BuildEscalationResponse tests
// ---------------------------------------------------------------------------

func makeDecision(reason, priority string) *models.EscalationDecision {
	return &models.EscalationDecision{
		ShouldEscalate: true,
		Reason:         reason,
		Priority:       priority,
	}
}

func TestBuildEscalationResponse_SetsSessionAttrs(t *testing.T) {
	persona := &models.Persona{PersonaID: "joseph"}
	decision := makeDecision("user_requested", "high")
	attrs := map[string]string{"existing_key": "existing_val"}

	resp := BuildEscalationResponse(persona, decision, attrs)

	sa := resp.SessionState.SessionAttributes
	if sa["escalation_requested"] != "true" {
		t.Errorf("expected escalation_requested=true, got %q", sa["escalation_requested"])
	}
	if sa["escalation_reason"] != "user_requested" {
		t.Errorf("expected escalation_reason=user_requested, got %q", sa["escalation_reason"])
	}
	if sa["escalation_priority"] != "high" {
		t.Errorf("expected escalation_priority=high, got %q", sa["escalation_priority"])
	}
	// Existing attribute must be preserved
	if sa["existing_key"] != "existing_val" {
		t.Errorf("expected existing_key=existing_val to be preserved, got %q", sa["existing_key"])
	}
}

func TestBuildEscalationResponse_NilSessionAttrs(t *testing.T) {
	persona := &models.Persona{PersonaID: "jennifer"}
	decision := makeDecision("troubleshooting_exhausted", "medium")

	// Must not panic with nil sessionAttrs
	resp := BuildEscalationResponse(persona, decision, nil)

	sa := resp.SessionState.SessionAttributes
	if sa == nil {
		t.Fatal("expected session attributes map to be initialised, got nil")
	}
	if sa["escalation_requested"] != "true" {
		t.Errorf("expected escalation_requested=true, got %q", sa["escalation_requested"])
	}
}

func TestBuildEscalationResponse_ReturnsCloseDialogAction(t *testing.T) {
	persona := &models.Persona{PersonaID: "tangerine"}
	decision := makeDecision("user_requested", "high")

	resp := BuildEscalationResponse(persona, decision, nil)

	if resp.SessionState.DialogAction.Type != "Close" {
		t.Errorf("expected DialogAction.Type=Close, got %q", resp.SessionState.DialogAction.Type)
	}
	if len(resp.Messages) == 0 {
		t.Fatal("expected at least one message")
	}
	if resp.Messages[0].ContentType != "SSML" {
		t.Errorf("expected SSML ContentType, got %q", resp.Messages[0].ContentType)
	}
}

// ---------------------------------------------------------------------------
// getEscalationMessage tests
// ---------------------------------------------------------------------------

func TestGetEscalationMessage_NilPersona(t *testing.T) {
	msg := getEscalationMessage(nil, "user_requested")
	if msg != "Let me transfer you to a specialist who can help you further." {
		t.Errorf("unexpected nil-persona message: %q", msg)
	}
}

func TestGetEscalationMessage_PersonaEscalationPhraseUsedFirst(t *testing.T) {
	persona := &models.Persona{
		PersonaID: "tangerine",
		Phrases: models.Phrases{
			Escalation: []string{"Custom phrase here!", "Second phrase"},
		},
	}
	msg := getEscalationMessage(persona, "user_requested")
	if msg != "Custom phrase here!" {
		t.Errorf("expected persona.Phrases.Escalation[0], got %q", msg)
	}
}

func TestGetEscalationMessage_TangerineReasons(t *testing.T) {
	persona := &models.Persona{PersonaID: "tangerine"}
	tests := []struct {
		reason      string
		wantContain string
	}{
		{"user_requested", "brilliant specialists"},
		{"user_frustrated", "sort this out properly"},
		{"troubleshooting_exhausted", "tricks up their sleeve"},
	}
	for _, tc := range tests {
		msg := getEscalationMessage(persona, tc.reason)
		if msg == "" {
			t.Errorf("tangerine reason=%q returned empty message", tc.reason)
		}
		// Spot-check content contains persona-flavoured text
		if len(msg) < 10 {
			t.Errorf("tangerine reason=%q message too short: %q", tc.reason, msg)
		}
		// Verify the expected substring is present
		found := false
		for i := range msg {
			if i+len(tc.wantContain) <= len(msg) && msg[i:i+len(tc.wantContain)] == tc.wantContain {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("tangerine reason=%q: expected message to contain %q, got: %q",
				tc.reason, tc.wantContain, msg)
		}
	}
}

func TestGetEscalationMessage_JosephReasons(t *testing.T) {
	persona := &models.Persona{PersonaID: "joseph"}
	reasons := []string{"user_requested", "user_frustrated", "troubleshooting_exhausted"}
	for _, reason := range reasons {
		msg := getEscalationMessage(persona, reason)
		if msg == "" {
			t.Errorf("joseph reason=%q returned empty message", reason)
		}
	}
}

func TestGetEscalationMessage_JenniferReasons(t *testing.T) {
	persona := &models.Persona{PersonaID: "jennifer"}
	reasons := []string{"user_requested", "user_frustrated", "troubleshooting_exhausted"}
	for _, reason := range reasons {
		msg := getEscalationMessage(persona, reason)
		if msg == "" {
			t.Errorf("jennifer reason=%q returned empty message", reason)
		}
	}
}

func TestGetEscalationMessage_DefaultPersona(t *testing.T) {
	persona := &models.Persona{PersonaID: "unknown_persona"}
	msg := getEscalationMessage(persona, "user_requested")
	if msg != "Let me connect you with a specialist who can help you further." {
		t.Errorf("unexpected default-persona message: %q", msg)
	}
}

func TestGetEscalationMessage_EmptyEscalationSlice(t *testing.T) {
	// Empty slice → fall through to persona-specific switch
	persona := &models.Persona{
		PersonaID: "joseph",
		Phrases:   models.Phrases{Escalation: []string{}},
	}
	msg := getEscalationMessage(persona, "user_requested")
	if msg == "" {
		t.Error("expected non-empty message from joseph switch, got empty")
	}
}
