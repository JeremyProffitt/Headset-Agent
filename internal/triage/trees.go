// Tree DATA for the Headset Support Agent triage engine (B-04).
//
// This file encodes troubleshooting.md §1 — the Universal Pre-Flight Checklist
// and diagnostic Trees 1–8 — as instances of the B-03 schema in types.go.
// Every If-YES / If-NO branch in §1 maps to a Step transition or a Terminal
// (resolved / escalate-with-reason / RMA / route-to-tree). There is no
// "misc"/catch-all anywhere.
//
// Outcome semantics on question steps: OutcomeWorked carries the AFFIRMATIVE
// branch ("If YES" — the check passed / the fix helped) and OutcomeDidntWork
// the negative ("If NO" — still broken / already correct / check failed),
// mirroring the R/N mapping in docs/triage-design.md §5.
package triage

import "fmt"

// ---------------------------------------------------------------------------
// Stable tree IDs and routing sentinels
// ---------------------------------------------------------------------------

// Tree IDs, stable across sessions/logs (TriageState.CurrentTree values).
const (
	PreflightTreeID = "preflight"
	Tree1ID         = "tree1"
	Tree2ID         = "tree2"
	Tree3ID         = "tree3"
	Tree4ID         = "tree4"
	Tree5ID         = "tree5"
	Tree6ID         = "tree6"
	Tree7ID         = "tree7"
	Tree8ID         = "tree8"
)

// Routing sentinels used in Terminal.RouteTo.TreeID where the target tree is
// not known until runtime. The engine resolves them from TriageState.Symptom.
const (
	// RouteSymptomTree — "route to the tree for the classified symptom".
	// Used by the pre-flight handoff: the classifier runs at the route point
	// (docs/triage-design.md §5). If no symptom is classified yet the engine
	// returns ErrSymptomRequired so the handler can classify and call
	// SelectSymptomTree.
	RouteSymptomTree = "$symptom"
	// RouteOriginalTree — "return to the symptom the user originally had"
	// (Tree 3 step 1: the headset now appears, so resume Tree 1/2/etc.).
	// When the original symptom WAS not_detected, appearance resolves it and
	// the engine fires a Resolved terminal instead.
	RouteOriginalTree = "$original"
)

// Disposition codes for Resolved terminals (E-8.1 taxonomy subset).
const (
	// DispositionContainedResolved — the issue was fixed during the call.
	DispositionContainedResolved = "contained_resolved"
	// DispositionExpectationSet — behavior is expected/by-design (mono call
	// audio, unsupported call-control model, no-sidetone hardware); the agent
	// explained it and the user accepted. No fault to fix.
	DispositionExpectationSet = "expectation_set"
)

// EscalationKBDoc is the KB doc grounding every escalation/RMA handoff
// (the §1 "When to Escalate" criteria). B-05 hands this to the agent when a
// Terminal of kind escalate/rma fires (Terminal itself carries no KBDocRef).
const EscalationKBDoc KBDocRef = "trees/escalation-criteria.md"

// Branch keys for the n-way classification forks (Step.Branches). See
// docs/triage-design.md §7 and ClassifyBranch in classify.go.
const (
	BranchRobotic    = "robotic"     // tree5.s1 — robotic/choppy → network/CPU path
	BranchCrackling  = "crackling"   // tree5.s1 — crackling/static → USB/driver path
	BranchEcho       = "echo"        // tree5.s1 — echo → echo path
	BranchSelf       = "self"        // tree5.s4 / tree6.s1 — user hears own voice (sidetone)
	BranchOtherParty = "other_party" // tree5.s4 / tree6.s1 — the other party's side
	BranchButtons    = "buttons"     // tree7.s1 — call-control buttons dead
	BranchDesync     = "desync"      // tree7.s1 — mute state out of sync
)

// ---------------------------------------------------------------------------
// Reboot / driver-reinstall step flags — the "≥ 2 reboots / no progress" rule
// ---------------------------------------------------------------------------

// rebootSteps marks steps whose remediation includes a full reboot. The engine
// increments TriageState.RebootCount when such a step reports didnt_work (the
// user performed the reboot and the symptom remained).
var rebootSteps = map[StepID]bool{
	"preflight.s6": true, // pre-flight #6: reboot if needed
	"tree1.s6":     true, // Windows-layer remediation: ... reboot, driver
	"tree2.s5":     true, // mic privacy + port + reboot + driver
}

// driverSteps marks steps whose remediation includes a driver update/reinstall.
// The engine sets TriageState.DriverReinstalled on didnt_work.
var driverSteps = map[StepID]bool{
	"tree1.s6": true, // update/reinstall the audio driver (§2.7)
	"tree2.s5": true, // update/reinstall the audio driver (§2.7)
	"tree3.s4": true, // Device Manager: uninstall/scan/update driver
	"tree5.s3": true, // crackling path: update/reinstall the audio driver
	"tree6.s2": true, // call volume: update the driver, then escalate
	"tree8.s3": true, // update USB/chipset drivers + dock firmware
}

