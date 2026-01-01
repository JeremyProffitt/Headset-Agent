package handlers

import (
	"strings"
	"testing"

	"github.com/headset-support-agent/internal/models"
)

func createTestPersona() *models.Persona {
	return &models.Persona{
		PersonaID:   "test",
		DisplayName: "Test Agent",
		VoiceConfig: models.VoiceConfig{
			PollyVoiceID: "Joanna",
			PollyEngine:  "neural",
			LanguageCode: "en-US",
			Prosody: models.Prosody{
				Rate:  "100%",
				Pitch: "medium",
			},
		},
	}
}

func TestBuildSuccessResponse(t *testing.T) {
	persona := createTestPersona()
	message := "Hello, how can I help you?"
	sessionAttrs := map[string]string{
		"user_name": "John",
	}

	response := BuildSuccessResponse(persona, message, sessionAttrs)

	// Verify session state
	if response.SessionState == nil {
		t.Fatal("SessionState should not be nil")
	}
	if response.SessionState.DialogAction == nil {
		t.Fatal("DialogAction should not be nil")
	}
	if response.SessionState.DialogAction.Type != "ElicitIntent" {
		t.Errorf("DialogAction.Type = %v, want ElicitIntent", response.SessionState.DialogAction.Type)
	}

	// Verify session attributes preserved
	if response.SessionState.SessionAttributes["user_name"] != "John" {
		t.Error("Session attributes should be preserved")
	}

	// Verify message
	if len(response.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(response.Messages))
	}
	if response.Messages[0].ContentType != "SSML" {
		t.Errorf("ContentType = %v, want SSML", response.Messages[0].ContentType)
	}
	if !strings.Contains(response.Messages[0].Content, message) {
		t.Errorf("Message content should contain original message")
	}
}

func TestBuildErrorResponse(t *testing.T) {
	persona := createTestPersona()
	message := "Sorry, something went wrong"

	response := BuildErrorResponse(persona, message)

	if response.SessionState == nil {
		t.Fatal("SessionState should not be nil")
	}
	if response.SessionState.DialogAction.Type != "ElicitIntent" {
		t.Errorf("DialogAction.Type = %v, want ElicitIntent", response.SessionState.DialogAction.Type)
	}

	// Error response should not have session attributes
	if response.SessionState.SessionAttributes != nil && len(response.SessionState.SessionAttributes) > 0 {
		t.Error("Error response should not have session attributes")
	}

	if len(response.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(response.Messages))
	}
}

func TestBuildTestResponse(t *testing.T) {
	response := BuildTestResponse()

	if response.SessionState == nil {
		t.Fatal("SessionState should not be nil")
	}
	if response.SessionState.DialogAction.Type != "Close" {
		t.Errorf("DialogAction.Type = %v, want Close", response.SessionState.DialogAction.Type)
	}
	if response.SessionState.Intent == nil {
		t.Fatal("Intent should not be nil")
	}
	if response.SessionState.Intent.Name != "HealthCheck" {
		t.Errorf("Intent.Name = %v, want HealthCheck", response.SessionState.Intent.Name)
	}
	if response.SessionState.Intent.State != "Fulfilled" {
		t.Errorf("Intent.State = %v, want Fulfilled", response.SessionState.Intent.State)
	}

	if len(response.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(response.Messages))
	}
	if response.Messages[0].ContentType != "PlainText" {
		t.Errorf("ContentType = %v, want PlainText", response.Messages[0].ContentType)
	}
	if !strings.Contains(response.Messages[0].Content, "Health check passed") {
		t.Error("Test response should indicate health check passed")
	}
}

