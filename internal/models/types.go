package models

// Persona represents an agent persona configuration
type Persona struct {
	PersonaID     string      `json:"persona_id" dynamodbav:"persona_id"`
	DisplayName   string      `json:"display_name" dynamodbav:"display_name"`
	VoiceConfig   VoiceConfig `json:"voice_config" dynamodbav:"voice_config"`
	Personality   Personality `json:"personality" dynamodbav:"personality"`
	Phrases       Phrases     `json:"phrases" dynamodbav:"phrases"`
	SystemPrompt  string      `json:"system_prompt" dynamodbav:"system_prompt"`
	FillerPhrases []string    `json:"filler_phrases" dynamodbav:"filler_phrases"`
}

// VoiceConfig contains voice synthesis settings
type VoiceConfig struct {
	PollyVoiceID   string  `json:"polly_voice_id" dynamodbav:"polly_voice_id"`
	PollyEngine    string  `json:"polly_engine" dynamodbav:"polly_engine"`
	LanguageCode   string  `json:"language_code" dynamodbav:"language_code"`
	Prosody        Prosody `json:"prosody" dynamodbav:"prosody"`
	UseNovaSonic   bool    `json:"use_nova_sonic" dynamodbav:"use_nova_sonic"`
	NovaSonicVoice string  `json:"nova_sonic_voice" dynamodbav:"nova_sonic_voice"`
}

// Prosody contains speech rate and pitch settings
type Prosody struct {
	Rate  string `json:"rate" dynamodbav:"rate"`
	Pitch string `json:"pitch" dynamodbav:"pitch"`
}

// Personality contains character traits
type Personality struct {
	Origin      string   `json:"origin" dynamodbav:"origin"`
	Age         int      `json:"age" dynamodbav:"age"`
	Gender      string   `json:"gender" dynamodbav:"gender"`
	Traits      []string `json:"traits" dynamodbav:"traits"`
	SpeechStyle string   `json:"speech_style" dynamodbav:"speech_style"`
	Pace        string   `json:"pace" dynamodbav:"pace"`
}

// Phrases contains situation-specific phrases
type Phrases struct {
	Greeting      []string `json:"greeting" dynamodbav:"greeting"`
	Confirmation  []string `json:"confirmation" dynamodbav:"confirmation"`
	Encouragement []string `json:"encouragement" dynamodbav:"encouragement"`
	Empathy       []string `json:"empathy" dynamodbav:"empathy"`
	Escalation    []string `json:"escalation" dynamodbav:"escalation"`
}

// EscalationDecision represents the result of escalation detection
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

// Session represents a conversation session
type Session struct {
	SessionID    string            `json:"session_id" dynamodbav:"session_id"`
	PersonaID    string            `json:"persona_id" dynamodbav:"persona_id"`
	Attributes   map[string]string `json:"attributes" dynamodbav:"attributes"`
	TTL          int64             `json:"ttl" dynamodbav:"ttl"`
	CreatedAt    string            `json:"created_at" dynamodbav:"created_at"`
	LastActivity string            `json:"last_activity" dynamodbav:"last_activity"`
}

// NovaSonicConfig contains Nova Sonic specific configuration
type NovaSonicConfig struct {
	ModelID         string `json:"model_id"`
	VoiceID         string `json:"voice_id"`
	SampleRate      int    `json:"sample_rate"`
	SampleSizeBits  int    `json:"sample_size_bits"`
	ChannelCount    int    `json:"channel_count"`
	MaxTokens       int    `json:"max_tokens"`
	Temperature     float64 `json:"temperature"`
	TopP            float64 `json:"top_p"`
}

// AudioChunk represents a chunk of audio data for streaming
type AudioChunk struct {
	Content   string `json:"content"` // base64 encoded audio
	Timestamp int64  `json:"timestamp"`
	Final     bool   `json:"final"`
}
