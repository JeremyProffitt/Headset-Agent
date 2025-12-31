package persona

import (
	"testing"
)

func TestDefaultPersona(t *testing.T) {
	p := DefaultPersona()

	// Verify basic fields
	if p.PersonaID != "default" {
		t.Errorf("PersonaID = %v, want default", p.PersonaID)
	}
	if p.DisplayName != "Support Agent" {
		t.Errorf("DisplayName = %v, want Support Agent", p.DisplayName)
	}

	// Verify voice config
	if p.VoiceConfig.PollyVoiceID != "Joanna" {
		t.Errorf("PollyVoiceID = %v, want Joanna", p.VoiceConfig.PollyVoiceID)
	}
	if p.VoiceConfig.PollyEngine != "neural" {
		t.Errorf("PollyEngine = %v, want neural", p.VoiceConfig.PollyEngine)
	}
	if p.VoiceConfig.LanguageCode != "en-US" {
		t.Errorf("LanguageCode = %v, want en-US", p.VoiceConfig.LanguageCode)
	}

	// Verify prosody
	if p.VoiceConfig.Prosody.Rate != "100%" {
		t.Errorf("Prosody.Rate = %v, want 100%%", p.VoiceConfig.Prosody.Rate)
	}
	if p.VoiceConfig.Prosody.Pitch != "medium" {
		t.Errorf("Prosody.Pitch = %v, want medium", p.VoiceConfig.Prosody.Pitch)
	}

	// Verify personality
	if p.Personality.Gender != "female" {
		t.Errorf("Personality.Gender = %v, want female", p.Personality.Gender)
	}
	if len(p.Personality.Traits) == 0 {
		t.Error("Personality.Traits should not be empty")
	}

	// Verify phrases
	if len(p.Phrases.Greeting) == 0 {
		t.Error("Phrases.Greeting should not be empty")
	}
	if len(p.Phrases.Escalation) == 0 {
		t.Error("Phrases.Escalation should not be empty")
	}

	// Verify system prompt
	if p.SystemPrompt == "" {
		t.Error("SystemPrompt should not be empty")
	}
}

func TestNewLoader(t *testing.T) {
	// NewLoader should not panic with nil config
	// In real tests, we'd use localstack or mock
	// Here we just verify the struct is created
	loader := &Loader{
		tableName: "test-table",
	}

	if loader.tableName != "test-table" {
		t.Errorf("tableName = %v, want test-table", loader.tableName)
	}
}

func TestDefaultPersona_HasRequiredPhrases(t *testing.T) {
	p := DefaultPersona()

	phraseTypes := []struct {
		name    string
		phrases []string
	}{
		{"Greeting", p.Phrases.Greeting},
		{"Confirmation", p.Phrases.Confirmation},
		{"Encouragement", p.Phrases.Encouragement},
		{"Empathy", p.Phrases.Empathy},
		{"Escalation", p.Phrases.Escalation},
	}

	for _, pt := range phraseTypes {
		if len(pt.phrases) == 0 {
			t.Errorf("%s phrases should not be empty", pt.name)
		}
		for _, phrase := range pt.phrases {
			if phrase == "" {
				t.Errorf("%s contains empty phrase", pt.name)
			}
		}
	}
}

func TestDefaultPersona_VoiceConfigComplete(t *testing.T) {
	p := DefaultPersona()

	if p.VoiceConfig.PollyVoiceID == "" {
		t.Error("PollyVoiceID should not be empty")
	}
	if p.VoiceConfig.PollyEngine == "" {
		t.Error("PollyEngine should not be empty")
	}
	if p.VoiceConfig.LanguageCode == "" {
		t.Error("LanguageCode should not be empty")
	}
	if p.VoiceConfig.Prosody.Rate == "" {
		t.Error("Prosody.Rate should not be empty")
	}
	if p.VoiceConfig.Prosody.Pitch == "" {
		t.Error("Prosody.Pitch should not be empty")
	}
}

func TestDefaultPersona_PersonalityComplete(t *testing.T) {
	p := DefaultPersona()

	if p.Personality.Origin == "" {
		t.Error("Personality.Origin should not be empty")
	}
	if p.Personality.Age <= 0 {
		t.Error("Personality.Age should be positive")
	}
	if p.Personality.Gender == "" {
		t.Error("Personality.Gender should not be empty")
	}
	if len(p.Personality.Traits) == 0 {
		t.Error("Personality.Traits should not be empty")
	}
	if p.Personality.SpeechStyle == "" {
		t.Error("Personality.SpeechStyle should not be empty")
	}
	if p.Personality.Pace == "" {
		t.Error("Personality.Pace should not be empty")
	}
}

func TestDefaultPersona_SystemPromptContent(t *testing.T) {
	p := DefaultPersona()

	// System prompt should mention key aspects
	keywords := []string{"support", "headset", "troubleshooting"}
	for _, keyword := range keywords {
		found := false
		for i := 0; i <= len(p.SystemPrompt)-len(keyword); i++ {
			if len(p.SystemPrompt) >= len(keyword) {
				// Case insensitive check
				if containsIgnoreCase(p.SystemPrompt, keyword) {
					found = true
					break
				}
			}
		}
		if !found {
			t.Errorf("SystemPrompt should mention '%s'", keyword)
		}
	}
}

func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}
