// Package triage defines the trees-as-data schema for the Headset Support
// Agent's deterministic troubleshooting engine.
//
// Design stance (WS-B): the eight diagnostic trees in troubleshooting.md §1,
// the Universal Pre-Flight Checklist, and the "When to Escalate" criteria are
// DATA, not prose to improvise over. They are encoded here as a state machine.
//
// Division of responsibility (the "agent contract"):
//   - The triage engine OWNS navigation: which tree, which step, which
//     transition fires on a given outcome, and which terminal is reached.
//     It is the single source of truth for "where are we in troubleshooting."
//   - The Bedrock agent ONLY renders/explains the current step's content in
//     persona voice and answers free-form questions grounded in the KB. It
//     never decides the next step; it reports back an outcome and the engine
//     advances.
//
// This file contains TYPE DEFINITIONS, small enums, and consts only. It
// deliberately contains no engine logic and no tree data (those are B-04, in
// engine.go / trees.go). Session truth persists in SessionTable keyed by the
// Lex sessionId (= Connect ContactId) and is mirrored into Lex session
// attributes each turn; see TriageState and the attribute-key consts below.
package triage

// ---------------------------------------------------------------------------
// Symptom fork — the top level that maps the 8 quick-index symptoms plus the
// universal pre-flight checklist that always runs first.
// ---------------------------------------------------------------------------

// SymptomClass identifies one of the eight quick-index symptoms (Tree 1..8)
// plus the synthetic pre-flight phase that runs before any tree. It is the
// output of the B-04 symptom classifier and the key into the top-level
// symptom fork (SymptomFork).
type SymptomClass string

const (
	// SymptomPreFlight is the Universal Pre-Flight Checklist. It is not one of
	// the eight trees; it always runs first and, if unresolved, hands off to
	// the tree selected from the user's symptom.
	SymptomPreFlight SymptomClass = "preflight"

	// SymptomNoAudioOutput — Tree 1. "I can't hear anything / no sound."
	SymptomNoAudioOutput SymptomClass = "no_audio_output"
	// SymptomMicNotWorking — Tree 2 (most common). "They can't hear me."
	SymptomMicNotWorking SymptomClass = "mic_not_working"
	// SymptomNotDetected — Tree 3. "My headset isn't showing up at all."
	SymptomNotDetected SymptomClass = "not_detected"
	// SymptomOneSidedAudio — Tree 4. "Sound's only in one ear / it's mono."
	SymptomOneSidedAudio SymptomClass = "one_sided_audio"
	// SymptomDistortedAudio — Tree 5. "Choppy, robotic, crackly, or echoing."
	SymptomDistortedAudio SymptomClass = "distorted_audio"
	// SymptomVolumeSidetone — Tree 6. "Too quiet/loud / hearing myself."
	SymptomVolumeSidetone SymptomClass = "volume_sidetone"
	// SymptomMuteCallControl — Tree 7. "Mute out of sync / buttons dead."
	SymptomMuteCallControl SymptomClass = "mute_call_control"
	// SymptomIntermittentDrops — Tree 8. "It keeps cutting out / dropping."
	SymptomIntermittentDrops SymptomClass = "intermittent_drops"
)

// AllSymptomClasses lists every classifiable symptom, in quick-index order,
// excluding the always-first pre-flight phase. The B-04 classifier must map
// each of the eight index utterances onto exactly one of these — there is no
// "misc"/catch-all class.
var AllSymptomClasses = []SymptomClass{
	SymptomNoAudioOutput,
	SymptomMicNotWorking,
	SymptomNotDetected,
	SymptomOneSidedAudio,
	SymptomDistortedAudio,
	SymptomVolumeSidetone,
	SymptomMuteCallControl,
	SymptomIntermittentDrops,
}

// SymptomFork is the top-level routing table. PreFlight always runs first;
// when it does not resolve the issue, the engine selects Trees[class] using
// the classified SymptomClass. Every SymptomClass in AllSymptomClasses MUST
// have an entry in Trees (validated at engine init in B-04).
type SymptomFork struct {
	// PreFlight is the Universal Pre-Flight Checklist tree, run before any
	// symptom tree. Its terminals either resolve the ticket or route into a
	// symptom tree (TerminalKindRouteToTree).
	PreFlight *Tree
	// Trees maps each of the eight symptom classes to its diagnostic tree.
	Trees map[SymptomClass]*Tree
}

