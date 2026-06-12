package triage

import (
	"context"
	"errors"
	"testing"
)

// mockLLM implements ClassifierLLM for tests (the engine/classifier never
// calls Bedrock directly).
type mockLLM struct {
	class SymptomClass
	err   error
	calls int
}

func (m *mockLLM) ClassifySymptom(_ context.Context, _ string) (SymptomClass, error) {
	m.calls++
	return m.class, m.err
}

// The 8 Symptom-to-Tree Quick Index utterances (troubleshooting.md §1) must
// classify deterministically — no LLM needed.
func TestClassifyQuickIndexUtterances(t *testing.T) {
	cases := []struct {
		utterance string
		want      SymptomClass
	}{
		{"I can't hear anything / there's no sound in my headset.", SymptomNoAudioOutput},
		{"They can't hear me / my mic isn't working.", SymptomMicNotWorking},
		{"My headset isn't showing up at all / not detected.", SymptomNotDetected},
		{"Sound's only in one ear / it's mono.", SymptomOneSidedAudio},
		{"Audio is choppy, robotic, crackly, or echoing.", SymptomDistortedAudio},
		{"It's too quiet / too loud / I can (or can too loudly) hear myself.", SymptomVolumeSidetone},
		{"Mute is out of sync / my answer/end/mute buttons don't work.", SymptomMuteCallControl},
		{"It keeps cutting out / dropping / disconnecting.", SymptomIntermittentDrops},
	}
	llm := &mockLLM{} // present but must NOT be consulted
	c := NewClassifier(llm)
	for _, tc := range cases {
		t.Run(string(tc.want), func(t *testing.T) {
			got, err := c.Classify(context.Background(), tc.utterance, Slots{})
			if err != nil {
				t.Fatalf("Classify(%q): %v", tc.utterance, err)
			}
			if got != tc.want {
				t.Fatalf("Classify(%q) = %q, want %q", tc.utterance, got, tc.want)
			}
		})
	}
	if llm.calls != 0 {
		t.Errorf("LLM fallback consulted %d times on unambiguous index utterances", llm.calls)
	}
}

