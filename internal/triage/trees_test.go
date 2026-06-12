package triage

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Fork shape + referential integrity
// ---------------------------------------------------------------------------

func TestBuildSymptomForkValidates(t *testing.T) {
	fork := BuildSymptomFork()
	if err := ValidateFork(fork); err != nil {
		t.Fatalf("built-in fork failed validation: %v", err)
	}
	if _, err := NewEngine(fork); err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	// NewEngine must refuse an invalid fork.
	broken := BuildSymptomFork()
	delete(broken.Trees, SymptomMicNotWorking)
	if _, err := NewEngine(broken); err == nil {
		t.Fatal("NewEngine accepted a fork missing a symptom class")
	}
}

func TestNewDefaultEngineDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewDefaultEngine panicked: %v", r)
		}
	}()
	if e := NewDefaultEngine(); e == nil {
		t.Fatal("NewDefaultEngine returned nil")
	}
}

func TestForkShape(t *testing.T) {
	fork := BuildSymptomFork()

	if fork.PreFlight.ID != PreflightTreeID || fork.PreFlight.Symptom != SymptomPreFlight {
		t.Errorf("pre-flight tree mis-identified: %q/%q", fork.PreFlight.ID, fork.PreFlight.Symptom)
	}

	wantTrees := map[SymptomClass]struct {
		id    string
		steps int
	}{
		SymptomNoAudioOutput:     {Tree1ID, 6},
		SymptomMicNotWorking:     {Tree2ID, 5},
		SymptomNotDetected:       {Tree3ID, 5},
		SymptomOneSidedAudio:     {Tree4ID, 4},
		SymptomDistortedAudio:    {Tree5ID, 5}, // s1..s4 + s4b
		SymptomVolumeSidetone:    {Tree6ID, 3},
		SymptomMuteCallControl:   {Tree7ID, 6},
		SymptomIntermittentDrops: {Tree8ID, 4},
	}
	if len(fork.Trees) != len(wantTrees) {
		t.Fatalf("expected %d symptom trees, got %d", len(wantTrees), len(fork.Trees))
	}
	for class, want := range wantTrees {
		tree, ok := fork.Trees[class]
		if !ok {
			t.Errorf("missing tree for class %q", class)
			continue
		}
		if tree.ID != want.id {
			t.Errorf("class %q: tree ID = %q, want %q", class, tree.ID, want.id)
		}
		if tree.Symptom != class {
			t.Errorf("tree %q: Symptom = %q, want %q", tree.ID, tree.Symptom, class)
		}
		if len(tree.Steps) != want.steps {
			t.Errorf("tree %q: %d steps, want %d", tree.ID, len(tree.Steps), want.steps)
		}
		if tree.Title == "" || tree.Goal == "" {
			t.Errorf("tree %q: missing title/goal", tree.ID)
		}
		if _, ok := tree.Steps[tree.EntryStepID]; !ok {
			t.Errorf("tree %q: entry step %q not in Steps", tree.ID, tree.EntryStepID)
		}
	}
	if len(fork.PreFlight.Steps) != 6 {
		t.Errorf("pre-flight: %d steps, want 6", len(fork.PreFlight.Steps))
	}
}

// Every step must reference its tree's KB doc.
func TestKBDocWiring(t *testing.T) {
	fork := BuildSymptomFork()
	wantDoc := map[string]KBDocRef{
		PreflightTreeID: "trees/preflight-checklist.md",
		Tree1ID:         "trees/tree-1-no-audio-output.md",
		Tree2ID:         "trees/tree-2-mic-not-working.md",
		Tree3ID:         "trees/tree-3-headset-not-detected.md",
		Tree4ID:         "trees/tree-4-one-sided-audio.md",
		Tree5ID:         "trees/tree-5-distorted-choppy-echo.md",
		Tree6ID:         "trees/tree-6-volume-sidetone.md",
		Tree7ID:         "trees/tree-7-mute-sync-buttons.md",
		Tree8ID:         "trees/tree-8-intermittent-disconnects.md",
	}
	for _, tree := range allTrees(fork) {
		want, ok := wantDoc[tree.ID]
		if !ok {
			t.Fatalf("unexpected tree %q", tree.ID)
		}
		for id, step := range tree.Steps {
			if step.KBDocRef != want {
				t.Errorf("%s/%s: KBDocRef = %q, want %q", tree.ID, id, step.KBDocRef, want)
			}
			if step.ReadAloudKey == "" {
				t.Errorf("%s/%s: empty ReadAloudKey", tree.ID, id)
			}
		}
	}
	if EscalationKBDoc != "trees/escalation-criteria.md" {
		t.Errorf("EscalationKBDoc = %q", EscalationKBDoc)
	}
}