// ---------------------------------------------------------------------------
// Tree / Step / Transition / Terminal — the trees-as-data schema.
// ---------------------------------------------------------------------------

// Tree is one diagnostic flow: an ordered set of steps with a stable entry
// point. Each tree corresponds to a §1 tree (or the pre-flight checklist).
type Tree struct {
	// ID is the stable identifier for this tree, e.g. "tree1" or "preflight".
	ID string
	// Symptom is the symptom class this tree services (SymptomPreFlight for
	// the pre-flight checklist).
	Symptom SymptomClass
	// Title is the human/agent-facing name, e.g. "No Audio Output".
	Title string
	// Goal is the one-line framing from the tree header, used by the agent to
	// orient the user (e.g. "get sound into the user's ears").
	Goal string
	// EntryStepID is the StepID where navigation begins for this tree.
	EntryStepID StepID
	// Steps holds every step in the tree, keyed by StepID. The engine resolves
	// transitions by looking up the target StepID here.
	Steps map[StepID]*Step
}

// StepID is a stable, human-readable identifier for a step, unique within its
// tree (e.g. "tree1.s2"). Stability matters: attempted_steps in TriageState
// stores StepIDs, and Lex attributes / logs reference them across turns.
type StepID string

// Step is a single decision point read aloud to the user. Steps are sequenced
// cheapest-and-most-common-fix-first, mirroring the source guide.
//
// Navigation is fully deterministic: after the agent reports an Outcome, the
// engine consults the matching Transition (OnWorked / OnDidntWork / OnUnclear)
// and either moves to NextStep or fires a Terminal. There is no implicit
// fall-through and no catch-all: every Step must define behavior for all three
// outcomes (even if OnUnclear just re-asks the same step).
type Step struct {
	// ID is this step's stable identifier (Tree.Steps key).
	ID StepID
	// Ordinal is the 1-based position shown to the user/agent ("step 3"),
	// matching the numbered list in the source tree.
	Ordinal int

	// ReadAloudKey references the prompt text for this step. It is a key into
	// the read-aloud prompt catalog (the verbatim branch text from §1), NOT the
	// text itself — keeping data and copy separable and letting the agent
	// fetch persona-rendered phrasing. See PromptRef.
	ReadAloudKey PromptRef

	// KBDocRef optionally names the knowledge-base doc that explains this step
	// in click-by-click detail (the §2/§3/§4 deep references). Empty when the
	// step is self-contained. The agent uses this to ground free-form answers.
	KBDocRef KBDocRef

	// Outcome transitions. Exactly the three user-reportable outcomes are
	// modeled; the engine requires all three to be set on every step.
	OnWorked    Transition // user reports the step fixed the problem
	OnDidntWork Transition // user reports it did not help / still broken
	OnUnclear   Transition // ambiguous/off-script answer; usually re-ask once
}

// Outcome is what the engine expects back from the agent each turn after a
// step has been read: the user's verdict on "did that fix it?".
type Outcome string

const (
	// OutcomeWorked — the fix resolved the issue ("yes, fixed").
	OutcomeWorked Outcome = "worked"
	// OutcomeDidntWork — the fix did not help ("no, still broken").
	OutcomeDidntWork Outcome = "didnt_work"
	// OutcomeUnclear — the answer was ambiguous or off-script; B-05 re-prompts
	// once, then treats a second unclear answer as OutcomeDidntWork.
	OutcomeUnclear Outcome = "unclear"
)

// Transition is the deterministic edge taken for a given outcome. Exactly one
// of NextStep or Terminal is set (validated in B-04):
//   - NextStep set  => advance within the tree (or jump to another tree via a
//     StepRef that names a different tree's entry).
//   - Terminal set  => the flow ends (resolved or escalate/RMA).
type Transition struct {
	// NextStep, when set, is the step to navigate to next. It may reference a
	// step in the same tree or the entry of another tree (cross-tree jumps,
	// e.g. Tree 1 step 1 "If NO -> Tree 3").
	NextStep *StepRef
	// Terminal, when set, ends the flow with a resolved/escalate/RMA outcome.
	Terminal *Terminal
}