// IsRebootStep reports whether the step's remediation includes a full reboot.
func IsRebootStep(id StepID) bool { return rebootSteps[id] }

// IsDriverReinstallStep reports whether the step's remediation includes a
// driver update/reinstall.
func IsDriverReinstallStep(id StepID) bool { return driverSteps[id] }

// isGuardedRemediation reports whether entering this step would re-run a
// reboot/driver remediation — the set the "≥ 2 reboots / no progress" guard
// (ReasonRebootLimit) protects against looping back into.
func isGuardedRemediation(id StepID) bool { return rebootSteps[id] || driverSteps[id] }

// ---------------------------------------------------------------------------
// Construction helpers (data-shape sugar only; no logic)
// ---------------------------------------------------------------------------

func next(id StepID) Transition { return Transition{NextStep: &StepRef{StepID: id}} }

func jump(treeID string, id StepID) Transition {
	return Transition{NextStep: &StepRef{TreeID: treeID, StepID: id}}
}

func term(t *Terminal) Transition { return Transition{Terminal: t} }

func resolvedT(key PromptRef, disposition string) *Terminal {
	return &Terminal{Kind: TerminalKindResolved, Disposition: disposition, ReadAloudKey: key}
}

func escalateT(reason EscalationReason, priority EscalationPriority, key PromptRef) *Terminal {
	return &Terminal{Kind: TerminalKindEscalate, Reason: reason, Priority: priority, ReadAloudKey: key}
}

func rmaT(key PromptRef) *Terminal {
	return &Terminal{Kind: TerminalKindRMA, Reason: ReasonHardwareFault, Priority: PriorityHigh, ReadAloudKey: key}
}

func routeT(treeID string, id StepID, key PromptRef) *Terminal {
	return &Terminal{Kind: TerminalKindRouteToTree, RouteTo: &StepRef{TreeID: treeID, StepID: id}, ReadAloudKey: key}
}

// selfRef returns the OnUnclear self-reference transition ("re-ask this step").
func selfRef(id StepID) Transition { return next(id) }

// ---------------------------------------------------------------------------
// BuildSymptomFork — the fully-populated fork (pre-flight + Trees 1–8)
// ---------------------------------------------------------------------------

// BuildSymptomFork constructs the complete symptom fork from troubleshooting.md
// §1. A fresh value is built on every call so callers may not mutate shared
// state. Referential integrity is validated by ValidateFork / NewEngine.
func BuildSymptomFork() *SymptomFork {
	return &SymptomFork{
		PreFlight: buildPreflight(),
		Trees: map[SymptomClass]*Tree{
			SymptomNoAudioOutput:     buildTree1(),
			SymptomMicNotWorking:     buildTree2(),
			SymptomNotDetected:       buildTree3(),
			SymptomOneSidedAudio:     buildTree4(),
			SymptomDistortedAudio:    buildTree5(),
			SymptomVolumeSidetone:    buildTree6(),
			SymptomMuteCallControl:   buildTree7(),
			SymptomIntermittentDrops: buildTree8(),
		},
	}
}

// buildPreflight — Universal Pre-Flight Checklist (always runs first).
// Each item: worked → Resolved(contained_resolved); didn't help → next item.
// After the final item (reboot), an unresolved symptom routes to the
// classifier-selected tree ($symptom sentinel) — pre-flight never dead-ends.
func buildPreflight() *Tree {
	const kb = KBDocRef("trees/preflight-checklist.md")
	t := &Tree{
		ID:          PreflightTreeID,
		Symptom:     SymptomPreFlight,
		Title:       "Universal Pre-Flight Checklist",
		Goal:        "resolve the large majority of headset tickets in under two minutes before entering any tree",
		EntryStepID: "preflight.s1",
		Steps:       map[StepID]*Step{},
	}
	items := []struct {
		id  StepID
		ord int
	}{
		{"preflight.s1", 1}, // reseat USB directly into the computer
		{"preflight.s2", 2}, // confirm selected/default device (Windows + softphone)
		{"preflight.s3", 3}, // check the hardware mute
		{"preflight.s4", 4}, // turn the volume up (headset dial + Windows mixer)
		{"preflight.s5", 5}, // only one app using the microphone
		{"preflight.s6", 6}, // reboot if needed (re-plug; refresh/restart Genesys)
	}
	for i, it := range items {
		var onDidnt Transition
		if i < len(items)-1 {
			onDidnt = next(items[i+1].id)
		} else {
			// Pre-flight exhausted → hand off to the symptom tree.
			onDidnt = term(routeT(RouteSymptomTree, "", "preflight.route_to_tree"))
		}
		t.Steps[it.id] = &Step{
			ID:           it.id,
			Ordinal:      it.ord,
			ReadAloudKey: PromptRef(it.id),
			KBDocRef:     kb,
			OnWorked:     term(resolvedT(PromptRef(string(it.id)+".resolved"), DispositionContainedResolved)),
			OnDidntWork:  onDidnt,
			OnUnclear:    selfRef(it.id),
		}
	}
	return t
}

