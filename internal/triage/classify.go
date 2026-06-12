// Symptom classifier (B-04): maps a user utterance plus any filled Lex slots
// (connection_type / brand / issue_type) onto exactly one of the eight
// SymptomClass values from the Symptom-to-Tree Quick Index.
//
// Strategy — deterministic FIRST:
//  1. Slot mapping: a filled issue_type slot resolves directly (no NLP).
//  2. Keyword scoring: weighted phrase matching against the quick-index
//     vocabulary; a strict highest score wins.
//  3. Optional LLM fallback: when keywords are ambiguous (no match or a tie),
//     a single call to the injected ClassifierLLM (Haiku via Converse in
//     production, a mock in tests). This package NEVER calls Bedrock itself.
//
// Quick-index utterance → tree mapping (tested in classify_test.go):
//
//	1 "I can't hear anything / there's no sound in my headset."        → no_audio_output   (Tree 1)
//	2 "They can't hear me / my mic isn't working."                     → mic_not_working   (Tree 2)
//	3 "My headset isn't showing up at all / not detected."             → not_detected      (Tree 3)
//	4 "Sound's only in one ear / it's mono."                           → one_sided_audio   (Tree 4)
//	5 "Audio is choppy, robotic, crackly, or echoing."                 → distorted_audio   (Tree 5)
//	6 "It's too quiet / too loud / I can too loudly hear myself."      → volume_sidetone   (Tree 6)
//	7 "Mute is out of sync / my answer/end/mute buttons don't work."   → mute_call_control (Tree 7)
//	8 "It keeps cutting out / dropping / disconnecting."               → intermittent_drops (Tree 8)
package triage

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrUnclassified is returned when neither slots nor keywords identify a
// symptom and no LLM fallback is available (or the LLM also failed). There is
// deliberately no catch-all class — the handler should re-ask the user.
var ErrUnclassified = errors.New("triage: utterance did not match a symptom class")

// ClassifierLLM is the single optional LLM fallback used when keyword
// classification is ambiguous. Production wires a Bedrock Haiku Converse
// client; tests inject a mock. Implementations must return one of the eight
// AllSymptomClasses values (validated by the classifier).
type ClassifierLLM interface {
	// ClassifySymptom maps a free-form utterance to a SymptomClass.
	ClassifySymptom(ctx context.Context, utterance string) (SymptomClass, error)
}

// Slots carries the Lex slot values that may already identify the symptom
// (B-07). ConnectionType and Brand do not determine a symptom class by
// themselves; they are accepted for interface completeness and used only as
// telemetry. IssueType, when filled, resolves the class deterministically.
type Slots struct {
	ConnectionType string // e.g. "usb" — not symptom-determining
	Brand          string // e.g. "jabra" — not symptom-determining
	IssueType      string // e.g. "no_audio", "mic_not_working" — maps directly
}

// Classifier performs deterministic slot/keyword classification with an
// optional injected LLM fallback for ambiguous utterances.
type Classifier struct {
	llm ClassifierLLM // may be nil (deterministic-only)
}

// NewClassifier builds a Classifier. llm may be nil to disable the fallback.
func NewClassifier(llm ClassifierLLM) *Classifier { return &Classifier{llm: llm} }

// ---------------------------------------------------------------------------
// Slot mapping (deterministic, checked first)
// ---------------------------------------------------------------------------

