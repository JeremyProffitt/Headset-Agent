package handlers

import (
	"context"
	"strings"
	"testing"

	"github.com/headset-support-agent/internal/models"
	"github.com/headset-support-agent/internal/session"
	"github.com/headset-support-agent/internal/triage"
)

// ---------------------------------------------------------------------------
// Outcome parsing
// ---------------------------------------------------------------------------

func TestParseOutcome(t *testing.T) {
	cases := []struct {
		transcript string
		want       triage.Outcome
	}{
		// worked
		{"yes", triage.OutcomeWorked},
		{"Yes!", triage.OutcomeWorked},
		{"yeah that worked", triage.OutcomeWorked},
		{"yep, all good now", triage.OutcomeWorked},
		{"that did it", triage.OutcomeWorked},
		{"it's fixed", triage.OutcomeWorked},
		{"yes that fixed it", triage.OutcomeWorked},
		{"they can hear me now", triage.OutcomeWorked},
		{"oh wow it works now", triage.OutcomeWorked},
		{"problem solved, thanks", triage.OutcomeWorked},
		{"yes the bar moves", triage.OutcomeWorked},
		{"I already fixed it", triage.OutcomeWorked},

		// didnt_work
		{"no", triage.OutcomeDidntWork},
		{"nope", triage.OutcomeDidntWork},
		{"nah", triage.OutcomeDidntWork},
		{"no, it didn't work", triage.OutcomeDidntWork},
		{"that didn't help", triage.OutcomeDidntWork},
		{"still not working", triage.OutcomeDidntWork},
		{"it doesn't work", triage.OutcomeDidntWork},
		{"same problem as before", triage.OutcomeDidntWork},
		{"no luck", triage.OutcomeDidntWork},
		{"still silent", triage.OutcomeDidntWork},
		{"nothing changed", triage.OutcomeDidntWork},
		{"it was already set to the headset", triage.OutcomeDidntWork},
		{"no sound at all", triage.OutcomeDidntWork},
		{"it's still broken", triage.OutcomeDidntWork},

		// unclear
		{"", triage.OutcomeUnclear},
		{"banana sandwich", triage.OutcomeUnclear},
		{"maybe", triage.OutcomeUnclear},
		{"hold on let me check", triage.OutcomeUnclear},
		{"what's a usb hub?", triage.OutcomeUnclear},
		{"well yes and no", triage.OutcomeUnclear}, // conflicting verdicts
		{"it sounds robotic", triage.OutcomeUnclear},
	}
	for _, tc := range cases {
		if got := ParseOutcome(tc.transcript); got != tc.want {
			t.Errorf("ParseOutcome(%q) = %q, want %q", tc.transcript, got, tc.want)
		}
	}
}

func TestIsRepeatRequest(t *testing.T) {
	if !isRepeatRequest("can you say that again please") {
		t.Error("expected repeat request")
	}
	if !isRepeatRequest("sorry, i didn't catch that") {
		t.Error("expected repeat request")
	}
	// Echo characterizations must NOT trigger a replay.
	if isRepeatRequest("they hear a repeat of my voice") {
		t.Error("echo description must not be a repeat request")
	}
}

func TestIsFreeFormQuestion(t *testing.T) {
	for _, q := range []string{"what's a usb hub?", "how do i open device manager", "where is the volume mixer"} {
		if !isFreeFormQuestion(q) {
			t.Errorf("expected %q to be a free-form question", q)
		}
	}
	for _, s := range []string{"no", "the other person hears it", "banana sandwich"} {
		if isFreeFormQuestion(s) {
			t.Errorf("did not expect %q to be a free-form question", s)
		}
	}
}

// ---------------------------------------------------------------------------
// Prompt catalog completeness — every reachable PromptRef has spoken copy
// ---------------------------------------------------------------------------

func TestPromptCatalogCoversAllTreeKeys(t *testing.T) {
	fork := triage.BuildSymptomFork()
	trees := []*triage.Tree{fork.PreFlight}
	for _, tr := range fork.Trees {
		trees = append(trees, tr)
	}

	var keys []triage.PromptRef
	addTerminal := func(tr triage.Transition) {
		if tr.Terminal != nil && tr.Terminal.ReadAloudKey != "" {
			keys = append(keys, tr.Terminal.ReadAloudKey)
		}
	}
	for _, tree := range trees {
		for _, step := range tree.Steps {
			keys = append(keys, step.ReadAloudKey)
			addTerminal(step.OnWorked)
			addTerminal(step.OnDidntWork)
			addTerminal(step.OnUnclear)
		}
	}

	// Engine-fired terminals: handoff.<reason> for every escalation reason,
	// and the $original-route resolution key fired from engine.routeToTree.
	reasons := []triage.EscalationReason{
		triage.ReasonUserRequested, triage.ReasonUserFrustrated,
		triage.ReasonTroubleshootingExhausted, triage.ReasonHardwareFault,
		triage.ReasonRebootLimit, triage.ReasonTreeExhausted,
		triage.ReasonManagedMachinePolicy, triage.ReasonUserCannotPerform,
		triage.ReasonAccessibilityNeed, triage.ReasonGenesysPlatform,
	}
	for _, r := range reasons {
		keys = append(keys, triage.EscalationTerminal(r).ReadAloudKey)
	}
	keys = append(keys, "tree3.s1.resolved")

	for _, key := range keys {
		if _, ok := PromptText(key); !ok {
			t.Errorf("prompt catalog missing copy for key %q", key)
		}
	}
}

// ---------------------------------------------------------------------------
// Direct turn-driver tests (real engine, no Lambda plumbing)
// ---------------------------------------------------------------------------

