package agents

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
		SystemPrompt: "You are a test agent.",
	}
}

func TestBuildSystemContext(t *testing.T) {
	persona := createTestPersona()
	context := buildSystemContext(persona)

	// Should contain persona's system prompt
	if !strings.Contains(context, persona.SystemPrompt) {
		t.Error("Context should contain persona's system prompt")
	}

	// Should contain troubleshooting knowledge section
	if !strings.Contains(context, "TROUBLESHOOTING KNOWLEDGE") {
		t.Error("Context should contain troubleshooting knowledge section")
	}

	// Should mention knowledge bases
	knowledgeBases := []string{"USB headset", "Bluetooth", "Windows audio", "Genesys Cloud"}
	for _, kb := range knowledgeBases {
		if !strings.Contains(context, kb) {
			t.Errorf("Context should mention %s knowledge base", kb)
		}
	}

	// Should contain behavior guidelines
	if !strings.Contains(context, "BEHAVIOR GUIDELINES") {
		t.Error("Context should contain behavior guidelines")
	}

	// Should contain voice optimization
	if !strings.Contains(context, "VOICE OPTIMIZATION") {
		t.Error("Context should contain voice optimization section")
	}

	// Should mention the voice ID
	if !strings.Contains(context, persona.VoiceConfig.PollyVoiceID) {
		t.Error("Context should mention the Polly voice ID")
	}
}

func TestBuildSystemContext_Personas(t *testing.T) {
	personas := []*models.Persona{
		{
			PersonaID:    "tangerine",
			SystemPrompt: "You are Tangerine, an Irish support agent.",
			VoiceConfig: models.VoiceConfig{
				PollyVoiceID: "Niamh",
			},
		},
		{
			PersonaID:    "joseph",
			SystemPrompt: "You are Joseph, a professional Canadian agent.",
			VoiceConfig: models.VoiceConfig{
				PollyVoiceID: "Matthew",
			},
		},
		{
			PersonaID:    "jennifer",
			SystemPrompt: "You are Jennifer, a friendly Texan agent.",
			VoiceConfig: models.VoiceConfig{
				PollyVoiceID: "Kendra",
			},
		},
	}

	for _, persona := range personas {
		t.Run(persona.PersonaID, func(t *testing.T) {
			context := buildSystemContext(persona)

			// Each persona's context should be unique
			if !strings.Contains(context, persona.SystemPrompt) {
				t.Errorf("Context should contain %s's system prompt", persona.PersonaID)
			}
			if !strings.Contains(context, persona.VoiceConfig.PollyVoiceID) {
				t.Errorf("Context should mention %s voice", persona.VoiceConfig.PollyVoiceID)
			}
		})
	}
}

func TestInvokeAgentInput_Validation(t *testing.T) {
	// Test that InvokeAgentInput struct has all required fields
	input := InvokeAgentInput{
		AgentID:      "test-agent-id",
		AgentAliasID: "test-alias-id",
		SessionID:    "test-session",
		InputText:    "Hello",
		Persona:      createTestPersona(),
	}

	if input.AgentID == "" {
		t.Error("AgentID should not be empty")
	}
	if input.AgentAliasID == "" {
		t.Error("AgentAliasID should not be empty")
	}
	if input.SessionID == "" {
		t.Error("SessionID should not be empty")
	}
	if input.InputText == "" {
		t.Error("InputText should not be empty")
	}
	if input.Persona == nil {
		t.Error("Persona should not be nil")
	}
}

func TestNewBedrockClient(t *testing.T) {
	// NewBedrockClient should create a client struct
	// We can't fully test without AWS credentials, but we can verify structure
	// In real tests, we'd use mocks or localstack

	// This is a placeholder for integration tests
	t.Skip("Requires AWS credentials for full test")
}

func TestBuildSystemContext_Guidelines(t *testing.T) {
	persona := createTestPersona()
	context := buildSystemContext(persona)

	guidelines := []string{
		"responses under 3 sentences",
		"ONE question at a time",
		"Confirm each step",
		"layman's terms",
	}

	for _, guideline := range guidelines {
		if !strings.Contains(context, guideline) {
			t.Errorf("Context should contain guideline: %s", guideline)
		}
	}
}

func TestBuildSystemContext_VoiceOptimization(t *testing.T) {
	persona := createTestPersona()
	context := buildSystemContext(persona)

	voiceGuidelines := []string{
		"spoken",
		"jargon",
		"conversation",
		"pauses",
	}

	for _, vg := range voiceGuidelines {
		if !strings.Contains(context, vg) {
			t.Errorf("Context should mention voice optimization: %s", vg)
		}
	}
}
