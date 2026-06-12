// Triage engine (B-04): pure, deterministic navigation over the trees-as-data
// schema. Given a TriageState and a TurnResult the engine produces the next
// StepView or fires a Terminal. It performs NO I/O — no Bedrock, no DynamoDB,
// no Lex. The handler (B-05) persists the mutated TriageState via the session
// store and hands StepView to the Bedrock agent for persona rendering.
//
// Cross-cutting rules implemented here (docs/triage-design.md §6):
//   - unclear re-prompts once (UnclearRepromptLimit), then counts as didnt_work;
//   - reboot / driver-reinstall steps increment RebootCount / set
//     DriverReinstalled when they report didnt_work;
//   - the "≥ 2 reboots / no progress" guard fires ReasonRebootLimit instead of
//     entering another reboot/driver step once RebootCount >= RebootLimit and
//     the driver has been reinstalled;
//   - FailedSteps counts failed FIX attempts inside symptom trees (pre-flight
//     checks and characterization forks are not fixes); at
//     FailedStepsEscalationThreshold the engine escalates with
//     ReasonTroubleshootingExhausted instead of continuing;
//   - FrustrationCount >= FrustrationEscalationThreshold (maintained by the
//     B-05 detector) escalates with ReasonUserFrustrated at the next turn.
package triage

import (
	"errors"
	"fmt"
)

// Sentinel errors returned by the engine. The handler branches on these.
var (
	// ErrNotStarted — Advance/View called before Start (empty CurrentTree).
	ErrNotStarted = errors.New("triage: flow not started — call Start first")
	// ErrFlowEnded — the flow already reached a Resolved/Escalate/RMA terminal.
	ErrFlowEnded = errors.New("triage: flow already ended (resolved or escalated)")
	// ErrSymptomRequired — a $symptom route fired but TriageState.Symptom is
	// not classified yet. The handler must run the Classifier and then call
	// SelectSymptomTree (do NOT re-call Advance with the same TurnResult —
	// counters for that turn were already applied).
	ErrSymptomRequired = errors.New("triage: symptom classification required — classify, then call SelectSymptomTree")
)

// Engine navigates a validated SymptomFork. It is stateless and safe for
// concurrent use; all per-contact state lives in the caller's TriageState.
type Engine struct {
	fork  *SymptomFork
	trees map[string]*Tree // by Tree.ID, including pre-flight
}

// NewEngine validates the fork's referential integrity (ValidateFork) and
// returns a ready engine. Construction fails loudly on any schema violation.
func NewEngine(fork *SymptomFork) (*Engine, error) {
	if err := ValidateFork(fork); err != nil {
		return nil, err
	}
	trees := map[string]*Tree{fork.PreFlight.ID: fork.PreFlight}
	for _, t := range fork.Trees {
		trees[t.ID] = t
	}
	return &Engine{fork: fork, trees: trees}, nil
}

// NewDefaultEngine builds the engine over BuildSymptomFork(). It panics on a
// validation error, which can only happen if the compiled-in tree data is
// inconsistent (a programming error caught by trees_test.go).
func NewDefaultEngine() *Engine {
	e, err := NewEngine(BuildSymptomFork())
	if err != nil {
		panic(fmt.Sprintf("triage: built-in tree data invalid: %v", err))
	}
	return e
}

// ---------------------------------------------------------------------------
// Flow entry points
// ---------------------------------------------------------------------------

// Start begins (or restarts) the flow at the Universal Pre-Flight Checklist
// and returns the first StepView. The classified symptom, if already known
// (set via SetSymptom), is preserved so the post-pre-flight route resolves
// without another classifier call.
func (e *Engine) Start(state *TriageState) StepView {
	pf := e.fork.PreFlight
	state.CurrentTree = pf.ID
	state.CurrentStep = pf.EntryStepID
	state.UnclearStreak = 0
	return e.viewOf(pf, pf.Steps[pf.EntryStepID])
}

