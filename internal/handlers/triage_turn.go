// Triage turn driver (B-05): wires the deterministic triage engine (B-04) into
// the Lex fulfillment handler with a "did it work?" verify loop and durable
// step-state.
//
// Division of labor (docs/triage-design.md / triage/types.go agent contract):
//   - The triage ENGINE owns navigation: which step, transitions, terminals,
//     and counters (failed_steps, reboot_count). It performs no I/O.
//   - The BEDROCK AGENT only answers free-form side questions (grounded in the
//     step's KB doc) via the injected FreeFormAnswerer. It never navigates.
//   - This driver parses the user's reply into a TurnResult Outcome, drives
//     exactly one engine turn, renders the StepView from the prompt catalog
//     (prompts.go) so the flow keeps working even when Bedrock is unavailable,
//     and persists TriageState via the session accessors.
//
// Per-turn flow: LoadTriageState(session) → classify (first turn) or
// ParseOutcome + Engine.Advance → render StepView / terminal → SaveTriageState
// (which also mirrors the keys into the Lex session attributes so the echoed
// attrs never go stale-wins on the next merge).
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"unicode"

	"github.com/headset-support-agent/internal/logging"
	"github.com/headset-support-agent/internal/models"
	"github.com/headset-support-agent/internal/session"
	"github.com/headset-support-agent/internal/triage"
)

// Session-attribute keys owned by the triage turn driver (not part of the
// triage.Attr* Lex-mirror set or the session.Key* set).
const (
	// KeyAwaitingSymptom is "true" while the pre-flight handoff is blocked on
	// symptom classification (triage.ErrSymptomRequired): the next utterance
	// is fed to the classifier instead of the outcome parser.
	KeyAwaitingSymptom = "awaiting_symptom"
	// KeyDisposition records the terminal disposition code (E-8.1 taxonomy)
	// when a Resolved terminal fires, e.g. "contained_resolved".
	KeyDisposition = "disposition"
	// KeyTriageTree / KeyTriageStep carry the exact tree and step the flow
	// stopped at into the escalation attributes for the Connect warm-transfer
	// context (OPS-2), alongside escalation_reason / escalation_priority set
	// by BuildEscalationResponse and attempted_steps from the state mirror.
	KeyTriageTree = "triage_tree"
	KeyTriageStep = "triage_step"
)

// symptomClarifyPrompt is asked when pre-flight hands off but no symptom has
// been classified yet (ErrSymptomRequired). The options mirror the quick index
// so the keyword classifier can resolve the reply deterministically.
const symptomClarifyPrompt = "I want to make sure we go down the right path. Which best describes it: no sound at all; the other party can't hear you; the headset isn't detected; sound in only one ear; choppy, crackly, or echoing audio; a volume issue or hearing your own voice; mute or call buttons misbehaving; or it keeps cutting out?"

// FreeFormAnswerer answers a side question in persona voice, grounded in the
// given KB doc reference. Production wires a closure over the Bedrock agent
// client; tests inject a stub; nil disables free-form side answers (the turn
// then falls back to the engine's unclear re-prompt — graceful degrade).
type FreeFormAnswerer func(ctx context.Context, sessionID, question string, kbDoc triage.KBDocRef) (string, error)

// TriageDeps bundles the dependencies HandleTriageTurn needs. Engine and
// Classifier are required; FreeForm is optional.
type TriageDeps struct {
	Engine     *triage.Engine
	Classifier *triage.Classifier
	FreeForm   FreeFormAnswerer
}

// ---------------------------------------------------------------------------
// Outcome parsing — transcript → worked / didnt_work / unclear
// ---------------------------------------------------------------------------

// didntWorkPhrases are substrings that signal the step did not help. They are
// checked BEFORE workedPhrases so "still not working" never matches a positive.
var didntWorkPhrases = []string{
	"didn't work", "did not work", "didnt work", "doesn't work", "does not work",
	"didn't help", "did not help", "didn't fix", "did not fix", "didn't do anything",
	"didn't change", "hasn't worked", "hasn't helped", "isn't working", "is not working",
	"not working", "not fixed", "isn't fixed", "no change", "no difference", "no luck",
	"nothing happened", "nothing changed", "still not", "still no", "still the same",
	"still broken", "still doesn't", "still does not", "still can't", "still cannot",
	"still silent", "still nothing", "still muted", "still quiet", "still too quiet",
	"still cutting", "still dropping", "still drops", "still happening", "still there",
	"same problem", "same issue", "same thing", "made it worse", "even worse",
	"still won't", "won't work", "no good", "not helping",
	"already correct", "already set", "already selected", "already there",
	"already done that", "already did that", "already unmuted", "already off",
	"already on", "already up", "it's already",
}

