package models

// Persona represents an agent persona configuration
type Persona struct {
	PersonaID   string      `json:"persona_id" dynamodbav:"persona_id"`
	DisplayName string      `json:"display_name" dynamodbav:"display_name"`
	VoiceConfig VoiceConfig `json:"voice_config" dynamodbav:"voice_config"`
	Personality Personality `json:"personality" dynamodbav:"personality"`
	Phrases     Phrases     `json:"phrases" dynamodbav:"phrases"`
	SystemPrompt string     `json:"system_prompt" dynamodbav:"system_prompt"`
	FillerPhrases []string  `json:"filler_phrases" dynamodbav:"filler_phrases"`
}

// VoiceConfig contains Amazon Polly and Nova Sonic voice settings
type VoiceConfig struct {
	PollyVoiceID       string  `json:"polly_voice_id" dynamodbav:"polly_voice_id"`
	PollyEngine        string  `json:"polly_engine" dynamodbav:"polly_engine"`
	LanguageCode       string  `json:"language_code" dynamodbav:"language_code"`
	Prosody            Prosody `json:"prosody" dynamodbav:"prosody"`
	UseNovaSonic       bool    `json:"use_nova_sonic" dynamodbav:"use_nova_sonic"`
	NovaSonicVoiceID   string  `json:"nova_sonic_voice_id,omitempty" dynamodbav:"nova_sonic_voice_id,omitempty"`
	FallbackPollyVoiceID string `json:"fallback_polly_voice_id,omitempty" dynamodbav:"fallback_polly_voice_id,omitempty"`
}

// Prosody contains speech rate and pitch settings
type Prosody struct {
	Rate  string `json:"rate" dynamodbav:"rate"`
	Pitch string `json:"pitch" dynamodbav:"pitch"`
}

// Personality contains persona character traits
type Personality struct {
	Origin      string   `json:"origin" dynamodbav:"origin"`
	Age         int      `json:"age" dynamodbav:"age"`
	Gender      string   `json:"gender" dynamodbav:"gender"`
	Traits      []string `json:"traits" dynamodbav:"traits"`
	SpeechStyle string   `json:"speech_style" dynamodbav:"speech_style"`
	Pace        string   `json:"pace" dynamodbav:"pace"`
}

// Phrases contains persona-specific phrases for different situations
type Phrases struct {
	Greeting      []string `json:"greeting" dynamodbav:"greeting"`
	Confirmation  []string `json:"confirmation" dynamodbav:"confirmation"`
	Encouragement []string `json:"encouragement" dynamodbav:"encouragement"`
	Empathy       []string `json:"empathy" dynamodbav:"empathy"`
	Escalation    []string `json:"escalation" dynamodbav:"escalation"`
}

// EscalationDecision contains the result of escalation detection
type EscalationDecision struct {
	ShouldEscalate bool   `json:"should_escalate"`
	Reason         string `json:"reason"`
	Priority       string `json:"priority"`
}

// AgentResponse represents a response from Bedrock agent
type AgentResponse struct {
	OutputText string            `json:"output_text"`
	SessionID  string            `json:"session_id"`
	Metadata   map[string]string `json:"metadata"`
}