// StepRef points at a step, optionally in a different tree. When TreeID is
// empty the reference is within the current tree.
type StepRef struct {
	// TreeID is the target tree; empty means "this tree".
	TreeID string
	// StepID is the target step. When jumping cross-tree without a specific
	// step, set it to the target Tree.EntryStepID.
	StepID StepID
}

// ---------------------------------------------------------------------------
// Terminals — resolved, escalate-with-reason (incl. the "≥2 reboots / no
// progress" rule), and RMA. Every leaf of every §1 tree maps to one of these.
// ---------------------------------------------------------------------------

// TerminalKind classifies how a flow ends.
type TerminalKind string

const (
	// TerminalKindResolved — the issue was fixed; route to a "contained /
	// resolved" disposition and Close.
	TerminalKindResolved TerminalKind = "resolved"
	// TerminalKindEscalate — hand off to a human / Tier 2 with a reason.
	TerminalKindEscalate TerminalKind = "escalate"
	// TerminalKindRMA — suspected hardware fault / defective unit; open an
	// RMA/replacement case. (A specialized escalation kept distinct so the
	// engine and handoff can branch on it.)
	TerminalKindRMA TerminalKind = "rma"
	// TerminalKindRouteToTree — pre-flight (or a cross-cutting branch) hands
	// navigation to a symptom tree rather than ending. The engine continues
	// into RouteTo rather than closing the contact.
	TerminalKindRouteToTree TerminalKind = "route_to_tree"
)

// EscalationReason enumerates the reasons a flow escalates. The first three
// mirror internal/handlers/escalation.go so the engine and the keyword/
// frustration detector share one vocabulary; the remainder encode the
// tree-driven "When to Escalate" criteria from §1 so terminals carry a precise
// reason into the warm-transfer handoff (OPS-2). No catch-all reason exists.
type EscalationReason string

const (
	// --- shared with internal/handlers/escalation.go ---
	ReasonUserRequested            EscalationReason = "user_requested"
	ReasonUserFrustrated           EscalationReason = "user_frustrated"
	ReasonTroubleshootingExhausted EscalationReason = "troubleshooting_exhausted"

	// --- tree-driven "When to Escalate" criteria (§1) ---

	// ReasonHardwareFault — symptom follows the headset across ports/machines,
	// audio cuts with movement, an earcup/mic is consistently dead, or
	// detection fails everywhere. Typically paired with TerminalKindRMA.
	ReasonHardwareFault EscalationReason = "hardware_fault"
	// ReasonRebootLimit — the "≥ 2 reboots / no progress" rule: rebooted twice
	// and reinstalled the driver with no change. Stop looping; escalate.
	ReasonRebootLimit EscalationReason = "reboot_limit"
	// ReasonTreeExhausted — reached the end of a tree (including a reboot and a
	// driver reinstall) and the symptom remains.
	ReasonTreeExhausted EscalationReason = "tree_exhausted"
	// ReasonManagedMachinePolicy — locked-down/managed machine, no admin
	// rights, USB whitelisting/policy blocks the change. Route to Tier 2 / IT.
	ReasonManagedMachinePolicy EscalationReason = "managed_machine_policy"
	// ReasonUserCannotPerform — the user is unable/uncomfortable performing
	// Device Manager / power-setting / driver steps.
	ReasonUserCannotPerform EscalationReason = "user_cannot_perform"
	// ReasonAccessibilityNeed — an accommodation is required (mono audio,
	// amplification, assistive device, slower pace). Route to an equipped path.
	ReasonAccessibilityNeed EscalationReason = "accessibility_need"
	// ReasonGenesysPlatform — symptom persists on a good wired connection with
	// the headset confirmed good at the Windows layer (WebRTC/codec, call
	// control on a supported model, persistent echo, mute desync). Route to
	// Tier 2 / Genesys admin via §4.
	ReasonGenesysPlatform EscalationReason = "genesys_platform"
)

// EscalationPriority maps to the priority field on models.EscalationDecision.
type EscalationPriority string

