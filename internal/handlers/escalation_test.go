package handlers

import (
	"testing"

	"github.com/headset-support-agent/internal/models"
)

func TestDetectEscalation_ExplicitKeywords(t *testing.T) {
	tests := []struct {
		name       string
		transcript string
		wantEsc    bool
		wantReason string
	}{
		{
			name:       "agent keyword",
			transcript: "I want to talk to an agent",
			wantEsc:    true,
			wantReason: "user_requested",
		},
		{
			name:       "human keyword",
			transcript: "Can I speak to a human please",
			wantEsc:    true,
			wantReason: "user_requested",
		},
		{
			name:       "representative keyword",
			transcript: "Get me a representative",
			wantEsc:    true,
			wantReason: "user_requested",
		},
		{
			name:       "manager keyword",
			transcript: "Let me talk to your manager",
			wantEsc:    true,
			wantReason: "user_requested",
		},
		{
			name:       "case insensitive",
			transcript: "I WANT A REAL PERSON",
			wantEsc:    true,
			wantReason: "user_requested",
		},
		{
			name:       "no escalation keyword",
			transcript: "My headset isn't working",
			wantEsc:    false,
			wantReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectEscalation(tt.transcript, 0, 0)
			if result.ShouldEscalate != tt.wantEsc {
				t.Errorf("DetectEscalation() ShouldEscalate = %v, want %v", result.ShouldEscalate, tt.wantEsc)
			}
			if tt.wantEsc && result.Reason != tt.wantReason {
				t.Errorf("DetectEscalation() Reason = %v, want %v", result.Reason, tt.wantReason)
			}
		})
	}
}

func TestDetectEscalation_FrustrationThreshold(t *testing.T) {
	tests := []struct {
		name             string
		transcript       string
		frustrationCount int
		wantEsc          bool
		wantReason       string
	}{
		{
			name:             "frustration below threshold",
			transcript:       "this is ridiculous",
			frustrationCount: 1,
			wantEsc:          false,
		},
		{
			name:             "frustration at threshold",
			transcript:       "this doesn't work",
			frustrationCount: 2,
			wantEsc:          true,
			wantReason:       "user_frustrated",
		},
		{
			name:             "frustration above threshold",
			transcript:       "still not working",
			frustrationCount: 5,
			wantEsc:          true,
			wantReason:       "user_frustrated",
		},
		{
			name:             "no frustration indicators",
			transcript:       "I tried that step",
			frustrationCount: 2,
			wantEsc:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectEscalation(tt.transcript, tt.frustrationCount, 0)
			if result.ShouldEscalate != tt.wantEsc {
				t.Errorf("DetectEscalation() ShouldEscalate = %v, want %v", result.ShouldEscalate, tt.wantEsc)
			}
			if tt.wantEsc && result.Reason != tt.wantReason {
				t.Errorf("DetectEscalation() Reason = %v, want %v", result.Reason, tt.wantReason)
			}
		})
	}
}

func TestDetectEscalation_FailedSteps(t *testing.T) {
	tests := []struct {
		name        string
		failedSteps int
		wantEsc     bool
		wantReason  string
	}{
		{
			name:        "few failed steps",
			failedSteps: 2,
			wantEsc:     false,
		},
		{
			name:        "at threshold",
			failedSteps: 4,
			wantEsc:     false,
		},
		{
			name:        "above threshold",
			failedSteps: 5,
			wantEsc:     true,
			wantReason:  "troubleshooting_exhausted",
		},
		{
			name:        "many failed steps",
			failedSteps: 10,
			wantEsc:     true,
			wantReason:  "troubleshooting_exhausted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectEscalation("testing", 0, tt.failedSteps)
			if result.ShouldEscalate != tt.wantEsc {
				t.Errorf("DetectEscalation() ShouldEscalate = %v, want %v", result.ShouldEscalate, tt.wantEsc)
			}
			if tt.wantEsc && result.Reason != tt.wantReason {
				t.Errorf("DetectEscalation() Reason = %v, want %v", result.Reason, tt.wantReason)
			}
		})
	}
}

