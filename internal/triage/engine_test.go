package triage

import (
	"errors"
	"reflect"
	"testing"
)

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	e, err := NewEngine(BuildSymptomFork())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return e
}

// enterTree starts a state directly at a symptom tree's entry (the normal
// production flow would run pre-flight first; tree walks test the trees).
func enterTree(t *testing.T, e *Engine, class SymptomClass) *TriageState {
	t.Helper()
	state := &TriageState{}
	if _, err := e.SelectSymptomTree(state, class); err != nil {
		t.Fatalf("SelectSymptomTree(%s): %v", class, err)
	}
	return state
}

func mustAdvance(t *testing.T, e *Engine, state *TriageState, outcome Outcome, utterance string) StepView {
	t.Helper()
	view, err := e.Advance(state, TurnResult{Outcome: outcome, RawUtterance: utterance})
	if err != nil {
		t.Fatalf("Advance(%s, %q) at %s/%s: %v", outcome, utterance, state.CurrentTree, state.CurrentStep, err)
	}
	return view
}

// turn is one scripted user response.
type turn struct {
	outcome   Outcome
	utterance string
}

func w(u ...string) turn { return turn{outcome: OutcomeWorked, utterance: first(u)} }
func n(u ...string) turn { return turn{outcome: OutcomeDidntWork, utterance: first(u)} }
func u(s string) turn    { return turn{outcome: OutcomeUnclear, utterance: s} }

func first(u []string) string {
	if len(u) > 0 {
		return u[0]
	}
	return ""
}

// ---------------------------------------------------------------------------
// Table-driven walks: every tree path to every terminal
// ---------------------------------------------------------------------------