// slotIssueMap maps normalized issue_type slot values (lowercased, spaces and
// hyphens folded to underscores) to a SymptomClass. The eight canonical
// SymptomClass strings map to themselves.
var slotIssueMap = map[string]SymptomClass{
	// canonical class names
	string(SymptomNoAudioOutput):     SymptomNoAudioOutput,
	string(SymptomMicNotWorking):     SymptomMicNotWorking,
	string(SymptomNotDetected):       SymptomNotDetected,
	string(SymptomOneSidedAudio):     SymptomOneSidedAudio,
	string(SymptomDistortedAudio):    SymptomDistortedAudio,
	string(SymptomVolumeSidetone):    SymptomVolumeSidetone,
	string(SymptomMuteCallControl):   SymptomMuteCallControl,
	string(SymptomIntermittentDrops): SymptomIntermittentDrops,
	// common slot synonyms
	"no_audio":       SymptomNoAudioOutput,
	"no_sound":       SymptomNoAudioOutput,
	"cant_hear":      SymptomNoAudioOutput,
	"mic":            SymptomMicNotWorking,
	"microphone":     SymptomMicNotWorking,
	"mic_issue":      SymptomMicNotWorking,
	"detection":      SymptomNotDetected,
	"not_recognized": SymptomNotDetected,
	"not_showing_up": SymptomNotDetected,
	"one_sided":      SymptomOneSidedAudio,
	"one_ear":        SymptomOneSidedAudio,
	"mono":           SymptomOneSidedAudio,
	"distorted":      SymptomDistortedAudio,
	"choppy":         SymptomDistortedAudio,
	"robotic":        SymptomDistortedAudio,
	"crackling":      SymptomDistortedAudio,
	"echo":           SymptomDistortedAudio,
	"volume":         SymptomVolumeSidetone,
	"sidetone":       SymptomVolumeSidetone,
	"too_quiet":      SymptomVolumeSidetone,
	"too_loud":       SymptomVolumeSidetone,
	"mute":           SymptomMuteCallControl,
	"mute_sync":      SymptomMuteCallControl,
	"buttons":        SymptomMuteCallControl,
	"call_control":   SymptomMuteCallControl,
	"drops":          SymptomIntermittentDrops,
	"disconnects":    SymptomIntermittentDrops,
	"cutting_out":    SymptomIntermittentDrops,
	"intermittent":   SymptomIntermittentDrops,
}

func normalizeSlot(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = strings.ReplaceAll(v, " ", "_")
	v = strings.ReplaceAll(v, "-", "_")
	return v
}

// ---------------------------------------------------------------------------
// Keyword vocabulary (the symptom-index phrases, weighted)
// ---------------------------------------------------------------------------

type scoredPhrase struct {
	phrase string
	weight int
}

// symptomVocab is the quick-index vocabulary. Phrases are matched as
// lowercase substrings; weights favor the more diagnostic phrases so a mixed
// utterance lands on the dominant symptom.
var symptomVocab = map[SymptomClass][]scoredPhrase{
	SymptomNoAudioOutput: {
		{"can't hear anything", 5}, {"cannot hear anything", 5}, {"can not hear anything", 5},
		{"no sound", 5}, {"no audio", 4}, {"hear nothing", 4}, {"nothing in my headset", 4},
		{"can't hear them", 4}, {"can't hear the caller", 4}, {"can't hear the customer", 4},
		{"there's no sound", 5}, {"dead silence", 3}, {"silent", 2},
	},
	SymptomMicNotWorking: {
		{"can't hear me", 5}, {"cannot hear me", 5}, {"can not hear me", 5},
		{"they can't hear", 5}, {"nobody can hear me", 5}, {"no one can hear me", 5},
		{"mic isn't working", 5}, {"mic not working", 5}, {"mic doesn't work", 5},
		{"microphone", 3}, {"my mic", 3}, {"not picking up my voice", 4},
	},
	SymptomNotDetected: {
		{"not detected", 5}, {"isn't detected", 5},
		{"not showing up", 5}, {"isn't showing up", 5}, {"doesn't show up", 4},
		{"not recognized", 4}, {"isn't recognized", 4}, {"doesn't appear", 4},
		{"not listed", 3}, {"no headset found", 4}, {"windows doesn't see", 4},
	},
	SymptomOneSidedAudio: {
		{"one ear", 5}, {"one side", 5}, {"one-sided", 5},
		{"only the left", 4}, {"only the right", 4}, {"left ear only", 4}, {"right ear only", 4},
		{"it's mono", 4}, {"in mono", 4}, {"mono", 3},
	},
	SymptomDistortedAudio: {
		{"choppy", 5}, {"robotic", 5}, {"crackl", 5}, {"static", 4},
		{"distort", 4}, {"garbled", 4}, {"underwater", 4},
		{"echo", 4}, {"popping", 3}, {"buzzing", 3},
	},
	SymptomVolumeSidetone: {
		{"too quiet", 5}, {"too loud", 5}, {"hear myself", 5}, {"sidetone", 5},
		{"my own voice", 4}, {"hear my own", 4}, {"too soft", 4},
		{"volume", 3}, {"hard to hear them", 3},
	},
	SymptomMuteCallControl: {
		{"out of sync", 4}, {"mute", 4}, {"unmute", 3},
		{"buttons don't work", 5}, {"buttons do nothing", 5}, {"button", 3},
		{"answer/end", 4}, {"call control", 4}, {"answer button", 4}, {"end button", 4},
	},
	SymptomIntermittentDrops: {
		{"cutting out", 5}, {"cuts out", 5}, {"keeps cutting", 5},
		{"dropping", 5}, {"keeps dropping", 5}, {"audio drops", 4}, {"drops out", 4},
		{"disconnect", 5}, {"keeps disconnecting", 5}, {"intermittent", 4},
	},
}