// buildTree1 — Tree 1: No Audio Output / Can't Hear Anything.
func buildTree1() *Tree {
	const kb = KBDocRef("trees/tree-1-no-audio-output.md")
	return &Tree{
		ID:          Tree1ID,
		Symptom:     SymptomNoAudioOutput,
		Title:       "No Audio Output / Can't Hear Anything",
		Goal:        "get sound into the user's ears — work from \"is it even selected and turned up\" outward",
		EntryStepID: "tree1.s1",
		Steps: map[StepID]*Step{
			// 1. Plugged directly in and powered/recognized?
			//    NO → Tree 3 (detection problem, not output). YES → continue.
			"tree1.s1": {
				ID: "tree1.s1", Ordinal: 1, ReadAloudKey: "tree1.s1", KBDocRef: kb,
				OnWorked:    next("tree1.s2"),
				OnDidntWork: term(routeT(Tree3ID, "tree3.s1", "tree1.s1.route_tree3")),
				OnUnclear:   selfRef("tree1.s1"),
			},
			// 2. Headset set as the Windows output device?
			//    Fixing it restored sound → resolved. Already correct → continue.
			"tree1.s2": {
				ID: "tree1.s2", Ordinal: 2, ReadAloudKey: "tree1.s2", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree1.s2.resolved", DispositionContainedResolved)),
				OnDidntWork: next("tree1.s3"),
				OnUnclear:   selfRef("tree1.s2"),
			},
			// 3. Volume and mute check (Windows, inline dial, hardware mute, mixer).
			"tree1.s3": {
				ID: "tree1.s3", Ordinal: 3, ReadAloudKey: "tree1.s3", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree1.s3.resolved", DispositionContainedResolved)),
				OnDidntWork: next("tree1.s4"),
				OnUnclear:   selfRef("tree1.s3"),
			},
			// 4. Quick playback test outside the softphone.
			//    Hear it → problem is inside the softphone (s5). Silent → Windows layer (s6).
			"tree1.s4": {
				ID: "tree1.s4", Ordinal: 4, ReadAloudKey: "tree1.s4", KBDocRef: kb,
				OnWorked:    next("tree1.s5"),
				OnDidntWork: next("tree1.s6"),
				OnUnclear:   selfRef("tree1.s4"),
			},
			// 5. Softphone (Genesys) output device selection + test tone (§4).
			//    Still silent in calls only → beyond-the-desk Genesys/platform.
			"tree1.s5": {
				ID: "tree1.s5", Ordinal: 5, ReadAloudKey: "tree1.s5", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree1.s5.resolved", DispositionContainedResolved)),
				OnDidntWork: term(escalateT(ReasonGenesysPlatform, PriorityMedium, "tree1.s5.escalate")),
				OnUnclear:   selfRef("tree1.s5"),
			},
			// 6. Windows-layer remediation: troubleshooter → enable device →
			//    different port → reboot → update/reinstall driver (§2).
			//    Still no sound anywhere after reboot + driver → tree exhausted
			//    (suspect hardware; reboot-limit guard also covers re-entry).
			"tree1.s6": {
				ID: "tree1.s6", Ordinal: 6, ReadAloudKey: "tree1.s6", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree1.s6.resolved", DispositionContainedResolved)),
				OnDidntWork: term(escalateT(ReasonTreeExhausted, PriorityMedium, "tree1.s6.escalate")),
				OnUnclear:   selfRef("tree1.s6"),
			},
		},
	}
}