// workedPhrases are substrings that signal the step fixed the problem (or that
// a diagnostic check passed).
var workedPhrases = []string{
	"that worked", "it worked", "it works", "works now", "working now", "it's working",
	"that did it", "did the trick", "that fixed", "fixed it", "it's fixed", "is fixed",
	"problem solved", "solved", "sorted", "all good", "good now", "all set",
	"can hear them now", "hear them now", "i can hear them", "they can hear me",
	"sounds good", "sounds normal", "back to normal", "much better", "that helped",
	"shows up now", "showing up now", "detected now", "it appears now", "connected now",
	"stopped dropping", "no more drops", "in sync now",
}

// yesTokens / noTokens are bare-word verdicts, matched per token so "eyes"
// never matches "yes".
var yesTokens = map[string]bool{
	"yes": true, "yeah": true, "yep": true, "yup": true, "aye": true,
	"correct": true, "affirmative": true, "definitely": true, "absolutely": true,
}

var noTokens = map[string]bool{
	"no": true, "nope": true, "nah": true, "negative": true,
}

// ParseOutcome maps the user's transcript onto the engine's three outcomes:
// negative phrases first ("no/didn't/still/nope" → didnt_work), then positive
// phrases ("yes/worked/fixed/that did it" → worked), then bare yes/no tokens;
// anything else is unclear (the engine re-prompts once, then treats the next
// unclear answer as didnt_work).
func ParseOutcome(transcript string) triage.Outcome {
	low := strings.ToLower(strings.TrimSpace(transcript))
	if low == "" {
		return triage.OutcomeUnclear
	}
	for _, p := range didntWorkPhrases {
		if strings.Contains(low, p) {
			return triage.OutcomeDidntWork
		}
	}
	for _, p := range workedPhrases {
		if strings.Contains(low, p) {
			return triage.OutcomeWorked
		}
	}
	sawYes, sawNo := false, false
	for _, tok := range tokenize(low) {
		if yesTokens[tok] {
			sawYes = true
		}
		if noTokens[tok] {
			sawNo = true
		}
	}
	switch {
	case sawYes && !sawNo:
		return triage.OutcomeWorked
	case sawNo && !sawYes:
		return triage.OutcomeDidntWork
	}
	return triage.OutcomeUnclear
}

func tokenize(low string) []string {
	return strings.FieldsFunc(low, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '\''
	})
}

// repeatPhrases ask for the current step to be replayed without advancing.
// Deliberately specific ("repeat that", not bare "repeat") so echo
// descriptions like "they hear a repeat of my voice" don't trigger a replay.
var repeatPhrases = []string{
	"repeat that", "can you repeat", "please repeat", "say that again",
	"say it again", "come again", "one more time", "didn't catch that",
	"did not catch that", "what was that",
}

func isRepeatRequest(low string) bool {
	for _, p := range repeatPhrases {
		if strings.Contains(low, p) {
			return true
		}
	}
	return false
}

// questionPrefixes mark an utterance as a free-form side question (only
// consulted when the outcome parse came back unclear).
var questionPrefixes = []string{
	"how do i", "how do you", "how can i", "how would i",
	"where is", "where's", "where do i", "where do you",
	"what is", "what's", "what does", "what do you mean",
	"can you explain", "explain", "i don't know how", "i'm not sure how",
}