func testDeps() TriageDeps {
	return TriageDeps{
		Engine:     triage.NewDefaultEngine(),
		Classifier: triage.NewClassifier(nil),
	}
}

func testSession(id string) *models.Session {
	return &models.Session{SessionID: id, Attributes: make(map[string]string)}
}

func testPersona() *models.Persona {
	return &models.Persona{
		PersonaID:   "tangerine",
		VoiceConfig: models.VoiceConfig{Prosody: models.Prosody{Rate: "100%", Pitch: "medium"}},
	}
}

// TestHandleTriageTurn_UnhandledWhenUnclassified: an opener with no symptom
// keywords defers to the generic agent path.
func TestHandleTriageTurn_UnhandledWhenUnclassified(t *testing.T) {
	sess := testSession("s1")
	attrs := map[string]string{}
	_, handled := HandleTriageTurn(context.Background(), testDeps(), sess, "hello there", testPersona(), attrs)
	if handled {
		t.Fatal("expected unclassifiable opener to be unhandled")
	}
	if session.GetCurrentTree(sess) != "" {
		t.Errorf("flow must not start on an unclassified opener, got tree %q", session.GetCurrentTree(sess))
	}
}

// TestHandleTriageTurn_SymptomClarifierPath: pre-flight finishes with no
// classified symptom → ErrSymptomRequired → clarifier question → the next
// utterance classifies and enters the right tree.
func TestHandleTriageTurn_SymptomClarifierPath(t *testing.T) {
	deps := testDeps()
	sess := testSession("s2")
	attrs := map[string]string{}
	p := testPersona()
	ctx := context.Background()

	// Simulate a session parked on the last pre-flight item with no symptom
	// (e.g. started via slots in B-07, or the classifier was unavailable).
	session.SetCurrentTree(sess, "preflight")
	session.SetCurrentStep(sess, "preflight.s6")

	resp, handled := HandleTriageTurn(ctx, deps, sess, "no, that didn't help", p, attrs)
	if !handled {
		t.Fatal("expected the clarifier turn to be handled")
	}
	if !strings.Contains(resp.Messages[0].Content, "Which best describes it") {
		t.Errorf("expected the symptom clarifier, got %q", resp.Messages[0].Content)
	}
	if sess.Attributes[KeyAwaitingSymptom] != "true" {
		t.Errorf("awaiting_symptom=%q, want true", sess.Attributes[KeyAwaitingSymptom])
	}

	// The next utterance answers the clarifier and selects Tree 6.
	resp, handled = HandleTriageTurn(ctx, deps, sess, "it's about hearing my own voice, the sidetone", p, attrs)
	if !handled {
		t.Fatal("expected the symptom answer to be handled")
	}
	if got := session.GetCurrentTree(sess); got != "tree6" {
		t.Errorf("current_tree=%q, want tree6", got)
	}
	if sess.Attributes[KeyAwaitingSymptom] != "false" {
		t.Errorf("awaiting_symptom=%q, want false", sess.Attributes[KeyAwaitingSymptom])
	}
	if len(resp.Messages) == 0 || resp.Messages[0].Content == "" {
		t.Error("expected the tree entry step to be spoken")
	}
}

// TestHandleTriageTurn_BranchStepClassifiesUtterance: at the Tree 5 fork the
// raw utterance picks the branch even though the outcome parse is unclear.
func TestHandleTriageTurn_BranchStepClassifiesUtterance(t *testing.T) {
	deps := testDeps()
	sess := testSession("s3")
	attrs := map[string]string{}
	p := testPersona()
	ctx := context.Background()

	session.SetCurrentTree(sess, "tree5")
	session.SetCurrentStep(sess, "tree5.s1")
	session.SetString(sess, session.KeySymptom, string(triage.SymptomDistortedAudio))

	_, handled := HandleTriageTurn(ctx, deps, sess, "it's all crackly and full of static", p, attrs)
	if !handled {
		t.Fatal("expected branch turn to be handled")
	}
	if got := session.GetCurrentStep(sess); got != "tree5.s3" {
		t.Errorf("current_step=%q, want tree5.s3 (crackling branch)", got)
	}
}

// TestHandleTriageTurn_FlowEndedDefers: once resolved, subsequent turns fall
// through to the generic agent.
func TestHandleTriageTurn_FlowEndedDefers(t *testing.T) {
	sess := testSession("s4")
	session.SetBool(sess, session.KeyResolved, true)
	_, handled := HandleTriageTurn(context.Background(), testDeps(), sess, "thanks anyway", testPersona(), map[string]string{})
	if handled {
		t.Fatal("expected ended flow to defer to the agent")
	}
}

// TestHandleTriageTurn_RepeatReplaysStep: "say that again" replays the last
// rendered text without advancing.
func TestHandleTriageTurn_RepeatReplaysStep(t *testing.T) {
	deps := testDeps()
	sess := testSession("s5")
	attrs := map[string]string{}
	p := testPersona()
	ctx := context.Background()

	if _, handled := HandleTriageTurn(ctx, deps, sess, "my headset keeps cutting out", p, attrs); !handled {
		t.Fatal("expected flow start")
	}
	first := session.GetLastResponse(sess)

	resp, handled := HandleTriageTurn(ctx, deps, sess, "sorry, can you say that again", p, attrs)
	if !handled {
		t.Fatal("expected repeat turn to be handled")
	}
	if session.GetCurrentStep(sess) != "preflight.s1" {
		t.Errorf("repeat must not advance, got step %q", session.GetCurrentStep(sess))
	}
	if !strings.Contains(resp.Messages[0].Content, "USB port") || first == "" {
		t.Errorf("expected the step to be replayed, got %q", resp.Messages[0].Content)
	}
}