func TestDetectEscalation_Priority(t *testing.T) {
	// User requested escalation should be high priority
	result := DetectEscalation("I need to speak to an agent", 0, 0)
	if result.Priority != "high" {
		t.Errorf("User requested escalation should be high priority, got %v", result.Priority)
	}

	// Frustration escalation should be medium priority
	result = DetectEscalation("this is ridiculous and useless", 2, 0)
	if result.Priority != "medium" {
		t.Errorf("Frustration escalation should be medium priority, got %v", result.Priority)
	}
}

func TestBuildEscalationResponse(t *testing.T) {
	persona := &models.Persona{
		PersonaID:   "tangerine",
		DisplayName: "Tangerine",
		VoiceConfig: models.VoiceConfig{
			Prosody: models.Prosody{
				Rate:  "105%",
				Pitch: "medium",
			},
		},
	}

	decision := models.EscalationDecision{
		ShouldEscalate: true,
		Reason:         "user_requested",
		Priority:       "high",
	}

	sessionAttrs := map[string]string{
		"test_attr": "test_value",
	}

	response, err := BuildEscalationResponse(persona, decision, sessionAttrs)
	if err != nil {
		t.Fatalf("BuildEscalationResponse() error = %v", err)
	}

	// Verify session state
	if response.SessionState == nil {
		t.Fatal("Response should have SessionState")
	}
	if response.SessionState.DialogAction.Type != "Close" {
		t.Errorf("DialogAction.Type = %v, want Close", response.SessionState.DialogAction.Type)
	}
	if response.SessionState.Intent.Name != "EscalateToAgent" {
		t.Errorf("Intent.Name = %v, want EscalateToAgent", response.SessionState.Intent.Name)
	}

	// Verify escalation attributes were added
	if response.SessionState.SessionAttributes["escalation_requested"] != "true" {
		t.Error("Session should have escalation_requested = true")
	}
	if response.SessionState.SessionAttributes["escalation_reason"] != "user_requested" {
		t.Errorf("escalation_reason = %v, want user_requested", response.SessionState.SessionAttributes["escalation_reason"])
	}

	// Verify original attributes preserved
	if response.SessionState.SessionAttributes["test_attr"] != "test_value" {
		t.Error("Original session attributes should be preserved")
	}

	// Verify message exists
	if len(response.Messages) == 0 {
		t.Fatal("Response should have at least one message")
	}
	if response.Messages[0].ContentType != "SSML" {
		t.Errorf("Message ContentType = %v, want SSML", response.Messages[0].ContentType)
	}
}

func TestGetEscalationMessage_Personas(t *testing.T) {
	personas := []string{"tangerine", "joseph", "jennifer", "unknown"}
	reasons := []string{"requested", "frustrated", "exhausted", "default"}

	for _, personaID := range personas {
		persona := &models.Persona{PersonaID: personaID}
		for _, reason := range reasons {
			t.Run(personaID+"_"+reason, func(t *testing.T) {
				msg := getEscalationMessage(persona, reason)
				if msg == "" {
					t.Error("Escalation message should not be empty")
				}
				// All messages should mention transfer
				if !containsAny(msg, []string{"transfer", "connect", "get you to"}) {
					t.Errorf("Escalation message should mention transfer: %v", msg)
				}
			})
		}
	}
}

func TestGetEscalationMessage_CustomPhrases(t *testing.T) {
	persona := &models.Persona{
		PersonaID: "custom",
		Phrases: models.Phrases{
			Escalation: []string{"Custom escalation message"},
		},
	}

	msg := getEscalationMessage(persona, "requested")
	if msg != "Custom escalation message" {
		t.Errorf("Should use custom escalation phrase, got: %v", msg)
	}
}

// Helper function
func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if contains(s, sub) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