// No catch-all: every escalate/RMA terminal in the data carries one of the
// enumerated tree-driven reasons; every resolved terminal has a disposition.
func TestTerminalsHaveSpecificReasons(t *testing.T) {
	validTreeReasons := map[EscalationReason]bool{
		ReasonHardwareFault:        true,
		ReasonTreeExhausted:        true,
		ReasonManagedMachinePolicy: true,
		ReasonGenesysPlatform:      true,
	}
	fork := BuildSymptomFork()
	for _, tree := range allTrees(fork) {
		for id, step := range tree.Steps {
			for name, tr := range map[string]Transition{
				"worked": step.OnWorked, "didnt_work": step.OnDidntWork, "unclear": step.OnUnclear,
			} {
				if tr.Terminal == nil {
					continue
				}
				term := tr.Terminal
				loc := tree.ID + "/" + string(id) + "/" + name
				switch term.Kind {
				case TerminalKindResolved:
					if term.Disposition != DispositionContainedResolved && term.Disposition != DispositionExpectationSet {
						t.Errorf("%s: unknown disposition %q", loc, term.Disposition)
					}
				case TerminalKindEscalate:
					if !validTreeReasons[term.Reason] {
						t.Errorf("%s: escalate terminal has non-tree reason %q", loc, term.Reason)
					}
				case TerminalKindRMA:
					if term.Reason != ReasonHardwareFault || term.Priority != PriorityHigh {
						t.Errorf("%s: RMA terminal must be hardware_fault/high, got %q/%q", loc, term.Reason, term.Priority)
					}
				case TerminalKindRouteToTree:
					if term.RouteTo == nil {
						t.Errorf("%s: route terminal without RouteTo", loc)
					}
				default:
					t.Errorf("%s: unknown terminal kind %q", loc, term.Kind)
				}
			}
		}
	}
}

// The RMA leaves called out in docs/triage-design.md §6.
func TestRMALeaves(t *testing.T) {
	fork := BuildSymptomFork()
	wantRMA := []struct {
		tree string
		step StepID
		on   string // which transition
	}{
		{Tree2ID, "tree2.s5", "didnt_work"},
		{Tree3ID, "tree3.s3", "didnt_work"},
		{Tree3ID, "tree3.s5", "didnt_work"},
		{Tree4ID, "tree4.s4", "worked"},
		{Tree4ID, "tree4.s4", "didnt_work"},
		{Tree5ID, "tree5.s3", "didnt_work"},
		{Tree8ID, "tree8.s4", "worked"},
	}
	for _, w := range wantRMA {
		tree := treeByID(t, fork, w.tree)
		step := tree.Steps[w.step]
		tr := step.OnDidntWork
		if w.on == "worked" {
			tr = step.OnWorked
		}
		if tr.Terminal == nil || tr.Terminal.Kind != TerminalKindRMA {
			t.Errorf("%s/%s/%s: expected RMA terminal", w.tree, w.step, w.on)
		}
	}
}

func TestRebootDriverStepFlags(t *testing.T) {
	wantReboot := []StepID{"preflight.s6", "tree1.s6", "tree2.s5"}
	wantDriver := []StepID{"tree1.s6", "tree2.s5", "tree3.s4", "tree5.s3", "tree6.s2", "tree8.s3"}
	for _, id := range wantReboot {
		if !IsRebootStep(id) {
			t.Errorf("IsRebootStep(%s) = false", id)
		}
	}
	for _, id := range wantDriver {
		if !IsDriverReinstallStep(id) {
			t.Errorf("IsDriverReinstallStep(%s) = false", id)
		}
	}
	if IsRebootStep("tree4.s1") || IsDriverReinstallStep("tree4.s1") {
		t.Error("tree4.s1 should not be a reboot/driver step")
	}
}

// ---------------------------------------------------------------------------
// Validation negatives — ValidateFork catches every invariant violation
// ---------------------------------------------------------------------------