// SetSymptom records the classified symptom without navigating. Call this as
// soon as the Classifier resolves the user's opening utterance (typically
// before/while pre-flight runs) so the $symptom route resolves automatically.
func (e *Engine) SetSymptom(state *TriageState, class SymptomClass) error {
	if _, ok := e.fork.Trees[class]; !ok {
		return fmt.Errorf("triage: unknown symptom class %q", class)
	}
	state.Symptom = class
	return nil
}

// SelectSymptomTree records the classified symptom AND navigates to that
// tree's entry step. The handler calls this after Advance returned
// ErrSymptomRequired (the pre-flight handoff with no symptom yet classified).
func (e *Engine) SelectSymptomTree(state *TriageState, class SymptomClass) (StepView, error) {
	if state.Resolved || state.Escalated {
		return StepView{}, ErrFlowEnded
	}
	if err := e.SetSymptom(state, class); err != nil {
		return StepView{}, err
	}
	tree := e.fork.Trees[class]
	return e.enterStep(state, tree, tree.Steps[tree.EntryStepID])
}

// View re-renders the current step (for "repeat that" / pace replays) without
// mutating state.
func (e *Engine) View(state *TriageState) (StepView, error) {
	if state.Resolved || state.Escalated {
		return StepView{}, ErrFlowEnded
	}
	tree, step, err := e.lookup(state)
	if err != nil {
		return StepView{}, err
	}
	return e.viewOf(tree, step), nil
}

// Escalate fires an escalation/RMA terminal for reasons that originate
// OUTSIDE tree navigation — the B-05 keyword detector (user_requested), the
// frustration detector, user_cannot_perform, accessibility_need — and applies
// it to state. Tree-driven escalations fire from transitions automatically.
func (e *Engine) Escalate(state *TriageState, reason EscalationReason) StepView {
	return e.applyTerminal(state, EscalationTerminal(reason))
}

// EscalationTerminal builds a well-formed Escalate/RMA terminal for any
// EscalationReason: hardware_fault maps to TerminalKindRMA / high priority,
// user_requested escalates at high priority, everything else at medium —
// matching the priority table in docs/triage-design.md §6.
func EscalationTerminal(reason EscalationReason) *Terminal {
	kind := TerminalKindEscalate
	priority := PriorityMedium
	switch reason {
	case ReasonHardwareFault:
		kind = TerminalKindRMA
		priority = PriorityHigh
	case ReasonUserRequested:
		priority = PriorityHigh
	}
	return &Terminal{
		Kind:         kind,
		Reason:       reason,
		Priority:     priority,
		ReadAloudKey: PromptRef("handoff." + string(reason)),
	}
}

// ---------------------------------------------------------------------------
// Advance — the per-turn navigation function
// ---------------------------------------------------------------------------