// buildTree2 — Tree 2: Other Party Can't Hear Me / Microphone Not Working.
func buildTree2() *Tree {
	const kb = KBDocRef("trees/tree-2-mic-not-working.md")
	return &Tree{
		ID:          Tree2ID,
		Symptom:     SymptomMicNotWorking,
		Title:       "Other Party Can't Hear Me / Microphone Not Working",
		Goal:        "the #1 complaint — most common causes in order: hardware mute, wrong mic selected, app mic permission off, another app holding the mic",
		EntryStepID: "tree2.s1",
		Steps: map[StepID]*Step{
			// 1. Hardware mute first (most common).
			"tree2.s1": {
				ID: "tree2.s1", Ordinal: 1, ReadAloudKey: "tree2.s1", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree2.s1.resolved", DispositionContainedResolved)),
				OnDidntWork: next("tree2.s2"),
				OnUnclear:   selfRef("tree2.s1"),
			},
			// 2. Headset selected as the Windows input device?
			"tree2.s2": {
				ID: "tree2.s2", Ordinal: 2, ReadAloudKey: "tree2.s2", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree2.s2.resolved", DispositionContainedResolved)),
				OnDidntWork: next("tree2.s3"),
				OnUnclear:   selfRef("tree2.s2"),
			},
			// 3. Input level meter moves when speaking?
			//    Moves → Windows hears it; problem is softphone/permissions (s4).
			//    Doesn't move → Windows can't hear it (s5).
			"tree2.s3": {
				ID: "tree2.s3", Ordinal: 3, ReadAloudKey: "tree2.s3", KBDocRef: kb,
				OnWorked:    next("tree2.s4"),
				OnDidntWork: next("tree2.s5"),
				OnUnclear:   selfRef("tree2.s3"),
			},
			// 4. Softphone mic selection + browser permission + in-call mute (§4).
			//    Still silent to the other party → Genesys/platform escalation.
			"tree2.s4": {
				ID: "tree2.s4", Ordinal: 4, ReadAloudKey: "tree2.s4", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree2.s4.resolved", DispositionContainedResolved)),
				OnDidntWork: term(escalateT(ReasonGenesysPlatform, PriorityMedium, "tree2.s4.escalate")),
				OnUnclear:   selfRef("tree2.s4"),
			},
			// 5. Windows mic privacy + close mic-holding apps + port + reboot +
			//    driver (§2). Level meter still never moves after reboot + driver
			//    + known-good port → failed mic/boom → RMA.
			"tree2.s5": {
				ID: "tree2.s5", Ordinal: 5, ReadAloudKey: "tree2.s5", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree2.s5.resolved", DispositionContainedResolved)),
				OnDidntWork: term(rmaT("tree2.s5.rma")),
				OnUnclear:   selfRef("tree2.s5"),
			},
		},
	}
}

// buildTree3 — Tree 3: Headset Not Detected.
func buildTree3() *Tree {
	const kb = KBDocRef("trees/tree-3-headset-not-detected.md")
	return &Tree{
		ID:          Tree3ID,
		Symptom:     SymptomNotDetected,
		Title:       "Headset Not Detected",
		Goal:        "get the device to physically appear — until Windows sees it, no app can",
		EntryStepID: "tree3.s1",
		Steps: map[StepID]*Step{
			// 1. Reseat directly into the computer.
			//    Appears → return to the symptom the user originally had
			//    ($original sentinel: Tree 1/2/etc., or Resolved when the
			//    original complaint WAS detection). Still nothing → continue.
			"tree3.s1": {
				ID: "tree3.s1", Ordinal: 1, ReadAloudKey: "tree3.s1", KBDocRef: kb,
				OnWorked:    term(routeT(RouteOriginalTree, "", "tree3.s1.route_original")),
				OnDidntWork: next("tree3.s2"),
				OnUnclear:   selfRef("tree3.s1"),
			},
			// 2. Different USB port (different side/controller, A vs C).
			"tree3.s2": {
				ID: "tree3.s2", Ordinal: 2, ReadAloudKey: "tree3.s2", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree3.s2.resolved", DispositionContainedResolved)),
				OnDidntWork: next("tree3.s3"),
				OnUnclear:   selfRef("tree3.s2"),
			},
			// 3. Different computer or cable.
			//    Works elsewhere → this PC is the problem (s4).
			//    Fails on every port and machine → dead headset/cable → RMA.
			"tree3.s3": {
				ID: "tree3.s3", Ordinal: 3, ReadAloudKey: "tree3.s3", KBDocRef: kb,
				OnWorked:    next("tree3.s4"),
				OnDidntWork: term(rmaT("tree3.s3.rma")),
				OnUnclear:   selfRef("tree3.s3"),
			},
			// 4. Device Manager: yellow ! / unknown / absent → uninstall, scan,
			//    reboot, update driver / chipset+USB drivers (§2).
			//    Fixed → resolved. Not fixed → policy check (s5).
			"tree3.s4": {
				ID: "tree3.s4", Ordinal: 4, ReadAloudKey: "tree3.s4", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree3.s4.resolved", DispositionContainedResolved)),
				OnDidntWork: next("tree3.s5"),
				OnUnclear:   selfRef("tree3.s4"),
			},
			// 5. Managed-machine / policy check.
			//    Policy/whitelisting suspected → Tier 2 / IT.
			//    Known-good machine, no policy, still undetected → hardware → RMA.
			"tree3.s5": {
				ID: "tree3.s5", Ordinal: 5, ReadAloudKey: "tree3.s5", KBDocRef: kb,
				OnWorked:    term(escalateT(ReasonManagedMachinePolicy, PriorityMedium, "tree3.s5.escalate")),
				OnDidntWork: term(rmaT("tree3.s5.rma")),
				OnUnclear:   selfRef("tree3.s5"),
			},
		},
	}
}

