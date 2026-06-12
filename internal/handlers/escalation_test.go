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

// TestDetectEscalation_CounterBugNeverIncrements was originally written to
// document the WS-G-01 counter bug where frustration_count was never persisted.
//
// B-06 has LANDED. The fix is in the handler (cmd/lex-lambda/main.go):
// after calling DetectEscalation the handler reads frustration_count from the
// session store, passes it in, then persists (frustration_count +
// FrustrationDelta) back before saving. The 3-turn accumulation scenario is
// now covered by handler-level tests in cmd/lex-lambda/main_test.go
// (TestHandleLexRequest_FrustrationAccumulatesAcross3Turns).
//
// This test retains its original assertion: DetectEscalation itself does NOT
// escalate when each call receives frustrationCount=0 (a single phrase gives
// delta=1 < threshold of 3). That is CORRECT — accumulation is the handler's
// responsibility, not DetectEscalation's.
func TestDetectEscalation_CounterBugNeverIncrements(t *testing.T) {
	// Simulate 3 separate turns, each with one frustration phrase.
	// If the caller always passes frustrationCount=0 (as the buggy handler did),
	// DetectEscalation correctly returns ShouldEscalate=false each time —
	// escalation requires the accumulated counter, which the handler now persists.
	frustrationPhrases := []string{
		"this is ridiculous",
		"doesn't work",
		"still not working",
	}

	for i, phrase := range frustrationPhrases {
		decision := DetectEscalation(phrase, 0 /* accumulated count, not persisted */, 0)

		// With frustrationCount=0 and one phrase (delta=1), total=1 < 3 → no escalation.
		if decision.ShouldEscalate {
			t.Errorf(
				"turn %d: DetectEscalation alone should NOT escalate (counter accumulation "+
					"is the handler's job), but got ShouldEscalate=true reason=%q",
				i+1, decision.Reason,
			)
		}
		if decision.Reason == "user_frustrated" {
			t.Errorf(
				"turn %d: should not have reason=user_frustrated without accumulated count, got %q",
				i+1, decision.Reason,
			)
		}

		// B-06: FrustrationDelta must be 1 (exactly one phrase matched).
		if decision.FrustrationDelta != 1 {
			t.Errorf(
				"turn %d: expected FrustrationDelta=1, got %d",
				i+1, decision.FrustrationDelta,
			)
		}
	}
}

// ---------------------------------------------------------------------------
// B-06: FrustrationDelta field tests
// ---------------------------------------------------------------------------

// TestDetectEscalation_FrustrationDeltaZeroForNoPhrase confirms delta=0 when
// the transcript contains no frustration phrases.
func TestDetectEscalation_FrustrationDeltaZeroForNoPhrase(t *testing.T) {
	decision := DetectEscalation("my headset seems a bit quiet", 0, 0)
	if decision.FrustrationDelta != 0 {
		t.Errorf("expected FrustrationDelta=0 for benign input, got %d", decision.FrustrationDelta)
	}
}

// TestDetectEscalation_FrustrationDeltaOneForOnePhrase confirms delta=1 for a
// single matched indicator.
func TestDetectEscalation_FrustrationDeltaOneForOnePhrase(t *testing.T) {
	decision := DetectEscalation("this is ridiculous", 0, 0)
	if decision.FrustrationDelta != 1 {
		t.Errorf("expected FrustrationDelta=1, got %d", decision.FrustrationDelta)
	}
}

// TestDetectEscalation_FrustrationDeltaTwoForTwoPhrases confirms delta=2 when
// two distinct frustration indicators appear in the transcript.
func TestDetectEscalation_FrustrationDeltaTwoForTwoPhrases(t *testing.T) {
	decision := DetectEscalation("this is ridiculous and doesn't work", 0, 0)
	if decision.FrustrationDelta != 2 {
		t.Errorf("expected FrustrationDelta=2, got %d", decision.FrustrationDelta)
	}
}

// TestDetectEscalation_FrustrationDeltaPopulatedWhenEscalating confirms that
// FrustrationDelta is set even when the escalation fires (total >= 3).
func TestDetectEscalation_FrustrationDeltaPopulatedWhenEscalating(t *testing.T) {
	// frustrationCount=2, transcript has one phrase → delta=1, total=3 → escalate.
	decision := DetectEscalation("this is ridiculous", 2, 0)
	if !decision.ShouldEscalate {
		t.Fatal("expected escalation, got none")
	}
	if decision.FrustrationDelta != 1 {
		t.Errorf("expected FrustrationDelta=1 when escalating via accumulated count, got %d",
			decision.FrustrationDelta)
	}
}

// TestDetectEscalation_FrustrationDeltaZeroOnEscapeKeyword confirms delta=0
// when escalation is triggered by an escape keyword (not frustration phrases).
func TestDetectEscalation_FrustrationDeltaZeroOnEscapeKeyword(t *testing.T) {
	decision := DetectEscalation("I want to talk to a human", 0, 0)
	if !decision.ShouldEscalate {
		t.Fatal("expected user_requested escalation")
	}
	if decision.FrustrationDelta != 0 {
		t.Errorf("expected FrustrationDelta=0 for escape-keyword escalation, got %d",
			decision.FrustrationDelta)
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