// Advance consumes the agent's TurnResult for the current step and returns the
// next StepView (which may be terminal). It mutates state (counters, current
// position, resolved/escalated flags); the caller persists it afterwards.
func (e *Engine) Advance(state *TriageState, result TurnResult) (StepView, error) {
	if state.Resolved || state.Escalated {
		return StepView{}, ErrFlowEnded
	}
	tree, step, err := e.lookup(state)
	if err != nil {
		return StepView{}, err
	}

	// The agent answered a side question without a verdict: re-present the
	// same step untouched (no counters move).
	if result.FreeFormHandled {
		return e.viewOf(tree, step), nil
	}

	// Frustration threshold (maintained by the B-05 detector across turns).
	if state.FrustrationCount >= FrustrationEscalationThreshold {
		return e.applyTerminal(state, EscalationTerminal(ReasonUserFrustrated)), nil
	}

	// n-way fork steps: sub-classify the user's characterization first. This
	// runs regardless of Outcome — a characterization ("it sounds robotic")
	// is neither worked nor didnt_work, so the agent typically reports it as
	// unclear; the utterance still picks the branch deterministically.
	if len(step.Branches) > 0 {
		if key, ok := ClassifyBranch(step.ID, result.RawUtterance); ok {
			state.UnclearStreak = 0
			state.AttemptedSteps = append(state.AttemptedSteps, step.ID)
			return e.followRef(state, tree, step.Branches[key])
		}
		// No branch keyword matched → fall through: unclear re-asks the
		// characterization; worked/didnt_work alias to the most-common branch.
	}

	outcome := result.Outcome
	if outcome == OutcomeUnclear {
		state.UnclearStreak++
		if state.UnclearStreak <= UnclearRepromptLimit {
			// Re-prompt once: OnUnclear is a self-reference by construction.
			return e.follow(state, tree, step.OnUnclear)
		}
		// Second consecutive unclear answer counts as didnt_work.
		outcome = OutcomeDidntWork
	}
	state.UnclearStreak = 0
	state.AttemptedSteps = append(state.AttemptedSteps, step.ID)

	var tr Transition
	if outcome == OutcomeWorked {
		tr = step.OnWorked
	} else {
		// A failed FIX attempt inside a symptom tree counts toward the
		// troubleshooting-exhausted threshold. Pre-flight items are quick
		// checks and characterization forks are questions — neither counts.
		if tree.ID != PreflightTreeID && isFixStep(step) {
			state.FailedSteps++
		}
		// The "≥ 2 reboots / no progress" bookkeeping: the user performed the
		// remediation (reboot / driver reinstall) and the symptom remained.
		if IsRebootStep(step.ID) {
			state.RebootCount++
		}
		if IsDriverReinstallStep(step.ID) {
			state.DriverReinstalled = true
		}
		tr = step.OnDidntWork
	}
	return e.follow(state, tree, tr)
}

// isFixStep reports whether a step is a remediation attempt (its YES branch
// resolves the ticket) as opposed to a diagnostic check or fork.
func isFixStep(step *Step) bool {
	return step.OnWorked.Terminal != nil && step.OnWorked.Terminal.Kind == TerminalKindResolved
}

// ---------------------------------------------------------------------------
// Transition resolution
// ---------------------------------------------------------------------------

func (e *Engine) follow(state *TriageState, tree *Tree, tr Transition) (StepView, error) {
	if tr.NextStep != nil {
		return e.followRef(state, tree, tr.NextStep)
	}
	t := tr.Terminal
	if t.Kind == TerminalKindRouteToTree {
		return e.routeToTree(state, t)
	}
	return e.applyTerminal(state, t), nil
}

// followRef resolves a StepRef (same-tree or cross-tree) and enters the step.
func (e *Engine) followRef(state *TriageState, tree *Tree, ref *StepRef) (StepView, error) {
	targetTree := tree
	if ref.TreeID != "" && ref.TreeID != tree.ID {
		t, ok := e.trees[ref.TreeID]
		if !ok {
			return StepView{}, fmt.Errorf("triage: transition references unknown tree %q", ref.TreeID)
		}
		targetTree = t
	}
	step, ok := targetTree.Steps[ref.StepID]
	if !ok {
		return StepView{}, fmt.Errorf("triage: transition references unknown step %q in tree %q", ref.StepID, targetTree.ID)
	}
	return e.enterStep(state, targetTree, step)
}

// routeToTree resolves a TerminalKindRouteToTree, including the runtime
// sentinels, and continues navigation into the target tree (the flow does not
// end — docs/triage-design.md, TerminalKindRouteToTree).
func (e *Engine) routeToTree(state *TriageState, t *Terminal) (StepView, error) {
	ref := t.RouteTo
	switch ref.TreeID {
	case RouteSymptomTree:
		// Pre-flight handoff: the classified symptom selects the tree.
		tree, ok := e.fork.Trees[state.Symptom]
		if !ok {
			return StepView{}, ErrSymptomRequired
		}
		return e.enterStep(state, tree, tree.Steps[tree.EntryStepID])
	case RouteOriginalTree:
		// Tree 3 s1: detection restored → resume the original symptom tree.
		// When detection WAS the complaint (or no other symptom is known),
		// the appearance resolves the ticket.
		tree, ok := e.fork.Trees[state.Symptom]
		if !ok || state.Symptom == SymptomNotDetected {
			return e.applyTerminal(state, resolvedT("tree3.s1.resolved", DispositionContainedResolved)), nil
		}
		return e.enterStep(state, tree, tree.Steps[tree.EntryStepID])
	default:
		tree, ok := e.trees[ref.TreeID]
		if !ok {
			return StepView{}, fmt.Errorf("triage: route_to_tree references unknown tree %q", ref.TreeID)
		}
		stepID := ref.StepID
		if stepID == "" {
			stepID = tree.EntryStepID
		}
		step, ok := tree.Steps[stepID]
		if !ok {
			return StepView{}, fmt.Errorf("triage: route_to_tree references unknown step %q in tree %q", stepID, tree.ID)
		}
		return e.enterStep(state, tree, step)
	}
}