// buildTree4 — Tree 4: One-Sided Audio / Only One Ear / Mono.
func buildTree4() *Tree {
	const kb = KBDocRef("trees/tree-4-one-sided-audio.md")
	return &Tree{
		ID:          Tree4ID,
		Symptom:     SymptomOneSidedAudio,
		Title:       "One-Sided Audio / Only One Ear / Mono",
		Goal:        "separate a settings problem (balance, mono toggle) from a hardware fault (cable, earcup)",
		EntryStepID: "tree4.s1",
		Steps: map[StepID]*Step{
			// 1. Windows balance L/R equal?
			"tree4.s1": {
				ID: "tree4.s1", Ordinal: 1, ReadAloudKey: "tree4.s1", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree4.s1.resolved", DispositionContainedResolved)),
				OnDidntWork: next("tree4.s2"),
				OnUnclear:   selfRef("tree4.s1"),
			},
			// 2. "Mono audio" accessibility toggle off?
			"tree4.s2": {
				ID: "tree4.s2", Ordinal: 2, ReadAloudKey: "tree4.s2", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree4.s2.resolved", DispositionContainedResolved)),
				OnDidntWork: next("tree4.s3"),
				OnUnclear:   selfRef("tree4.s2"),
			},
			// 3. Stereo test outside the softphone.
			//    Both ears work → call audio is mono — expected for voice calls;
			//    reassure (expectation set). Still one-sided → hardware suspected.
			"tree4.s3": {
				ID: "tree4.s3", Ordinal: 3, ReadAloudKey: "tree4.s3", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree4.s3.resolved", DispositionExpectationSet)),
				OnDidntWork: next("tree4.s4"),
				OnUnclear:   selfRef("tree4.s3"),
			},
			// 4. Wiggle test / cable + connector / different port.
			//    Audio cuts with movement → failing cable/connector → RMA.
			//    Consistently dead in one ear → failed earcup driver → RMA.
			"tree4.s4": {
				ID: "tree4.s4", Ordinal: 4, ReadAloudKey: "tree4.s4", KBDocRef: kb,
				OnWorked:    term(rmaT("tree4.s4.rma_cable")),
				OnDidntWork: term(rmaT("tree4.s4.rma_earcup")),
				OnUnclear:   selfRef("tree4.s4"),
			},
		},
	}
}

// buildTree5 — Tree 5: Distorted, Choppy, Robotic, Crackling, or Echoing.
func buildTree5() *Tree {
	const kb = KBDocRef("trees/tree-5-distorted-choppy-echo.md")
	return &Tree{
		ID:          Tree5ID,
		Symptom:     SymptomDistortedAudio,
		Title:       "Distorted, Choppy, Robotic, Crackling, or Echoing Audio",
		Goal:        "robotic/choppy usually = network or CPU; crackling/static usually = USB/driver/port; echo usually = acoustic loop or sidetone — branch on which one",
		EntryStepID: "tree5.s1",
		Steps: map[StepID]*Step{
			// 1. Which best describes it? (3-way classification fork.)
			//    OnWorked/OnDidntWork alias to the most-common branch
			//    (robotic/choppy → network path) when no branch keyword matches.
			"tree5.s1": {
				ID: "tree5.s1", Ordinal: 1, ReadAloudKey: "tree5.s1", KBDocRef: kb,
				Branches: map[string]*StepRef{
					BranchRobotic:   {StepID: "tree5.s2"},
					BranchCrackling: {StepID: "tree5.s3"},
					BranchEcho:      {StepID: "tree5.s4"},
				},
				OnWorked:    next("tree5.s2"),
				OnDidntWork: next("tree5.s2"),
				OnUnclear:   selfRef("tree5.s1"),
			},
			// 2. Robotic/choppy → network/CPU: wired over Wi-Fi, off VPN, free
			//    CPU, Genesys "Test your media settings", restart app/browser (§4).
			//    Persists on a good wired connection → beyond the desk.
			"tree5.s2": {
				ID: "tree5.s2", Ordinal: 2, ReadAloudKey: "tree5.s2", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree5.s2.resolved", DispositionContainedResolved)),
				OnDidntWork: term(escalateT(ReasonGenesysPlatform, PriorityMedium, "tree5.s2.escalate")),
				OnUnclear:   selfRef("tree5.s2"),
			},
			// 3. Crackling/static → USB/driver: reseat off dock, different port,
			//    disable enhancements, lower sample rate, reinstall driver (§2).
			//    Persists on multiple ports after driver reinstall → hardware → RMA.
			"tree5.s3": {
				ID: "tree5.s3", Ordinal: 3, ReadAloudKey: "tree5.s3", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree5.s3.resolved", DispositionContainedResolved)),
				OnDidntWork: term(rmaT("tree5.s3.rma")),
				OnUnclear:   selfRef("tree5.s3"),
			},
			// 4. Echo — who hears it? (2-way fork.)
			//    User hears own voice → that's sidetone → Tree 6 step 3.
			//    Other party hears it → s4b. Alias default: other party (the
			//    §1 text treats other-party echo as the actionable local path).
			"tree5.s4": {
				ID: "tree5.s4", Ordinal: 4, ReadAloudKey: "tree5.s4", KBDocRef: kb,
				Branches: map[string]*StepRef{
					BranchSelf:       {TreeID: Tree6ID, StepID: "tree6.s3"},
					BranchOtherParty: {StepID: "tree5.s4b"},
				},
				OnWorked:    next("tree5.s4b"),
				OnDidntWork: next("tree5.s4b"),
				OnUnclear:   selfRef("tree5.s4"),
			},
			// 4b. Other-party echo: confirm headset not open speakers, lower
			//     speaker volume, echo cancellation on, correct mic+speaker in
			//     Genesys so AEC works (§4). Persists with a proper headset →
			//     beyond the desk.
			"tree5.s4b": {
				ID: "tree5.s4b", Ordinal: 5, ReadAloudKey: "tree5.s4b", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree5.s4b.resolved", DispositionContainedResolved)),
				OnDidntWork: term(escalateT(ReasonGenesysPlatform, PriorityMedium, "tree5.s4b.escalate")),
				OnUnclear:   selfRef("tree5.s4b"),
			},
		},
	}
}