const (
	PriorityHigh   EscalationPriority = "high"
	PriorityMedium EscalationPriority = "medium"
	PriorityLow    EscalationPriority = "low"
)

// Terminal is a leaf of the state machine. Its Kind dictates which other
// fields are meaningful (validated in B-04):
//   - Resolved      => Disposition set; Reason/Priority/RouteTo empty.
//   - Escalate, RMA => Reason set (and usually Priority); Disposition optional.
//   - RouteToTree   => RouteTo set; Reason/Disposition empty.
type Terminal struct {
	// Kind classifies the terminal.
	Kind TerminalKind

	// Reason is the escalation reason (for Escalate/RMA terminals). It feeds
	// models.EscalationDecision.Reason and the warm-transfer context.
	Reason EscalationReason
	// Priority is the escalation priority (for Escalate/RMA terminals).
	Priority EscalationPriority

	// Disposition is the terminal disposition code (E-8.1 taxonomy), e.g.
	// "contained_resolved". Set for Resolved terminals; optional otherwise.
	Disposition string

	// RouteTo is the target tree/step for TerminalKindRouteToTree (pre-flight
	// handing off to a symptom tree, or a cross-tree redirect).
	RouteTo *StepRef

	// ReadAloudKey references the closing/handoff line to read to the user
	// (e.g. the clean-handoff summary for an escalation). May be empty for a
	// pure RouteToTree.
	ReadAloudKey PromptRef
}

// ---------------------------------------------------------------------------
// References — keep copy and KB grounding out of the data graph.
// ---------------------------------------------------------------------------

// PromptRef is a stable key into the read-aloud prompt catalog. The catalog
// holds the verbatim §1 branch text; the agent renders it in persona voice.
// Storing a key (not the text) keeps the schema small, lets WS-E swap pacing/
// phrasing, and keeps the source-of-truth copy in one place.
type PromptRef string

// KBDocRef names a knowledge-base doc (the A-01 retrieval-sized splits, e.g.
// "windows/2.4-mic-privacy" or "genesys/4.3-browser-mic-permission") that
// explains a step's click-by-click detail. The agent grounds free-form answers
// against this doc. Empty when a step needs no deep reference.
type KBDocRef string

// ---------------------------------------------------------------------------
// Agent contract — the per-turn data exchanged between engine and agent.
// ---------------------------------------------------------------------------

// StepView is what the engine hands the Bedrock agent each turn: a read-only
// description of the current step. The agent renders ReadAloud in persona voice
// and may pull KBDoc to answer follow-ups, but it does NOT choose what comes
// next — it returns a TurnResult and the engine advances.
type StepView struct {
	TreeID     string    // current tree
	TreeTitle  string    // e.g. "Microphone / Other Party Can't Hear Me"
	Goal       string    // tree goal line, for orientation
	StepID     StepID    // current step
	Ordinal    int       // "step N"
	TotalSteps int       // steps in the current tree, for "step N of M"
	ReadAloud  PromptRef // prompt-catalog key the agent renders
	KBDoc      KBDocRef  // KB doc for grounding free-form answers (may be empty)
	IsTerminal bool      // true once a Terminal has fired (engine sets the close/handoff line)
	Terminal   *Terminal // populated when IsTerminal; agent reads its ReadAloudKey
}

// TurnResult is what the engine expects back from the agent (after the user
// responds): the classified Outcome of the step the user was just asked about,
// plus light telemetry. The engine maps Outcome -> Transition deterministically.
type TurnResult struct {
	// Outcome is the user's verdict: worked / didnt_work / unclear. This is the
	// only field that drives navigation.
	Outcome Outcome
	// FreeFormHandled is true if the agent answered a side question this turn
	// (grounded in KBDoc) without the user yet giving a worked/didnt verdict;
	// the engine then re-presents the same step rather than advancing.
	FreeFormHandled bool
	// RawUtterance is the user's transcript, retained for logging/telemetry and
	// for the escalation/frustration detector. Not used for navigation.
	RawUtterance string
}

// ---------------------------------------------------------------------------
// TriageState — the per-session navigation state persisted to SessionTable and
// mirrored into Lex session attributes each turn.
// ---------------------------------------------------------------------------