func isFreeFormQuestion(low string) bool {
	if strings.HasSuffix(low, "?") {
		return true
	}
	for _, p := range questionPrefixes {
		if strings.HasPrefix(low, p) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// TriageState <-> session round-trip
// ---------------------------------------------------------------------------

// LoadTriageState rebuilds the engine's TriageState from the session
// attributes (the B-02 store is the source of truth).
func LoadTriageState(sess *models.Session) *triage.TriageState {
	attempted := session.GetAttemptedSteps(sess)
	steps := make([]triage.StepID, len(attempted))
	for i, s := range attempted {
		steps[i] = triage.StepID(s)
	}
	return &triage.TriageState{
		CurrentTree:       session.GetCurrentTree(sess),
		CurrentStep:       triage.StepID(session.GetCurrentStep(sess)),
		Symptom:           triage.SymptomClass(session.GetString(sess, session.KeySymptom)),
		AttemptedSteps:    steps,
		FailedSteps:       session.GetFailedSteps(sess),
		FrustrationCount:  session.GetFrustrationCount(sess),
		RebootCount:       session.GetRebootCount(sess),
		DriverReinstalled: session.GetDriverReinstalled(sess),
		UnclearStreak:     session.GetUnclearStreak(sess),
		LastResponse:      session.GetLastResponse(sess),
		Resolved:          session.GetBool(sess, session.KeyResolved),
		Escalated:         session.GetBool(sess, session.KeyEscalated),
		EscalationReason:  triage.EscalationReason(session.GetString(sess, session.KeyEscalationReason)),
	}
}

// triageMirrorKeys is the attribute set SaveTriageState writes; the same keys
// are mirrored into the per-turn Lex attributes.
var triageMirrorKeys = []string{
	session.KeyCurrentTree, session.KeyCurrentStep, session.KeySymptom,
	session.KeyAttemptedSteps, session.KeyFailedSteps, session.KeyFrustrationCount,
	session.KeyRebootCount, session.KeyDriverReinstalled, session.KeyUnclearStreak,
	session.KeyLastResponse, session.KeyResolved, session.KeyEscalated,
	session.KeyEscalationReason,
}

// SaveTriageState writes the TriageState back onto the session attributes and
// mirrors the same keys into the Lex turn attributes (mirror may be nil).
// Mirroring matters: the load-merge lets incoming Lex attrs win on collision,
// so any key updated this turn MUST be refreshed in the response attrs or the
// stale echoed copy would clobber the store value next turn.
func SaveTriageState(sess *models.Session, st *triage.TriageState, mirror map[string]string) {
	session.SetCurrentTree(sess, st.CurrentTree)
	session.SetCurrentStep(sess, string(st.CurrentStep))
	session.SetString(sess, session.KeySymptom, string(st.Symptom))
	steps := make([]string, len(st.AttemptedSteps))
	for i, s := range st.AttemptedSteps {
		steps[i] = string(s)
	}
	session.SetAttemptedSteps(sess, steps)
	session.SetFailedSteps(sess, st.FailedSteps)
	session.SetFrustrationCount(sess, st.FrustrationCount)
	session.SetRebootCount(sess, st.RebootCount)
	session.SetDriverReinstalled(sess, st.DriverReinstalled)
	session.SetUnclearStreak(sess, st.UnclearStreak)
	session.SetLastResponse(sess, st.LastResponse)
	session.SetBool(sess, session.KeyResolved, st.Resolved)
	session.SetBool(sess, session.KeyEscalated, st.Escalated)
	session.SetString(sess, session.KeyEscalationReason, string(st.EscalationReason))
	if mirror != nil {
		for _, k := range triageMirrorKeys {
			mirror[k] = sess.Attributes[k]
		}
	}
}

func setAwaitingSymptom(sess *models.Session, attrs map[string]string, v bool) {
	session.SetBool(sess, KeyAwaitingSymptom, v)
	attrs[KeyAwaitingSymptom] = sess.Attributes[KeyAwaitingSymptom]
}

// ---------------------------------------------------------------------------
// HandleTriageTurn — one engine turn end to end
// ---------------------------------------------------------------------------

// HandleTriageTurn drives one triage engine turn for the given transcript.
// It returns (response, true) when the turn was handled by the triage flow,
// or (zero, false) when the turn should fall through to the generic Bedrock
// agent path (no symptom classified yet, flow already ended, or an internal
// navigation error). The caller persists the session afterwards.
func HandleTriageTurn(ctx context.Context, deps TriageDeps, sess *models.Session, transcript string, p *models.Persona, attrs map[string]string) (LexV2Response, bool) {
	if deps.Engine == nil || deps.Classifier == nil {
		return LexV2Response{}, false
	}
	st := LoadTriageState(sess)
	if st.Resolved || st.Escalated {
		slog.Debug("triage: flow already ended — deferring to agent",
			slog.String("session_id", sess.SessionID))
		return LexV2Response{}, false
	}
	low := strings.ToLower(strings.TrimSpace(transcript))

	// First substantive turn: classify the symptom and enter the flow at the
	// Universal Pre-Flight Checklist. An unclassifiable opener defers to the
	// generic agent (no tree guessing — there is no catch-all class).
	if st.CurrentTree == "" {
		class, err := deps.Classifier.Classify(ctx, transcript, triage.Slots{})
		if err != nil {
			slog.Debug("triage: opening utterance unclassified — deferring to agent",
				slog.String("session_id", sess.SessionID))
			return LexV2Response{}, false
		}
		if serr := deps.Engine.SetSymptom(st, class); serr != nil {
			slog.Warn("triage: SetSymptom failed — deferring to agent",
				slog.String("session_id", sess.SessionID),
				slog.String("error", serr.Error()))
			return LexV2Response{}, false
		}
		view := deps.Engine.Start(st)
		slog.Info("triage: flow started",
			slog.String("session_id", sess.SessionID),
			slog.String("symptom", string(class)),
			slog.String("tree", view.TreeID),
			slog.String("step", string(view.StepID)))
		text := "I can definitely help with that. Let's run a few quick checks first — they fix most headset issues in a couple of minutes. " + stepPrompt(view)
		return finishStep(sess, st, text, p, attrs), true
	}

	// "Repeat that" replays the current step without advancing or counting.
	if isRepeatRequest(low) {
		if view, err := deps.Engine.View(st); err == nil {
			text := st.LastResponse
			if text == "" {
				text = stepPrompt(view)
			}
			return finishStep(sess, st, text, p, attrs), true
		}
	}

	// Pre-flight handed off without a classified symptom: this utterance is
	// the answer to the "which best describes it" clarifier.
	if session.GetBool(sess, KeyAwaitingSymptom) {
		class, err := deps.Classifier.Classify(ctx, transcript, triage.Slots{})
		if err != nil {
			st.LastResponse = symptomClarifyPrompt
			SaveTriageState(sess, st, attrs)
			return BuildSuccessResponse(p, symptomClarifyPrompt, attrs), true
		}
		setAwaitingSymptom(sess, attrs, false)
		prevTree := st.CurrentTree
		view, serr := deps.Engine.SelectSymptomTree(st, class)
		if serr != nil {
			slog.Warn("triage: SelectSymptomTree failed — deferring to agent",
				slog.String("session_id", sess.SessionID),
				slog.String("error", serr.Error()))
			return LexV2Response{}, false
		}
		return respondToView(sess, st, view, p, attrs, prevTree), true
	}

	outcome := ParseOutcome(transcript)
	slog.Info("triage: advancing",
		slog.String("session_id", sess.SessionID),
		slog.String("tree", st.CurrentTree),
		slog.String("step", string(st.CurrentStep)),
		slog.String("outcome", string(outcome)))
	slog.Debug("triage: turn transcript",
		slog.String("session_id", sess.SessionID),
		slog.String("transcript", logging.Truncate(transcript, 80)))

	// Side question: Bedrock answers grounded in the step's KB doc and the
	// engine re-presents the same step (FreeFormHandled — no counters move).
	// If Bedrock is unavailable or fails, fall through to the engine's normal
	// unclear handling so the flow keeps working.
	if outcome == triage.OutcomeUnclear && deps.FreeForm != nil && isFreeFormQuestion(low) {
		if view, verr := deps.Engine.View(st); verr == nil {
			answer, ferr := deps.FreeForm(ctx, sess.SessionID, transcript, view.KBDoc)
			if ferr == nil && strings.TrimSpace(answer) != "" {
				if v2, aerr := deps.Engine.Advance(st, triage.TurnResult{FreeFormHandled: true, RawUtterance: transcript}); aerr == nil {
					text := strings.TrimSpace(answer) + " So — " + stepPrompt(v2)
					return finishStep(sess, st, text, p, attrs), true
				}
			}
			if ferr != nil {
				slog.Warn("triage: free-form answer failed — using unclear handling",
					slog.String("session_id", sess.SessionID),
					slog.String("error", ferr.Error()))
			}
		}
	}

	prevTree := st.CurrentTree
	view, err := deps.Engine.Advance(st, triage.TurnResult{Outcome: outcome, RawUtterance: transcript})
	if errors.Is(err, triage.ErrSymptomRequired) {
		// Counters for this turn were already applied — do NOT re-Advance.
		// Try classifying this same utterance; otherwise ask the clarifier.
		class, cerr := deps.Classifier.Classify(ctx, transcript, triage.Slots{})
		if cerr != nil {
			setAwaitingSymptom(sess, attrs, true)
			st.LastResponse = symptomClarifyPrompt
			SaveTriageState(sess, st, attrs)
			return BuildSuccessResponse(p, symptomClarifyPrompt, attrs), true
		}
		view, err = deps.Engine.SelectSymptomTree(st, class)
	}
	if err != nil {
		slog.Warn("triage: advance failed — deferring to agent",
			slog.String("session_id", sess.SessionID),
			slog.String("error", err.Error()))
		SaveTriageState(sess, st, attrs) // persist any counters that moved
		return LexV2Response{}, false
	}
	return respondToView(sess, st, view, p, attrs, prevTree), true
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

// stepPrompt renders a non-terminal StepView from the prompt catalog. The
// catalog-completeness test guarantees coverage; the fallback is defensive.
func stepPrompt(view triage.StepView) string {
	if text, ok := PromptText(view.ReadAloud); ok {
		return text
	}
	slog.Warn("triage: missing prompt copy for step",
		slog.String("read_aloud_key", string(view.ReadAloud)))
	return "Let's keep going with the next check. Try it, and then tell me — did that fix it, yes or no?"
}

// finishStep persists the state (recording the rendered text for "repeat")
// and builds the ElicitIntent response for a non-terminal step.
func finishStep(sess *models.Session, st *triage.TriageState, text string, p *models.Persona, attrs map[string]string) LexV2Response {
	st.LastResponse = text
	SaveTriageState(sess, st, attrs)
	return BuildSuccessResponse(p, text, attrs)
}

// respondToView turns the engine's StepView into a Lex response: terminal
// views close or escalate; ordinary steps re-elicit with the step copy.
func respondToView(sess *models.Session, st *triage.TriageState, view triage.StepView, p *models.Persona, attrs map[string]string, prevTree string) LexV2Response {
	if view.IsTerminal {
		return respondTerminal(sess, st, view, p, attrs)
	}
	text := stepPrompt(view)
	if st.UnclearStreak > 0 {
		// The engine re-asked the same step after an unclear answer.
		text = "Sorry — I just need a quick yes or no on that one. " + text
	} else if prevTree != "" && view.TreeID != prevTree {
		text = "Alright — let's dig into that specific problem. " + text
	}
	return finishStep(sess, st, text, p, attrs)
}

// respondTerminal handles Resolved (Close) and Escalate/RMA terminals. The
// escalation path reuses BuildEscalationResponse so the Connect transfer
// (OPS-2) sees the same escalation_requested / escalation_reason /
// escalation_priority attributes as the keyword and frustration detectors,
// plus triage_tree / triage_step / attempted_steps for the warm handoff.
func respondTerminal(sess *models.Session, st *triage.TriageState, view triage.StepView, p *models.Persona, attrs map[string]string) LexV2Response {
	t := view.Terminal
	if t.Kind == triage.TerminalKindResolved {
		text := promptOrDefault(t.ReadAloudKey, "Great — that fixed it! You're all set. Thanks for calling, and have a great day.")
		session.SetString(sess, KeyDisposition, t.Disposition)
		attrs[KeyDisposition] = t.Disposition
		st.LastResponse = text
		SaveTriageState(sess, st, attrs)
		slog.Info("triage: resolved",
			slog.String("session_id", sess.SessionID),
			slog.String("tree", view.TreeID),
			slog.String("step", string(view.StepID)),
			slog.String("disposition", t.Disposition))
		return BuildCloseResponse(p, text, attrs)
	}

	priority := string(t.Priority)
	if priority == "" {
		priority = string(triage.PriorityMedium)
	}
	attrs[KeyTriageTree] = view.TreeID
	attrs[KeyTriageStep] = string(view.StepID)
	decision := &models.EscalationDecision{
		ShouldEscalate: true,
		Reason:         string(t.Reason),
		Priority:       priority,
	}
	resp := BuildEscalationResponse(p, decision, attrs)
	// Prefer the terminal's specific handoff copy over the generic persona
	// escalation line when the catalog has it.
	if text, ok := PromptText(t.ReadAloudKey); ok {
		st.LastResponse = text
		resp.Messages = []Message{{ContentType: "SSML", Content: BuildSSML(text, p)}}
	}
	SaveTriageState(sess, st, attrs)
	slog.Info("triage: escalated",
		slog.String("session_id", sess.SessionID),
		slog.String("kind", string(t.Kind)),
		slog.String("reason", string(t.Reason)),
		slog.String("priority", priority),
		slog.String("tree", view.TreeID),
		slog.String("step", string(view.StepID)))
	return resp
}