// buildTree6 — Tree 6: Volume Too Low / Too Loud / Sidetone.
func buildTree6() *Tree {
	const kb = KBDocRef("trees/tree-6-volume-sidetone.md")
	return &Tree{
		ID:          Tree6ID,
		Symptom:     SymptomVolumeSidetone,
		Title:       "Volume Too Low / Too Loud / Sidetone",
		Goal:        "two different problems live here: call volume (the other party) and sidetone (the user's own voice) — branch first",
		EntryStepID: "tree6.s1",
		Steps: map[StepID]*Step{
			// 1. Other person's volume, or hearing your own voice? (2-way fork;
			//    alias default: call volume — the more common complaint.)
			"tree6.s1": {
				ID: "tree6.s1", Ordinal: 1, ReadAloudKey: "tree6.s1", KBDocRef: kb,
				Branches: map[string]*StepRef{
					BranchOtherParty: {StepID: "tree6.s2"},
					BranchSelf:       {StepID: "tree6.s3"},
				},
				OnWorked:    next("tree6.s2"),
				OnDidntWork: next("tree6.s2"),
				OnUnclear:   selfRef("tree6.s1"),
			},
			// 2. Call volume: inline dial, Windows volume + mixer, Properties >
			//    Levels, Loudness Equalization, app output + call volume (§2/§4).
			//    Max volume still too quiet on all apps after a driver update →
			//    suspect hardware/driver → escalate (tree exhausted).
			"tree6.s2": {
				ID: "tree6.s2", Ordinal: 2, ReadAloudKey: "tree6.s2", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree6.s2.resolved", DispositionContainedResolved)),
				OnDidntWork: term(escalateT(ReasonTreeExhausted, PriorityMedium, "tree6.s2.escalate")),
				OnUnclear:   selfRef("tree6.s2"),
			},
			// 3. Sidetone: vendor-app sidetone control (Poly Lens / Logi Tune /
			//    Jabra Direct, §3) lower/raise; Windows "Listen to this device"
			//    UNCHECKED (§2). Worked also covers the no-sidetone-feature model
			//    (expected behavior, expectation set in the prompt copy).
			//    Feature present but control has no effect → firmware/driver →
			//    escalate via §3 / Tier 2.
			"tree6.s3": {
				ID: "tree6.s3", Ordinal: 3, ReadAloudKey: "tree6.s3", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree6.s3.resolved", DispositionContainedResolved)),
				OnDidntWork: term(escalateT(ReasonGenesysPlatform, PriorityMedium, "tree6.s3.escalate")),
				OnUnclear:   selfRef("tree6.s3"),
			},
		},
	}
}