// TriageState is the durable navigation state for one contact. It is stored in
// SessionTable keyed by the Lex sessionId (= Connect ContactId) by the B-01
// session store and mirrored into Lex session attributes each turn (see the
// Attr* key consts) so the stateless Lambda can resume mid-tree.
//
// The struct/db tags match the typed-accessor field list in B-01 so the
// session store and triage engine agree on the schema.
type TriageState struct {
	// CurrentTree is the active tree ID ("preflight" until a symptom tree is
	// selected). Empty before pre-flight begins.
	CurrentTree string `json:"current_tree" dynamodbav:"current_tree"`
	// CurrentStep is the active StepID within CurrentTree.
	CurrentStep StepID `json:"current_step" dynamodbav:"current_step"`
	// Symptom is the classified symptom class, once known.
	Symptom SymptomClass `json:"symptom" dynamodbav:"symptom"`

	// AttemptedSteps is the ordered list of StepIDs already presented and tried,
	// for clean handoff (so Tier 2 doesn't repeat them) and to avoid re-loops.
	AttemptedSteps []StepID `json:"attempted_steps" dynamodbav:"attempted_steps"`
	// FailedSteps counts steps that did not resolve the issue; feeds the ≥5
	// troubleshooting-exhausted escalation threshold.
	FailedSteps int `json:"failed_steps" dynamodbav:"failed_steps"`
	// FrustrationCount accumulates the frustration delta across turns; feeds the
	// ≥3 user-frustrated escalation threshold.
	FrustrationCount int `json:"frustration_count" dynamodbav:"frustration_count"`

	// RebootCount tracks full reboots performed this session and DriverReinstalled
	// whether a driver reinstall has been done; together they implement the
	// "≥ 2 reboots / no progress" rule (ReasonRebootLimit).
	RebootCount       int  `json:"reboot_count" dynamodbav:"reboot_count"`
	DriverReinstalled bool `json:"driver_reinstalled" dynamodbav:"driver_reinstalled"`

	// UnclearStreak counts consecutive OutcomeUnclear answers on the current
	// step; B-05 re-prompts once (streak==1) then treats the next as didnt_work.
	UnclearStreak int `json:"unclear_streak" dynamodbav:"unclear_streak"`

	// LastResponse is the last rendered step text, for "repeat"/pace replays.
	LastResponse string `json:"last_response" dynamodbav:"last_response"`

	// Resolved is set true when a Resolved terminal fires.
	Resolved bool `json:"resolved" dynamodbav:"resolved"`
	// Escalated is set true (with EscalationReason) when an Escalate/RMA
	// terminal fires.
	Escalated        bool             `json:"escalated" dynamodbav:"escalated"`
	EscalationReason EscalationReason `json:"escalation_reason" dynamodbav:"escalation_reason"`
}

// Lex session-attribute keys: the subset of TriageState mirrored into Lex
// session attributes each turn so the stateless handler can resume without a
// SessionTable read on the hot path. SessionTable remains the source of truth;
// these are the projection. Keys are snake_case to match existing attributes.
const (
	AttrCurrentTree      = "current_tree"
	AttrCurrentStep      = "current_step"
	AttrSymptom          = "symptom"
	AttrFailedSteps      = "failed_steps"
	AttrFrustrationCount = "frustration_count"
	AttrRebootCount      = "reboot_count"
	AttrEscalationReason = "escalation_reason"
	AttrResolved         = "resolved"
	AttrEscalated        = "escalated"
)

// RebootLimit is the threshold for the "≥ 2 reboots / no progress" rule: at
// this many reboots with a driver reinstall and no change, the engine fires a
// ReasonRebootLimit escalation terminal instead of re-running steps.
const RebootLimit = 2

// FailedStepsEscalationThreshold mirrors the ≥5 failed-steps threshold in
// internal/handlers/escalation.go (troubleshooting_exhausted).
const FailedStepsEscalationThreshold = 5

// FrustrationEscalationThreshold mirrors the ≥3 frustration threshold in
// internal/handlers/escalation.go (user_frustrated).
const FrustrationEscalationThreshold = 3

// UnclearRepromptLimit is how many consecutive unclear answers B-05 tolerates
// (re-prompt once) before treating the next as OutcomeDidntWork.
const UnclearRepromptLimit = 1