// scoreSymptoms returns each class's keyword score for the utterance.
func scoreSymptoms(utterance string) map[SymptomClass]int {
	low := strings.ToLower(utterance)
	scores := make(map[SymptomClass]int, len(symptomVocab))
	for class, phrases := range symptomVocab {
		for _, p := range phrases {
			if strings.Contains(low, p.phrase) {
				scores[class] += p.weight
			}
		}
	}
	return scores
}

// ---------------------------------------------------------------------------
// Classify — slots → keywords → optional LLM fallback
// ---------------------------------------------------------------------------

// Classify resolves the symptom class for an utterance plus any filled slots.
// Order: issue_type slot (deterministic) → keyword scoring (deterministic,
// strict winner required) → injected LLM fallback (single call) → ErrUnclassified.
func (c *Classifier) Classify(ctx context.Context, utterance string, slots Slots) (SymptomClass, error) {
	// 1. Slot value wins outright.
	if v := normalizeSlot(slots.IssueType); v != "" {
		if class, ok := slotIssueMap[v]; ok {
			return class, nil
		}
	}

	// 2. Keyword scoring: require a strict winner with a positive score.
	scores := scoreSymptoms(utterance)
	best, second := SymptomClass(""), 0
	bestScore := 0
	for _, class := range AllSymptomClasses { // deterministic iteration order
		s := scores[class]
		if s > bestScore {
			second = bestScore
			best, bestScore = class, s
		} else if s > second {
			second = s
		}
	}
	if bestScore > 0 && bestScore > second {
		return best, nil
	}

	// 3. Ambiguous (no match or tie) → single LLM fallback if available.
	if c.llm != nil && strings.TrimSpace(utterance) != "" {
		class, err := c.llm.ClassifySymptom(ctx, utterance)
		if err != nil {
			return "", fmt.Errorf("triage: classifier LLM fallback failed: %w", err)
		}
		for _, valid := range AllSymptomClasses {
			if class == valid {
				return class, nil
			}
		}
		return "", fmt.Errorf("triage: classifier LLM returned invalid class %q: %w", class, ErrUnclassified)
	}

	return "", ErrUnclassified
}

// ---------------------------------------------------------------------------
// Branch sub-classification (the n-way fork steps, triage-design.md §7)
// ---------------------------------------------------------------------------