// enterStep applies the cross-cutting escalation guards, then moves the
// state's current position to the step and returns its view.
func (e *Engine) enterStep(state *TriageState, tree *Tree, step *Step) (StepView, error) {
	// "≥ 2 reboots / no progress": never loop back into another reboot/driver
	// remediation once both have been done without progress.
	if isGuardedRemediation(step.ID) && state.RebootCount >= RebootLimit && state.DriverReinstalled {
		return e.applyTerminal(state, EscalationTerminal(ReasonRebootLimit)), nil
	}
	// Troubleshooting exhausted: stop offering more fixes past the threshold.
	// Tree terminals (more specific reasons) are unaffected — this guard only
	// intercepts continuations onto yet another step.
	if state.FailedSteps >= FailedStepsEscalationThreshold {
		return e.applyTerminal(state, EscalationTerminal(ReasonTroubleshootingExhausted)), nil
	}
	state.CurrentTree = tree.ID
	state.CurrentStep = step.ID
	return e.viewOf(tree, step), nil
}

// applyTerminal marks the flow ended on state and builds the terminal view.
func (e *Engine) applyTerminal(state *TriageState, t *Terminal) StepView {
	switch t.Kind {
	case TerminalKindResolved:
		state.Resolved = true
	case TerminalKindEscalate, TerminalKindRMA:
		state.Escalated = true
		state.EscalationReason = t.Reason
	}
	view := StepView{
		TreeID:     state.CurrentTree,
		StepID:     state.CurrentStep,
		ReadAloud:  t.ReadAloudKey,
		IsTerminal: true,
		Terminal:   t,
	}
	if tree, ok := e.trees[state.CurrentTree]; ok {
		view.TreeTitle = tree.Title
		view.Goal = tree.Goal
		view.TotalSteps = len(tree.Steps)
		if step, ok := tree.Steps[state.CurrentStep]; ok {
			view.Ordinal = step.Ordinal
			view.KBDoc = step.KBDocRef
		}
	}
	return view
}

// ---------------------------------------------------------------------------
// Lookup / view helpers
// ---------------------------------------------------------------------------

func (e *Engine) lookup(state *TriageState) (*Tree, *Step, error) {
	if state.CurrentTree == "" {
		return nil, nil, ErrNotStarted
	}
	tree, ok := e.trees[state.CurrentTree]
	if !ok {
		return nil, nil, fmt.Errorf("triage: state references unknown tree %q", state.CurrentTree)
	}
	step, ok := tree.Steps[state.CurrentStep]
	if !ok {
		return nil, nil, fmt.Errorf("triage: state references unknown step %q in tree %q", state.CurrentStep, tree.ID)
	}
	return tree, step, nil
}

func (e *Engine) viewOf(tree *Tree, step *Step) StepView {
	return StepView{
		TreeID:     tree.ID,
		TreeTitle:  tree.Title,
		Goal:       tree.Goal,
		StepID:     step.ID,
		Ordinal:    step.Ordinal,
		TotalSteps: len(tree.Steps),
		ReadAloud:  step.ReadAloudKey,
		KBDoc:      step.KBDocRef,
	}
}