func TestTreePaths(t *testing.T) {
	type pathCase struct {
		name    string
		symptom SymptomClass
		turns   []turn

		// terminal expectations (when the path ends the flow)
		wantKind        TerminalKind
		wantReason      EscalationReason
		wantDisposition string

		// non-terminal expectations (when the path lands mid-flow)
		wantTree string
		wantStep StepID
	}

	cases := []pathCase{
		// ---- Tree 1 — No Audio Output -------------------------------------
		{name: "tree1 s1 NO routes to tree3", symptom: SymptomNoAudioOutput,
			turns: []turn{n()}, wantTree: Tree3ID, wantStep: "tree3.s1"},
		{name: "tree1 s2 fixed output device", symptom: SymptomNoAudioOutput,
			turns: []turn{w(), w()}, wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree1 s3 volume/mute was the cause", symptom: SymptomNoAudioOutput,
			turns: []turn{w(), n(), w()}, wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree1 s5 softphone selection resolved", symptom: SymptomNoAudioOutput,
			turns: []turn{w(), n(), n(), w(), w()}, wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree1 s5 still silent in calls -> genesys", symptom: SymptomNoAudioOutput,
			turns: []turn{w(), n(), n(), w(), n()}, wantKind: TerminalKindEscalate, wantReason: ReasonGenesysPlatform},
		{name: "tree1 s6 windows remediation resolved", symptom: SymptomNoAudioOutput,
			turns: []turn{w(), n(), n(), n(), w()}, wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree1 s6 no sound anywhere -> tree exhausted", symptom: SymptomNoAudioOutput,
			turns: []turn{w(), n(), n(), n(), n()}, wantKind: TerminalKindEscalate, wantReason: ReasonTreeExhausted},

		// ---- Tree 2 — Mic / Other Party Can't Hear ------------------------
		{name: "tree2 s1 hardware mute", symptom: SymptomMicNotWorking,
			turns: []turn{w()}, wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree2 s2 wrong input device", symptom: SymptomMicNotWorking,
			turns: []turn{n(), w()}, wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree2 s4 softphone mic/permission fixed", symptom: SymptomMicNotWorking,
			turns: []turn{n(), n(), w(), w()}, wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree2 s4 still silent -> genesys", symptom: SymptomMicNotWorking,
			turns: []turn{n(), n(), w(), n()}, wantKind: TerminalKindEscalate, wantReason: ReasonGenesysPlatform},
		{name: "tree2 s5 windows mic privacy fixed", symptom: SymptomMicNotWorking,
			turns: []turn{n(), n(), n(), w()}, wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree2 s5 meter never moves -> RMA", symptom: SymptomMicNotWorking,
			turns: []turn{n(), n(), n(), n()}, wantKind: TerminalKindRMA, wantReason: ReasonHardwareFault},

		// ---- Tree 3 — Not Detected ----------------------------------------
		{name: "tree3 s1 appears, detection was the complaint -> resolved", symptom: SymptomNotDetected,
			turns: []turn{w()}, wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree3 s2 different port worked", symptom: SymptomNotDetected,
			turns: []turn{n(), w()}, wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree3 s3 fails everywhere -> RMA", symptom: SymptomNotDetected,
			turns: []turn{n(), n(), n()}, wantKind: TerminalKindRMA, wantReason: ReasonHardwareFault},
		{name: "tree3 s4 device manager fixed", symptom: SymptomNotDetected,
			turns: []turn{n(), n(), w(), w()}, wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree3 s5 policy suspected -> managed machine", symptom: SymptomNotDetected,
			turns: []turn{n(), n(), w(), n(), w()}, wantKind: TerminalKindEscalate, wantReason: ReasonManagedMachinePolicy},
		{name: "tree3 s5 known-good machine, no policy -> RMA", symptom: SymptomNotDetected,
			turns: []turn{n(), n(), w(), n(), n()}, wantKind: TerminalKindRMA, wantReason: ReasonHardwareFault},

		// ---- Tree 4 — One-Sided / Mono ------------------------------------
		{name: "tree4 s1 balance fixed", symptom: SymptomOneSidedAudio,
			turns: []turn{w()}, wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree4 s2 mono toggle fixed", symptom: SymptomOneSidedAudio,
			turns: []turn{n(), w()}, wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree4 s3 stereo test fine -> mono call audio expected", symptom: SymptomOneSidedAudio,
			turns: []turn{n(), n(), w()}, wantKind: TerminalKindResolved, wantDisposition: DispositionExpectationSet},
		{name: "tree4 s4 cuts with movement -> RMA cable", symptom: SymptomOneSidedAudio,
			turns: []turn{n(), n(), n(), w()}, wantKind: TerminalKindRMA, wantReason: ReasonHardwareFault},
		{name: "tree4 s4 consistently dead earcup -> RMA", symptom: SymptomOneSidedAudio,
			turns: []turn{n(), n(), n(), n()}, wantKind: TerminalKindRMA, wantReason: ReasonHardwareFault},

		// ---- Tree 5 — Distorted / Choppy / Echo ---------------------------
		{name: "tree5 robotic branch resolved by wired/CPU", symptom: SymptomDistortedAudio,
			turns:    []turn{u("it sounds robotic and choppy"), w()},
			wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree5 robotic persists on wired -> genesys", symptom: SymptomDistortedAudio,
			turns:    []turn{u("the audio keeps breaking up, very robotic"), n()},
			wantKind: TerminalKindEscalate, wantReason: ReasonGenesysPlatform},
		{name: "tree5 crackling branch resolved", symptom: SymptomDistortedAudio,
			turns:    []turn{u("there's a crackling static noise"), w()},
			wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree5 crackle persists after driver -> RMA", symptom: SymptomDistortedAudio,
			turns:    []turn{u("loud static and popping sounds"), n()},
			wantKind: TerminalKindRMA, wantReason: ReasonHardwareFault},
		{name: "tree5 echo, user hears self -> tree6 sidetone step", symptom: SymptomDistortedAudio,
			turns:    []turn{u("there's an echo on the line"), u("I hear my own voice repeated")},
			wantTree: Tree6ID, wantStep: "tree6.s3"},
		{name: "tree5 echo, other party, isolated -> resolved", symptom: SymptomDistortedAudio,
			turns:    []turn{u("I'm getting an echo"), u("the customer hears it, they hear themselves"), w()},
			wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree5 echo persists with proper headset -> genesys", symptom: SymptomDistortedAudio,
			turns:    []turn{u("echo problems"), u("the other party hears the echo"), n()},
			wantKind: TerminalKindEscalate, wantReason: ReasonGenesysPlatform},
		{name: "tree5 unmatched characterization aliases to robotic path", symptom: SymptomDistortedAudio,
			turns: []turn{n("it just sounds bad")}, wantTree: Tree5ID, wantStep: "tree5.s2"},

		// ---- Tree 6 — Volume / Sidetone -----------------------------------
		{name: "tree6 call volume fixed", symptom: SymptomVolumeSidetone,
			turns:    []turn{u("the other person is too quiet"), w()},
			wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree6 max volume still too quiet -> tree exhausted", symptom: SymptomVolumeSidetone,
			turns:    []turn{u("callers are way too quiet even at max volume"), n()},
			wantKind: TerminalKindEscalate, wantReason: ReasonTreeExhausted},
		{name: "tree6 sidetone adjusted", symptom: SymptomVolumeSidetone,
			turns:    []turn{u("I keep hearing myself talk"), w()},
			wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree6 sidetone control has no effect -> escalate", symptom: SymptomVolumeSidetone,
			turns:    []turn{u("the sidetone is way too loud, I hear my own voice"), n()},
			wantKind: TerminalKindEscalate, wantReason: ReasonGenesysPlatform},
		{name: "tree6 unmatched characterization aliases to call volume", symptom: SymptomVolumeSidetone,
			turns: []turn{w("hmm")}, wantTree: Tree6ID, wantStep: "tree6.s2"},

		// ---- Tree 7 — Mute Sync / Call Control ----------------------------
		{name: "tree7 buttons, unsupported model -> expectation set", symptom: SymptomMuteCallControl,
			turns:    []turn{u("my answer and end buttons do nothing"), n()},
			wantKind: TerminalKindResolved, wantDisposition: DispositionExpectationSet},
		{name: "tree7 buttons, vendor app fixed", symptom: SymptomMuteCallControl,
			turns:    []turn{u("the buttons on the headset don't work"), w(), w()},
			wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree7 buttons, restart link fixed", symptom: SymptomMuteCallControl,
			turns:    []turn{u("the call buttons are dead"), w(), n(), w()},
			wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree7 buttons still dead -> genesys", symptom: SymptomMuteCallControl,
			turns:    []turn{u("none of the buttons work"), w(), n(), n()},
			wantKind: TerminalKindEscalate, wantReason: ReasonGenesysPlatform},
		{name: "tree7 desync resynced", symptom: SymptomMuteCallControl,
			turns:    []turn{u("the mute is out of sync with the app"), w()},
			wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree7 desync rebuilt link", symptom: SymptomMuteCallControl,
			turns:    []turn{u("headset shows muted but the app shows unmuted"), n(), w()},
			wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree7 desync keeps drifting -> genesys", symptom: SymptomMuteCallControl,
			turns:    []turn{u("mute keeps going out of sync"), n(), n()},
			wantKind: TerminalKindEscalate, wantReason: ReasonGenesysPlatform},

		// ---- Tree 8 — Intermittent Drops ----------------------------------
		{name: "tree8 s1 off the dock fixed", symptom: SymptomIntermittentDrops,
			turns: []turn{w()}, wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree8 s2 different port fixed", symptom: SymptomIntermittentDrops,
			turns: []turn{n(), w()}, wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree8 s3 power management fixed", symptom: SymptomIntermittentDrops,
			turns: []turn{n(), n(), w()}, wantKind: TerminalKindResolved, wantDisposition: DispositionContainedResolved},
		{name: "tree8 s4 only this machine -> tree exhausted (Tier 2, not RMA)", symptom: SymptomIntermittentDrops,
			turns:    []turn{n(), n(), n(), n()},
			wantKind: TerminalKindEscalate, wantReason: ReasonTreeExhausted},
	}

	e := newTestEngine(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := enterTree(t, e, tc.symptom)
			var view StepView
			for _, tn := range tc.turns {
				view = mustAdvance(t, e, state, tn.outcome, tn.utterance)
			}
			if tc.wantKind != "" {
				if !view.IsTerminal || view.Terminal == nil {
					t.Fatalf("expected terminal %q, got non-terminal at %s/%s", tc.wantKind, state.CurrentTree, state.CurrentStep)
				}
				if view.Terminal.Kind != tc.wantKind {
					t.Fatalf("terminal kind = %q, want %q", view.Terminal.Kind, tc.wantKind)
				}
				switch tc.wantKind {
				case TerminalKindResolved:
					if !state.Resolved {
						t.Error("state.Resolved not set")
					}
					if view.Terminal.Disposition != tc.wantDisposition {
						t.Errorf("disposition = %q, want %q", view.Terminal.Disposition, tc.wantDisposition)
					}
				case TerminalKindEscalate, TerminalKindRMA:
					if !state.Escalated {
						t.Error("state.Escalated not set")
					}
					if state.EscalationReason != tc.wantReason {
						t.Errorf("state reason = %q, want %q", state.EscalationReason, tc.wantReason)
					}
					if view.Terminal.Reason != tc.wantReason {
						t.Errorf("terminal reason = %q, want %q", view.Terminal.Reason, tc.wantReason)
					}
				}
				return
			}
			// Non-terminal landing expectation.
			if view.IsTerminal {
				t.Fatalf("expected to land mid-flow at %s/%s, got terminal %+v", tc.wantTree, tc.wantStep, view.Terminal)
			}
			if state.CurrentTree != tc.wantTree || state.CurrentStep != tc.wantStep {
				t.Fatalf("landed at %s/%s, want %s/%s", state.CurrentTree, state.CurrentStep, tc.wantTree, tc.wantStep)
			}
		})
	}
}