// branchVocab holds, per fork step, the weighted phrase vocabulary for each
// branch key. The engine calls ClassifyBranch with the raw utterance; a strict
// highest score picks the branch. More-specific branches (e.g. "self"/sidetone
// phrasing) carry heavier weights than generic ones.
var branchVocab = map[StepID]map[string][]scoredPhrase{
	// Tree 5 step 1 — which best describes the distortion?
	"tree5.s1": {
		BranchRobotic: {
			{"robotic", 5}, {"choppy", 5}, {"underwater", 5},
			{"cutting in and out", 5}, {"cuts in and out", 5}, {"garbled", 4}, {"breaking up", 4},
		},
		BranchCrackling: {
			{"crackl", 5}, {"static", 5}, {"popping", 4}, {"pops", 3}, {"buzz", 4}, {"hiss", 3},
		},
		BranchEcho: {
			{"echo", 5}, {"repeat of", 4}, {"hear it twice", 4}, {"voice repeated", 4},
		},
	},
	// Tree 5 step 4 — who hears the echo?
	"tree5.s4": {
		BranchSelf: {
			{"hear myself", 5}, {"my own voice", 5}, {"i hear my", 5},
			{"hear my voice", 5}, {"hearing myself", 5}, {"i do", 3}, {"me", 1},
		},
		BranchOtherParty: {
			{"they hear", 5}, {"they're hearing", 5}, {"other party", 5}, {"other person", 5},
			{"caller", 4}, {"customer", 4}, {"the other side", 4}, {"they do", 3}, {"them", 2},
		},
	},
	// Tree 6 step 1 — other party's volume, or hearing your own voice?
	"tree6.s1": {
		BranchSelf: {
			{"hear myself", 5}, {"hearing myself", 5}, {"my own voice", 5}, {"sidetone", 5},
			{"hear my own", 5}, {"can't hear myself", 5}, {"myself", 4},
		},
		BranchOtherParty: {
			{"other person", 5}, {"other party", 5}, {"they sound", 4}, {"caller", 4},
			{"customer", 4}, {"too quiet", 4}, {"too loud", 4}, {"can't hear them", 5},
			{"hard to hear", 4}, {"volume", 3},
		},
	},
	// Tree 7 step 1 — buttons dead, or mute out of sync?
	"tree7.s1": {
		BranchButtons: {
			{"button", 5}, {"buttons", 5}, {"answer", 4}, {"end call", 4},
			{"do nothing", 4}, {"don't work", 3}, {"doesn't work", 3}, {"not working", 3}, {"dead", 3},
		},
		BranchDesync: {
			{"out of sync", 5}, {"desync", 5}, {"drift", 4}, {"disagree", 4},
			{"shows muted", 5}, {"shows unmuted", 5}, {"says i'm muted", 5},
			{"still muted", 4}, {"still shows", 3}, {"mismatch", 4},
		},
	},
}

// branchOrder gives a stable iteration order per fork step (map iteration is
// randomized; ties must resolve deterministically to "no match").
var branchOrder = map[StepID][]string{
	"tree5.s1": {BranchRobotic, BranchCrackling, BranchEcho},
	"tree5.s4": {BranchSelf, BranchOtherParty},
	"tree6.s1": {BranchSelf, BranchOtherParty},
	"tree7.s1": {BranchButtons, BranchDesync},
}

// ClassifyBranch sub-classifies the user's characterization on an n-way fork
// step. Returns (branchKey, true) on a strict keyword winner, ("", false)
// when the utterance is ambiguous or matches nothing — the engine then falls
// back to the step's Outcome transitions (alias to the most-common branch).
func ClassifyBranch(stepID StepID, utterance string) (string, bool) {
	vocab, ok := branchVocab[stepID]
	if !ok || strings.TrimSpace(utterance) == "" {
		return "", false
	}
	low := strings.ToLower(utterance)
	best, bestScore, second := "", 0, 0
	for _, key := range branchOrder[stepID] {
		score := 0
		for _, p := range vocab[key] {
			if strings.Contains(low, p.phrase) {
				score += p.weight
			}
		}
		if score > bestScore {
			second = bestScore
			best, bestScore = key, score
		} else if score > second {
			second = score
		}
	}
	if bestScore > 0 && bestScore > second {
		return best, true
	}
	return "", false
}