// buildTree7 — Tree 7: Mute Out of Sync / Call-Control Buttons Not Working.
func buildTree7() *Tree {
	const kb = KBDocRef("trees/tree-7-mute-sync-buttons.md")
	return &Tree{
		ID:          Tree7ID,
		Symptom:     SymptomMuteCallControl,
		Title:       "Mute Out of Sync / Call-Control Buttons Not Working",
		Goal:        "Genesys gives audio to any USB headset but supports call-control buttons only on specific models — confirm support before chasing a bug",
		EntryStepID: "tree7.s1",
		Steps: map[StepID]*Step{
			// 1. Buttons dead, or mute out of sync? (2-way fork; alias default:
			//    buttons — listed first in §1.)
			"tree7.s1": {
				ID: "tree7.s1", Ordinal: 1, ReadAloudKey: "tree7.s1", KBDocRef: kb,
				Branches: map[string]*StepRef{
					BranchButtons: {StepID: "tree7.s2"},
					BranchDesync:  {StepID: "tree7.s5"},
				},
				OnWorked:    next("tree7.s2"),
				OnDidntWork: next("tree7.s2"),
				OnUnclear:   selfRef("tree7.s1"),
			},
			// 2. Model on the supported call-control list for Genesys (§4/§3)?
			//    Supported → continue. Not supported → expected behavior: user
			//    answers/mutes in the app; audio still works (expectation set).
			"tree7.s2": {
				ID: "tree7.s2", Ordinal: 2, ReadAloudKey: "tree7.s2", KBDocRef: kb,
				OnWorked:    next("tree7.s3"),
				OnDidntWork: term(resolvedT("tree7.s2.resolved", DispositionExpectationSet)),
				OnUnclear:   selfRef("tree7.s2"),
			},
			// 3. Vendor software installed and running (Poly Lens / Jabra Direct /
			//    Logi Tune / EPOS Connect), headset connected in it (§3)?
			"tree7.s3": {
				ID: "tree7.s3", Ordinal: 3, ReadAloudKey: "tree7.s3", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree7.s3.resolved", DispositionContainedResolved)),
				OnDidntWork: next("tree7.s4"),
				OnUnclear:   selfRef("tree7.s3"),
			},
			// 4. Restart the link: re-plug direct, restart vendor app, restart
			//    Genesys app / refresh browser tab so call control re-handshakes (§4).
			//    Supported model + vendor app + restarted, still dead → Tier 2.
			"tree7.s4": {
				ID: "tree7.s4", Ordinal: 4, ReadAloudKey: "tree7.s4", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree7.s4.resolved", DispositionContainedResolved)),
				OnDidntWork: term(escalateT(ReasonGenesysPlatform, PriorityMedium, "tree7.s4.escalate")),
				OnUnclear:   selfRef("tree7.s4"),
			},
			// 5. Mute desync: re-sync by toggling once on each side to a known
			//    state, then a fresh test call.
			"tree7.s5": {
				ID: "tree7.s5", Ordinal: 5, ReadAloudKey: "tree7.s5", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree7.s5.resolved", DispositionContainedResolved)),
				OnDidntWork: next("tree7.s6"),
				OnUnclear:   selfRef("tree7.s5"),
			},
			// 6. Vendor app present + supported model + restart vendor app and
			//    Genesys to rebuild the link (§3/§4). Sync still drifts on a
			//    supported model with current vendor software → Tier 2
			//    (firmware/middleware; prompt copy carries the hot-mic safety
			//    call-out: trust the in-app mute indicator until fixed).
			"tree7.s6": {
				ID: "tree7.s6", Ordinal: 6, ReadAloudKey: "tree7.s6", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree7.s6.resolved", DispositionContainedResolved)),
				OnDidntWork: term(escalateT(ReasonGenesysPlatform, PriorityMedium, "tree7.s6.escalate")),
				OnUnclear:   selfRef("tree7.s6"),
			},
		},
	}
}