// Scripted Tree-8 walk reaches RMA with every step attempted in order.
func TestTree8ScriptedRMAInOrder(t *testing.T) {
	e := newTestEngine(t)
	state := enterTree(t, e, SymptomIntermittentDrops)

	mustAdvance(t, e, state, OutcomeDidntWork, "still dropping off the dock")
	mustAdvance(t, e, state, OutcomeDidntWork, "different port, still drops")
	mustAdvance(t, e, state, OutcomeDidntWork, "power management changed, still drops")
	view := mustAdvance(t, e, state, OutcomeWorked, "yes, wiggling the cable makes it cut out")

	if !view.IsTerminal || view.Terminal.Kind != TerminalKindRMA {
		t.Fatalf("expected RMA terminal, got %+v", view)
	}
	if view.Terminal.Reason != ReasonHardwareFault || view.Terminal.Priority != PriorityHigh {
		t.Errorf("RMA terminal = %q/%q, want hardware_fault/high", view.Terminal.Reason, view.Terminal.Priority)
	}
	wantOrder := []StepID{"tree8.s1", "tree8.s2", "tree8.s3", "tree8.s4"}
	if !reflect.DeepEqual(state.AttemptedSteps, wantOrder) {
		t.Errorf("AttemptedSteps = %v, want %v", state.AttemptedSteps, wantOrder)
	}
	if state.FailedSteps != 3 { // s1..s3 are failed fixes; s4 is the wiggle diagnostic
		t.Errorf("FailedSteps = %d, want 3", state.FailedSteps)
	}
	if !state.DriverReinstalled { // tree8.s3 includes USB/chipset driver updates
		t.Error("DriverReinstalled not set after tree8.s3 didnt_work")
	}
}