// Conversational paraphrases keep landing on the right tree.
func TestClassifyParaphrases(t *testing.T) {
	cases := []struct {
		utterance string
		want      SymptomClass
	}{
		{"there's no audio at all in my headset", SymptomNoAudioOutput},
		{"the customer says they can't hear me", SymptomMicNotWorking},
		{"windows doesn't see the headset", SymptomNotDetected},
		{"I only get sound in the left ear only", SymptomOneSidedAudio},
		{"everyone sounds robotic and garbled", SymptomDistortedAudio},
		{"I keep hearing my own voice, the sidetone is awful", SymptomVolumeSidetone},
		{"the mute button and the app are out of sync", SymptomMuteCallControl},
		{"the headset keeps disconnecting every few minutes", SymptomIntermittentDrops},
	}
	c := NewClassifier(nil)
	for _, tc := range cases {
		got, err := c.Classify(context.Background(), tc.utterance, Slots{})
		if err != nil {
			t.Errorf("Classify(%q): %v", tc.utterance, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Classify(%q) = %q, want %q", tc.utterance, got, tc.want)
		}
	}
}

// A filled issue_type slot resolves deterministically and wins outright.
func TestClassifyFromSlots(t *testing.T) {
	cases := []struct {
		issueType string
		want      SymptomClass
	}{
		{"no_audio", SymptomNoAudioOutput},
		{"No Sound", SymptomNoAudioOutput}, // normalization: case + spaces
		{"mic_not_working", SymptomMicNotWorking},
		{"microphone", SymptomMicNotWorking},
		{"not-detected", SymptomNotDetected}, // hyphen folding
		{"one_sided", SymptomOneSidedAudio},
		{"choppy", SymptomDistortedAudio},
		{"sidetone", SymptomVolumeSidetone},
		{"call_control", SymptomMuteCallControl},
		{"disconnects", SymptomIntermittentDrops},
	}
	c := NewClassifier(nil)
	for _, tc := range cases {
		got, err := c.Classify(context.Background(), "", Slots{
			IssueType:      tc.issueType,
			ConnectionType: "usb",   // accepted, not symptom-determining
			Brand:          "jabra", // accepted, not symptom-determining
		})
		if err != nil {
			t.Errorf("slot %q: %v", tc.issueType, err)
			continue
		}
		if got != tc.want {
			t.Errorf("slot %q = %q, want %q", tc.issueType, got, tc.want)
		}
	}
}

// Slot value beats a contradictory utterance (deterministic-first contract).
func TestSlotBeatsUtterance(t *testing.T) {
	c := NewClassifier(nil)
	got, err := c.Classify(context.Background(), "there's no sound", Slots{IssueType: "mic"})
	if err != nil {
		t.Fatal(err)
	}
	if got != SymptomMicNotWorking {
		t.Fatalf("slot should win: got %q", got)
	}
}

func TestClassifyAmbiguousFallsBackToLLM(t *testing.T) {
	llm := &mockLLM{class: SymptomMicNotWorking}
	c := NewClassifier(llm)
	got, err := c.Classify(context.Background(), "my headset is acting weird on calls", Slots{})
	if err != nil {
		t.Fatal(err)
	}
	if got != SymptomMicNotWorking {
		t.Fatalf("got %q, want LLM fallback result", got)
	}
	if llm.calls != 1 {
		t.Fatalf("LLM called %d times, want exactly 1", llm.calls)
	}
}

func TestClassifyTieIsAmbiguous(t *testing.T) {
	// "mono" (one_sided, 3) ties "button" (mute_call_control, 3).
	llm := &mockLLM{class: SymptomOneSidedAudio}
	c := NewClassifier(llm)
	got, err := c.Classify(context.Background(), "the mono button", Slots{})
	if err != nil {
		t.Fatal(err)
	}
	if llm.calls != 1 {
		t.Fatalf("tied keyword scores must defer to the LLM; calls = %d", llm.calls)
	}
	if got != SymptomOneSidedAudio {
		t.Fatalf("got %q", got)
	}
}

func TestClassifyNoMatchWithoutLLM(t *testing.T) {
	c := NewClassifier(nil)
	_, err := c.Classify(context.Background(), "my headset is acting weird", Slots{})
	if !errors.Is(err, ErrUnclassified) {
		t.Fatalf("expected ErrUnclassified, got %v", err)
	}
}

func TestClassifyEmptyUtteranceSkipsLLM(t *testing.T) {
	llm := &mockLLM{class: SymptomNoAudioOutput}
	c := NewClassifier(llm)
	_, err := c.Classify(context.Background(), "   ", Slots{})
	if !errors.Is(err, ErrUnclassified) {
		t.Fatalf("expected ErrUnclassified, got %v", err)
	}
	if llm.calls != 0 {
		t.Errorf("LLM must not be called for an empty utterance")
	}
}

func TestClassifyLLMErrorPropagates(t *testing.T) {
	llm := &mockLLM{err: errors.New("throttled")}
	c := NewClassifier(llm)
	_, err := c.Classify(context.Background(), "something is off with my headset", Slots{})
	if err == nil {
		t.Fatal("expected error from LLM fallback")
	}
}

func TestClassifyLLMInvalidClassRejected(t *testing.T) {
	llm := &mockLLM{class: "misc"} // no catch-all class exists
	c := NewClassifier(llm)
	_, err := c.Classify(context.Background(), "something is off with my headset", Slots{})
	if !errors.Is(err, ErrUnclassified) {
		t.Fatalf("expected ErrUnclassified for invalid LLM class, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Branch sub-classification (the n-way fork steps)
// ---------------------------------------------------------------------------

func TestClassifyBranch(t *testing.T) {
	cases := []struct {
		step      StepID
		utterance string
		want      string
		wantOK    bool
	}{
		// Tree 5 s1 — robotic / crackling / echo
		{"tree5.s1", "it's all robotic and choppy", BranchRobotic, true},
		{"tree5.s1", "voices sound underwater and keep cutting in and out", BranchRobotic, true},
		{"tree5.s1", "I hear crackling and static", BranchCrackling, true},
		{"tree5.s1", "there's a buzzing, popping noise", BranchCrackling, true},
		{"tree5.s1", "there's an echo on every call", BranchEcho, true},
		{"tree5.s1", "it just sounds bad", "", false},
		// Tree 5 s4 — who hears the echo
		{"tree5.s4", "I hear my own voice repeated back", BranchSelf, true},
		{"tree5.s4", "the customer hears the echo, they hear themselves", BranchOtherParty, true},
		// Tree 6 s1 — call volume vs sidetone
		{"tree6.s1", "the other person is too quiet", BranchOtherParty, true},
		{"tree6.s1", "I keep hearing myself in the headset", BranchSelf, true},
		{"tree6.s1", "the sidetone is too strong", BranchSelf, true},
		// Tree 7 s1 — buttons vs desync
		{"tree7.s1", "the answer and mute buttons do nothing", BranchButtons, true},
		{"tree7.s1", "the headset shows muted but the app shows unmuted", BranchDesync, true},
		// no vocabulary for ordinary steps / unknown steps
		{"tree1.s1", "anything", "", false},
		// empty utterance
		{"tree5.s1", "", "", false},
		// tied score → ambiguous → false ("dead" buttons:3 vs "still shows" desync:3)
		{"tree7.s1", "dead, still shows", "", false},
	}
	for _, tc := range cases {
		got, ok := ClassifyBranch(tc.step, tc.utterance)
		if ok != tc.wantOK || got != tc.want {
			t.Errorf("ClassifyBranch(%s, %q) = (%q, %v), want (%q, %v)",
				tc.step, tc.utterance, got, ok, tc.want, tc.wantOK)
		}
	}
}