// buildTree8 — Tree 8: Intermittent Disconnects / Audio Drops.
func buildTree8() *Tree {
	const kb = KBDocRef("trees/tree-8-intermittent-disconnects.md")
	return &Tree{
		ID:          Tree8ID,
		Symptom:     SymptomIntermittentDrops,
		Title:       "Intermittent Disconnects / Audio Drops",
		Goal:        "the classic root cause is the dock/hub/USB port plus Windows power management putting the USB device to sleep — work that first",
		EntryStepID: "tree8.s1",
		Steps: map[StepID]*Step{
			// 1. Get off the dock/hub — plug directly into the computer.
			"tree8.s1": {
				ID: "tree8.s1", Ordinal: 1, ReadAloudKey: "tree8.s1", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree8.s1.resolved", DispositionContainedResolved)),
				OnDidntWork: next("tree8.s2"),
				OnUnclear:   selfRef("tree8.s1"),
			},
			// 2. Different direct USB port (different side/controller, A vs C).
			"tree8.s2": {
				ID: "tree8.s2", Ordinal: 2, ReadAloudKey: "tree8.s2", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree8.s2.resolved", DispositionContainedResolved)),
				OnDidntWork: next("tree8.s3"),
				OnUnclear:   selfRef("tree8.s2"),
			},
			// 3. Disable USB power management / selective suspend; update
			//    USB/chipset drivers and dock firmware (§2).
			"tree8.s3": {
				ID: "tree8.s3", Ordinal: 3, ReadAloudKey: "tree8.s3", KBDocRef: kb,
				OnWorked:    term(resolvedT("tree8.s3.resolved", DispositionContainedResolved)),
				OnDidntWork: next("tree8.s4"),
				OnUnclear:   selfRef("tree8.s3"),
			},
			// 4. Rule out cable/headset: wiggle test, different cable, different
			//    computer. Movement triggers drops or drops on another machine →
			//    hardware/cable → RMA. Only drops on this one machine after all
			//    of the above → Tier 2 / IT (host USB controller, dock firmware,
			//    or driver beyond the desk) — tree exhausted, not RMA, because
			//    the hardware tested good elsewhere.
			"tree8.s4": {
				ID: "tree8.s4", Ordinal: 4, ReadAloudKey: "tree8.s4", KBDocRef: kb,
				OnWorked:    term(rmaT("tree8.s4.rma")),
				OnDidntWork: term(escalateT(ReasonTreeExhausted, PriorityMedium, "tree8.s4.escalate")),
				OnUnclear:   selfRef("tree8.s4"),
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Referential-integrity validation (schema invariants, triage-design.md §2)
// ---------------------------------------------------------------------------

// ValidateFork checks every schema invariant from docs/triage-design.md:
//  1. every SymptomClass in AllSymptomClasses has a tree;
//  2. every Step defines all three outcome transitions;
//  3. every Transition sets exactly one of NextStep / Terminal;
//  4. every NextStep / RouteTo / Branches StepRef resolves to a real step
//     (routing sentinels excepted — they resolve at runtime);
//  5. Terminal.Kind constrains which fields are set.
func ValidateFork(f *SymptomFork) error {
	if f == nil || f.PreFlight == nil {
		return fmt.Errorf("triage: fork or pre-flight tree is nil")
	}
	trees := map[string]*Tree{f.PreFlight.ID: f.PreFlight}
	for _, class := range AllSymptomClasses {
		t, ok := f.Trees[class]
		if !ok || t == nil {
			return fmt.Errorf("triage: symptom class %q has no tree", class)
		}
		trees[t.ID] = t
	}

	resolve := func(from string, ref *StepRef) error {
		if ref == nil {
			return fmt.Errorf("triage: %s: nil StepRef", from)
		}
		switch ref.TreeID {
		case RouteSymptomTree, RouteOriginalTree:
			return nil // runtime sentinel — resolved from TriageState.Symptom
		}
		treeID := ref.TreeID
		target, ok := trees[treeID]
		if treeID == "" {
			return fmt.Errorf("triage: %s: empty TreeID in resolved ref (caller must fill)", from)
		}
		if !ok {
			return fmt.Errorf("triage: %s: unknown tree %q", from, treeID)
		}
		if _, ok := target.Steps[ref.StepID]; !ok {
			return fmt.Errorf("triage: %s: unknown step %q in tree %q", from, ref.StepID, treeID)
		}
		return nil
	}

	checkTransition := func(from string, tr Transition, ownTree string) error {
		hasNext := tr.NextStep != nil
		hasTerm := tr.Terminal != nil
		if hasNext == hasTerm {
			return fmt.Errorf("triage: %s: transition must set exactly one of NextStep/Terminal", from)
		}
		if hasNext {
			ref := *tr.NextStep
			if ref.TreeID == "" {
				ref.TreeID = ownTree
			}
			return resolve(from, &ref)
		}
		t := tr.Terminal
		switch t.Kind {
		case TerminalKindResolved:
			if t.Disposition == "" {
				return fmt.Errorf("triage: %s: resolved terminal missing disposition", from)
			}
		case TerminalKindEscalate, TerminalKindRMA:
			if t.Reason == "" || t.Priority == "" {
				return fmt.Errorf("triage: %s: %s terminal missing reason/priority", from, t.Kind)
			}
		case TerminalKindRouteToTree:
			if t.RouteTo == nil {
				return fmt.Errorf("triage: %s: route_to_tree terminal missing RouteTo", from)
			}
			switch t.RouteTo.TreeID {
			case RouteSymptomTree, RouteOriginalTree:
				return nil
			}
			return resolve(from, t.RouteTo)
		default:
			return fmt.Errorf("triage: %s: unknown terminal kind %q", from, t.Kind)
		}
		return nil
	}

	for _, tree := range trees {
		if tree.EntryStepID == "" {
			return fmt.Errorf("triage: tree %q has no entry step", tree.ID)
		}
		if _, ok := tree.Steps[tree.EntryStepID]; !ok {
			return fmt.Errorf("triage: tree %q entry step %q not found", tree.ID, tree.EntryStepID)
		}
		for id, step := range tree.Steps {
			if step == nil || step.ID != id {
				return fmt.Errorf("triage: tree %q: step key %q does not match step ID", tree.ID, id)
			}
			loc := fmt.Sprintf("%s/%s", tree.ID, id)
			if step.ReadAloudKey == "" {
				return fmt.Errorf("triage: %s: missing ReadAloudKey", loc)
			}
			if err := checkTransition(loc+"/worked", step.OnWorked, tree.ID); err != nil {
				return err
			}
			if err := checkTransition(loc+"/didnt_work", step.OnDidntWork, tree.ID); err != nil {
				return err
			}
			if err := checkTransition(loc+"/unclear", step.OnUnclear, tree.ID); err != nil {
				return err
			}
			for key, ref := range step.Branches {
				branchRef := *ref
				if branchRef.TreeID == "" {
					branchRef.TreeID = tree.ID
				}
				if err := resolve(fmt.Sprintf("%s/branch[%s]", loc, key), &branchRef); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