// ---------------------------------------------------------------------------
// Pre-flight
// ---------------------------------------------------------------------------

func TestPreflightResolvesAtEachItem(t *testing.T) {
	e := newTestEngine(t)
	for failBefore := 0; failBefore < 6; failBefore++ {
		state := &TriageState{}
		view := e.Start(state)
		if view.TreeID != PreflightTreeID || view.StepID != "preflight.s1" {
			t.Fatalf("Start landed at %s/%s", view.TreeID, view.StepID)
		}
		for i := 0; i < failBefore; i++ {
			view = mustAdvance(t, e, state, OutcomeDidntWork, "")
		}
		view = mustAdvance(t, e, state, OutcomeWorked, "that fixed it")
		if !view.IsTerminal || view.Terminal.Kind != TerminalKindResolved {
			t.Fatalf("pre-flight item %d: expected resolved terminal, got %+v", failBefore+1, view)
		}
		if view.Terminal.Disposition != DispositionContainedResolved {
			t.Errorf("pre-flight item %d: disposition %q", failBefore+1, view.Terminal.Disposition)
		}
		if state.FailedSteps != 0 {
			t.Errorf("pre-flight checks must not count as failed steps; got %d", state.FailedSteps)
		}
	}
}

func TestPreflightRoutesToClassifiedSymptomTree(t *testing.T) {
	e := newTestEngine(t)
	state := &TriageState{}
	if err := e.SetSymptom(state, SymptomMicNotWorking); err != nil {
		t.Fatal(err)
	}
	e.Start(state)
	var view StepView
	for i := 0; i < 6; i++ {
		view = mustAdvance(t, e, state, OutcomeDidntWork, "")
	}
	if view.IsTerminal {
		t.Fatalf("route-to-tree must continue, not end: %+v", view.Terminal)
	}
	if state.CurrentTree != Tree2ID || state.CurrentStep != "tree2.s1" {
		t.Fatalf("landed at %s/%s, want tree2/tree2.s1", state.CurrentTree, state.CurrentStep)
	}
	if state.RebootCount != 1 {
		t.Errorf("RebootCount = %d after failed pre-flight reboot, want 1", state.RebootCount)
	}
	if len(state.AttemptedSteps) != 6 {
		t.Errorf("AttemptedSteps = %v, want all 6 pre-flight items", state.AttemptedSteps)
	}
}