func TestValidateForkNegatives(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(f *SymptomFork)
		wantErr string
	}{
		{
			name:    "nil preflight",
			mutate:  func(f *SymptomFork) { f.PreFlight = nil },
			wantErr: "pre-flight",
		},
		{
			name:    "missing symptom class",
			mutate:  func(f *SymptomFork) { delete(f.Trees, SymptomOneSidedAudio) },
			wantErr: "has no tree",
		},
		{
			name: "dangling next step",
			mutate: func(f *SymptomFork) {
				f.Trees[SymptomNoAudioOutput].Steps["tree1.s1"].OnWorked = next("tree1.nope")
			},
			wantErr: "unknown step",
		},
		{
			name: "dangling cross-tree ref",
			mutate: func(f *SymptomFork) {
				f.Trees[SymptomNoAudioOutput].Steps["tree1.s1"].OnWorked = jump("tree99", "tree99.s1")
			},
			wantErr: "unknown tree",
		},
		{
			name: "transition with both next and terminal",
			mutate: func(f *SymptomFork) {
				s := f.Trees[SymptomNoAudioOutput].Steps["tree1.s2"]
				s.OnWorked = Transition{
					NextStep: &StepRef{StepID: "tree1.s3"},
					Terminal: resolvedT("x", DispositionContainedResolved),
				}
			},
			wantErr: "exactly one",
		},
		{
			name: "transition with neither",
			mutate: func(f *SymptomFork) {
				f.Trees[SymptomNoAudioOutput].Steps["tree1.s2"].OnUnclear = Transition{}
			},
			wantErr: "exactly one",
		},
		{
			name: "resolved terminal without disposition",
			mutate: func(f *SymptomFork) {
				f.Trees[SymptomNoAudioOutput].Steps["tree1.s2"].OnWorked = term(&Terminal{Kind: TerminalKindResolved})
			},
			wantErr: "missing disposition",
		},
		{
			name: "escalate terminal without reason",
			mutate: func(f *SymptomFork) {
				f.Trees[SymptomNoAudioOutput].Steps["tree1.s5"].OnDidntWork = term(&Terminal{Kind: TerminalKindEscalate})
			},
			wantErr: "missing reason",
		},
		{
			name: "route terminal without RouteTo",
			mutate: func(f *SymptomFork) {
				f.PreFlight.Steps["preflight.s6"].OnDidntWork = term(&Terminal{Kind: TerminalKindRouteToTree})
			},
			wantErr: "missing RouteTo",
		},
		{
			name: "unknown terminal kind",
			mutate: func(f *SymptomFork) {
				f.Trees[SymptomNoAudioOutput].Steps["tree1.s2"].OnWorked = term(&Terminal{Kind: "weird"})
			},
			wantErr: "unknown terminal kind",
		},
		{
			name: "entry step missing",
			mutate: func(f *SymptomFork) {
				f.Trees[SymptomIntermittentDrops].EntryStepID = "tree8.nope"
			},
			wantErr: "entry step",
		},
		{
			name: "step key/ID mismatch",
			mutate: func(f *SymptomFork) {
				tree := f.Trees[SymptomVolumeSidetone]
				s := tree.Steps["tree6.s2"]
				s.ID = "tree6.sX"
			},
			wantErr: "does not match",
		},
		{
			name: "dangling branch ref",
			mutate: func(f *SymptomFork) {
				f.Trees[SymptomDistortedAudio].Steps["tree5.s1"].Branches[BranchEcho] = &StepRef{StepID: "tree5.nope"}
			},
			wantErr: "unknown step",
		},
		{
			name: "missing read-aloud key",
			mutate: func(f *SymptomFork) {
				f.Trees[SymptomMicNotWorking].Steps["tree2.s1"].ReadAloudKey = ""
			},
			wantErr: "ReadAloudKey",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fork := BuildSymptomFork() // fresh copy every time
			tc.mutate(fork)
			err := ValidateFork(fork)
			if err == nil {
				t.Fatalf("expected validation error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// BuildSymptomFork must return independent copies (no shared mutable state).
func TestBuildSymptomForkIsFresh(t *testing.T) {
	a := BuildSymptomFork()
	b := BuildSymptomFork()
	a.Trees[SymptomNoAudioOutput].Steps["tree1.s1"].ReadAloudKey = "mutated"
	if b.Trees[SymptomNoAudioOutput].Steps["tree1.s1"].ReadAloudKey == "mutated" {
		t.Fatal("BuildSymptomFork shares state between calls")
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func allTrees(f *SymptomFork) []*Tree {
	out := []*Tree{f.PreFlight}
	for _, class := range AllSymptomClasses {
		out = append(out, f.Trees[class])
	}
	return out
}

func treeByID(t *testing.T, f *SymptomFork, id string) *Tree {
	t.Helper()
	for _, tree := range allTrees(f) {
		if tree.ID == id {
			return tree
		}
	}
	t.Fatalf("tree %q not found", id)
	return nil
}