func TestBuildSSML(t *testing.T) {
	tests := []struct {
		name     string
		persona  *models.Persona
		text     string
		wantRate string
		wantPitch string
	}{
		{
			name: "default prosody",
			persona: &models.Persona{
				VoiceConfig: models.VoiceConfig{
					Prosody: models.Prosody{
						Rate:  "100%",
						Pitch: "medium",
					},
				},
			},
			text:      "Hello there",
			wantRate:  "100%",
			wantPitch: "medium",
		},
		{
			name: "fast speech",
			persona: &models.Persona{
				VoiceConfig: models.VoiceConfig{
					Prosody: models.Prosody{
						Rate:  "120%",
						Pitch: "high",
					},
				},
			},
			text:      "Quick response",
			wantRate:  "120%",
			wantPitch: "high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ssml := BuildSSML(tt.persona, tt.text)

			// Verify SSML structure
			if !strings.HasPrefix(ssml, "<speak>") {
				t.Error("SSML should start with <speak>")
			}
			if !strings.HasSuffix(ssml, "</speak>") {
				t.Error("SSML should end with </speak>")
			}

			// Verify prosody settings
			if !strings.Contains(ssml, tt.wantRate) {
				t.Errorf("SSML should contain rate %s", tt.wantRate)
			}
			if !strings.Contains(ssml, tt.wantPitch) {
				t.Errorf("SSML should contain pitch %s", tt.wantPitch)
			}

			// Verify text is included
			if !strings.Contains(ssml, tt.text) {
				t.Error("SSML should contain the original text")
			}
		})
	}
}

func TestBuildCloseResponse(t *testing.T) {
	persona := createTestPersona()
	message := "Goodbye, have a great day!"
	intentName := "EndSession"

	response := BuildCloseResponse(persona, message, intentName)

	if response.SessionState == nil {
		t.Fatal("SessionState should not be nil")
	}
	if response.SessionState.DialogAction.Type != "Close" {
		t.Errorf("DialogAction.Type = %v, want Close", response.SessionState.DialogAction.Type)
	}
	if response.SessionState.Intent == nil {
		t.Fatal("Intent should not be nil")
	}
	if response.SessionState.Intent.Name != intentName {
		t.Errorf("Intent.Name = %v, want %v", response.SessionState.Intent.Name, intentName)
	}
	if response.SessionState.Intent.State != "Fulfilled" {
		t.Errorf("Intent.State = %v, want Fulfilled", response.SessionState.Intent.State)
	}

	if len(response.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(response.Messages))
	}
	if response.Messages[0].ContentType != "SSML" {
		t.Errorf("ContentType = %v, want SSML", response.Messages[0].ContentType)
	}
}

func TestBuildSuccessResponse_NilSessionAttrs(t *testing.T) {
	persona := createTestPersona()
	message := "Test message"

	// Should not panic with nil session attributes
	response := BuildSuccessResponse(persona, message, nil)

	if response.SessionState == nil {
		t.Fatal("SessionState should not be nil")
	}
	// Session attributes should be nil (not initialized)
	if response.SessionState.SessionAttributes != nil {
		t.Log("SessionAttributes was initialized, which is fine")
	}
}

func TestLexV2Response_JSONStructure(t *testing.T) {
	// Verify the response structure matches Lex V2 expectations
	response := LexV2Response{
		SessionState: &SessionState{
			DialogAction: &DialogAction{
				Type: "ElicitIntent",
			},
			Intent: &Intent{
				Name:  "TestIntent",
				State: "InProgress",
			},
			SessionAttributes: map[string]string{
				"key": "value",
			},
		},
		Messages: []Message{
			{
				ContentType: "PlainText",
				Content:     "Hello",
			},
		},
	}

	// Verify all fields are accessible
	if response.SessionState.DialogAction.Type != "ElicitIntent" {
		t.Error("DialogAction.Type mismatch")
	}
	if response.SessionState.Intent.Name != "TestIntent" {
		t.Error("Intent.Name mismatch")
	}
	if response.SessionState.SessionAttributes["key"] != "value" {
		t.Error("SessionAttributes mismatch")
	}
	if response.Messages[0].Content != "Hello" {
		t.Error("Messages content mismatch")
	}
}