func TestPreflightWithoutSymptomRequiresClassification(t *testing.T) {
	e := newTestEngine(t)
	state := &TriageState{}
	e.Start(state)
	var err error
	for i := 0; i < 6; i++ {
		_, err = e.Advance(state, TurnResult{Outcome: OutcomeDidntWork})
	}
	if !errors.Is(err, ErrSymptomRequired) {
		t.Fatalf("expected ErrSymptomRequired, got %v", err)
	}
	// Handler classifies, then selects the tree.
	view, err := e.SelectSymptomTree(state, SymptomNoAudioOutput)
	if err != nil {
		t.Fatalf("SelectSymptomTree: %v", err)
	}
	if view.TreeID != Tree1ID || view.StepID != "tree1.s1" {
		t.Fatalf("landed at %s/%s, want tree1/tree1.s1", view.TreeID, view.StepID)
	}
}

// ---------------------------------------------------------------------------
// Cross-tree routing: tree3.s1 "$original" sentinel
// ---------------------------------------------------------------------------

func TestTree3ReturnsToOriginalSymptomTree(t *testing.T) {
	e := newTestEngine(t)
	state := enterTree(t, e, SymptomNoAudioOutput)

	// Tree 1 step 1: not recognized → detour to Tree 3.
	view := mustAdvance(t, e, state, OutcomeDidntWork, "no chime, nothing shows up")
	if state.CurrentTree != Tree3ID {
		t.Fatalf("expected detour to tree3, at %s", state.CurrentTree)
	}
	// Tree 3 step 1: reseat made it appear → back to the original tree.
	view = mustAdvance(t, e, state, OutcomeWorked, "it appeared after plugging into the computer")
	if view.IsTerminal {
		t.Fatalf("expected return to tree1, got terminal %+v", view.Terminal)
	}
	if state.CurrentTree != Tree1ID || state.CurrentStep != "tree1.s1" {
		t.Fatalf("landed at %s/%s, want tree1/tree1.s1", state.CurrentTree, state.CurrentStep)
	}
}

// ---------------------------------------------------------------------------
// Engine cross-cutting rules
// ---------------------------------------------------------------------------

func TestUnclearRepromptsOnceThenCountsAsDidntWork(t *testing.T) {
	e := newTestEngine(t)
	state := enterTree(t, e, SymptomIntermittentDrops)

	// First unclear: re-ask the same step.
	view := mustAdvance(t, e, state, OutcomeUnclear, "huh?")
	if view.StepID != "tree8.s1" || view.IsTerminal {
		t.Fatalf("first unclear should re-ask tree8.s1, got %s (terminal=%v)", view.StepID, view.IsTerminal)
	}
	if state.UnclearStreak != 1 {
		t.Errorf("UnclearStreak = %d, want 1", state.UnclearStreak)
	}
	// Second consecutive unclear: treated as didnt_work → advance to s2.
	view = mustAdvance(t, e, state, OutcomeUnclear, "I don't know what you mean")
	if view.StepID != "tree8.s2" {
		t.Fatalf("second unclear should advance to tree8.s2, got %s", view.StepID)
	}
	if state.UnclearStreak != 0 {
		t.Errorf("UnclearStreak = %d after resolution, want 0", state.UnclearStreak)
	}
	if state.FailedSteps != 1 {
		t.Errorf("FailedSteps = %d, want 1 (unclear-twice counts as a failed fix)", state.FailedSteps)
	}
}

func TestFreeFormHandledRepresentsSameStep(t *testing.T) {
	e := newTestEngine(t)
	state := enterTree(t, e, SymptomMicNotWorking)
	before := *state

	view, err := e.Advance(state, TurnResult{
		Outcome:         OutcomeUnclear,
		FreeFormHandled: true,
		RawUtterance:    "wait, where is the mute button on a Jabra?",
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.StepID != "tree2.s1" || view.IsTerminal {
		t.Fatalf("free-form turn must re-present tree2.s1, got %s", view.StepID)
	}
	if state.UnclearStreak != before.UnclearStreak || state.FailedSteps != before.FailedSteps ||
		len(state.AttemptedSteps) != len(before.AttemptedSteps) {
		t.Error("free-form turn must not move any counters")
	}
}

func TestRebootLimitGuard(t *testing.T) {
	e := newTestEngine(t)
	// User already rebooted twice and reinstalled the driver this session.
	state := enterTree(t, e, SymptomNoAudioOutput)
	state.RebootCount = RebootLimit
	state.DriverReinstalled = true

	// Walk to s4; its didnt_work would enter s6 (a reboot+driver step) —
	// the guard must escalate with reboot_limit instead of looping.
	mustAdvance(t, e, state, OutcomeWorked, "")    // s1 → s2
	mustAdvance(t, e, state, OutcomeDidntWork, "") // s2 → s3
	mustAdvance(t, e, state, OutcomeDidntWork, "") // s3 → s4
	view := mustAdvance(t, e, state, OutcomeDidntWork, "still silent in the test")

	if !view.IsTerminal || view.Terminal.Kind != TerminalKindEscalate {
		t.Fatalf("expected escalate terminal, got %+v", view)
	}
	if view.Terminal.Reason != ReasonRebootLimit {
		t.Fatalf("reason = %q, want %q", view.Terminal.Reason, ReasonRebootLimit)
	}
	if state.EscalationReason != ReasonRebootLimit || !state.Escalated {
		t.Errorf("state = escalated:%v reason:%q", state.Escalated, state.EscalationReason)
	}
}

func TestRebootAndDriverBookkeeping(t *testing.T) {
	e := newTestEngine(t)
	state := enterTree(t, e, SymptomNoAudioOutput)
	// s1 yes, s2/s3/s4 no → s6 (reboot+driver), no → terminal.
	mustAdvance(t, e, state, OutcomeWorked, "")
	mustAdvance(t, e, state, OutcomeDidntWork, "")
	mustAdvance(t, e, state, OutcomeDidntWork, "")
	mustAdvance(t, e, state, OutcomeDidntWork, "")
	view := mustAdvance(t, e, state, OutcomeDidntWork, "rebooted and reinstalled, still nothing")

	if state.RebootCount != 1 || !state.DriverReinstalled {
		t.Errorf("bookkeeping: reboots=%d driver=%v, want 1/true", state.RebootCount, state.DriverReinstalled)
	}
	if view.Terminal == nil || view.Terminal.Reason != ReasonTreeExhausted {
		t.Fatalf("expected tree_exhausted terminal, got %+v", view.Terminal)
	}
}

func TestFailedStepsThresholdEscalates(t *testing.T) {
	e := newTestEngine(t)
	state := enterTree(t, e, SymptomMicNotWorking)
	state.FailedSteps = FailedStepsEscalationThreshold - 1 // accumulated earlier (e.g. prior trees)

	// tree2.s1 is a fix step; didnt_work brings FailedSteps to the threshold,
	// so the engine escalates instead of continuing to s2.
	view := mustAdvance(t, e, state, OutcomeDidntWork, "still muted apparently")
	if !view.IsTerminal || view.Terminal.Reason != ReasonTroubleshootingExhausted {
		t.Fatalf("expected troubleshooting_exhausted, got %+v", view.Terminal)
	}
	if state.FailedSteps != FailedStepsEscalationThreshold {
		t.Errorf("FailedSteps = %d, want %d", state.FailedSteps, FailedStepsEscalationThreshold)
	}
}

func TestFrustrationThresholdEscalates(t *testing.T) {
	e := newTestEngine(t)
	state := enterTree(t, e, SymptomNoAudioOutput)
	state.FrustrationCount = FrustrationEscalationThreshold // maintained by the B-05 detector

	view := mustAdvance(t, e, state, OutcomeDidntWork, "this is ridiculous")
	if !view.IsTerminal || view.Terminal.Reason != ReasonUserFrustrated {
		t.Fatalf("expected user_frustrated, got %+v", view.Terminal)
	}
}

func TestExternallyTriggeredEscalations(t *testing.T) {
	cases := []struct {
		reason       EscalationReason
		wantKind     TerminalKind
		wantPriority EscalationPriority
	}{
		{ReasonUserRequested, TerminalKindEscalate, PriorityHigh},
		{ReasonUserCannotPerform, TerminalKindEscalate, PriorityMedium},
		{ReasonAccessibilityNeed, TerminalKindEscalate, PriorityMedium},
		{ReasonUserFrustrated, TerminalKindEscalate, PriorityMedium},
		{ReasonHardwareFault, TerminalKindRMA, PriorityHigh},
	}
	e := newTestEngine(t)
	for _, tc := range cases {
		t.Run(string(tc.reason), func(t *testing.T) {
			state := enterTree(t, e, SymptomNoAudioOutput)
			view := e.Escalate(state, tc.reason)
			if !view.IsTerminal || view.Terminal.Kind != tc.wantKind {
				t.Fatalf("kind = %+v, want %q", view.Terminal, tc.wantKind)
			}
			if view.Terminal.Priority != tc.wantPriority {
				t.Errorf("priority = %q, want %q", view.Terminal.Priority, tc.wantPriority)
			}
			if !state.Escalated || state.EscalationReason != tc.reason {
				t.Errorf("state = escalated:%v reason:%q", state.Escalated, state.EscalationReason)
			}
			if view.Terminal.ReadAloudKey == "" {
				t.Error("escalation terminal must carry a handoff ReadAloudKey")
			}
		})
	}
}

// Every EscalationReason must be reachable; this asserts the full inventory
// (tree-driven reasons are covered by TestTreePaths; engine guards and the
// external API cover the rest — this is the cross-check).
func TestEveryEscalationReasonReachable(t *testing.T) {
	reached := map[EscalationReason]bool{
		// tree-driven (TestTreePaths)
		ReasonHardwareFault:        true, // tree2.s5 / tree3.s3 / tree3.s5 / tree4.s4 / tree5.s3 / tree8.s4
		ReasonTreeExhausted:        true, // tree1.s6 / tree6.s2 / tree8.s4
		ReasonManagedMachinePolicy: true, // tree3.s5
		ReasonGenesysPlatform:      true, // tree1.s5 / tree2.s4 / tree5 / tree6.s3 / tree7
		// engine guards
		ReasonRebootLimit:              true, // TestRebootLimitGuard
		ReasonTroubleshootingExhausted: true, // TestFailedStepsThresholdEscalates
		ReasonUserFrustrated:           true, // TestFrustrationThresholdEscalates
		// external API (B-05 detector / intake)
		ReasonUserRequested:     true, // TestExternallyTriggeredEscalations
		ReasonUserCannotPerform: true, // TestExternallyTriggeredEscalations
		ReasonAccessibilityNeed: true, // TestExternallyTriggeredEscalations
	}
	all := []EscalationReason{
		ReasonUserRequested, ReasonUserFrustrated, ReasonTroubleshootingExhausted,
		ReasonHardwareFault, ReasonRebootLimit, ReasonTreeExhausted,
		ReasonManagedMachinePolicy, ReasonUserCannotPerform, ReasonAccessibilityNeed,
		ReasonGenesysPlatform,
	}
	for _, r := range all {
		if !reached[r] {
			t.Errorf("escalation reason %q has no covering test", r)
		}
		// And each must produce a well-formed terminal.
		term := EscalationTerminal(r)
		if term.Reason != r || term.Priority == "" || term.ReadAloudKey == "" {
			t.Errorf("EscalationTerminal(%q) malformed: %+v", r, term)
		}
	}
}

// ---------------------------------------------------------------------------
// Lifecycle errors and views
// ---------------------------------------------------------------------------

func TestLifecycleErrors(t *testing.T) {
	e := newTestEngine(t)

	// Not started.
	if _, err := e.Advance(&TriageState{}, TurnResult{Outcome: OutcomeWorked}); !errors.Is(err, ErrNotStarted) {
		t.Errorf("Advance before Start: %v, want ErrNotStarted", err)
	}
	if _, err := e.View(&TriageState{}); !errors.Is(err, ErrNotStarted) {
		t.Errorf("View before Start: %v, want ErrNotStarted", err)
	}

	// Ended (resolved).
	state := enterTree(t, e, SymptomMicNotWorking)
	mustAdvance(t, e, state, OutcomeWorked, "unmuted, they hear me now")
	if _, err := e.Advance(state, TurnResult{Outcome: OutcomeWorked}); !errors.Is(err, ErrFlowEnded) {
		t.Errorf("Advance after resolution: %v, want ErrFlowEnded", err)
	}
	if _, err := e.View(state); !errors.Is(err, ErrFlowEnded) {
		t.Errorf("View after resolution: %v, want ErrFlowEnded", err)
	}
	if _, err := e.SelectSymptomTree(state, SymptomNoAudioOutput); !errors.Is(err, ErrFlowEnded) {
		t.Errorf("SelectSymptomTree after resolution: %v, want ErrFlowEnded", err)
	}

	// Unknown symptom class.
	if err := e.SetSymptom(&TriageState{}, "made_up"); err == nil {
		t.Error("SetSymptom with unknown class should fail")
	}

	// Corrupt state.
	if _, err := e.View(&TriageState{CurrentTree: "treeX", CurrentStep: "x"}); err == nil {
		t.Error("View with unknown tree should fail")
	}
	if _, err := e.View(&TriageState{CurrentTree: Tree1ID, CurrentStep: "tree1.nope"}); err == nil {
		t.Error("View with unknown step should fail")
	}
}

func TestViewRendersCurrentStep(t *testing.T) {
	e := newTestEngine(t)
	state := enterTree(t, e, SymptomOneSidedAudio)
	view, err := e.View(state)
	if err != nil {
		t.Fatal(err)
	}
	if view.TreeID != Tree4ID || view.StepID != "tree4.s1" || view.Ordinal != 1 {
		t.Fatalf("view = %+v", view)
	}
	if view.TreeTitle == "" || view.Goal == "" || view.TotalSteps != 4 {
		t.Errorf("view orientation fields incomplete: %+v", view)
	}
	if view.ReadAloud == "" || view.KBDoc == "" {
		t.Errorf("view missing prompt/KB refs: %+v", view)
	}
}
