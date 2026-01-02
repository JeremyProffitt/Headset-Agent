package handlers

import (
	"fmt"
	"html"

	"github.com/headset-support-agent/internal/models"
)

// LexV2Response represents a Lex V2 fulfillment response
type LexV2Response struct {
	SessionState SessionState `json:"sessionState"`
	Messages     []Message    `json:"messages"`
}

// SessionState represents the session state in a Lex response
type SessionState struct {
	SessionAttributes map[string]string `json:"sessionAttributes,omitempty"`
	DialogAction      DialogAction      `json:"dialogAction"`
	Intent            *Intent           `json:"intent,omitempty"`
}

// DialogAction represents the dialog action in a Lex response
type DialogAction struct {
	Type string `json:"type"`
}

// Intent represents an intent in a Lex response
type Intent struct {
	Name              string `json:"name"`
	State             string `json:"state"`
	ConfirmationState string `json:"confirmationState,omitempty"`
}

// Message represents a message in a Lex response
type Message struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

// BuildSuccessResponse creates a successful Lex response with persona styling
func BuildSuccessResponse(persona *models.Persona, text string, sessionAttrs map[string]string) LexV2Response {
	ssml := BuildSSML(text, persona)

	return LexV2Response{
		SessionState: SessionState{
			SessionAttributes: sessionAttrs,
			DialogAction: DialogAction{
				Type: "ElicitIntent",
			},
		},
		Messages: []Message{
			{
				ContentType: "SSML",
				Content:     ssml,
			},
		},
	}
}

// BuildErrorResponse creates an error response with persona styling
func BuildErrorResponse(persona *models.Persona, text string) LexV2Response {
	ssml := BuildSSML(text, persona)

	return LexV2Response{
		SessionState: SessionState{
			DialogAction: DialogAction{
				Type: "ElicitIntent",
			},
		},
		Messages: []Message{
			{
				ContentType: "SSML",
				Content:     ssml,
			},
		},
	}
}

// BuildCloseResponse creates a close dialog response
func BuildCloseResponse(persona *models.Persona, text string, sessionAttrs map[string]string) LexV2Response {
	ssml := BuildSSML(text, persona)

	return LexV2Response{
		SessionState: SessionState{
			SessionAttributes: sessionAttrs,
			DialogAction: DialogAction{
				Type: "Close",
			},
			Intent: &Intent{
				Name:  "TroubleshootIntent",
				State: "Fulfilled",
			},
		},
		Messages: []Message{
			{
				ContentType: "SSML",
				Content:     ssml,
			},
		},
	}
}

// BuildTestResponse creates a health check response
func BuildTestResponse() LexV2Response {
	return LexV2Response{
		SessionState: SessionState{
			DialogAction: DialogAction{
				Type: "Close",
			},
			Intent: &Intent{
				Name:  "TroubleshootIntent",
				State: "Fulfilled",
			},
		},
		Messages: []Message{
			{
				ContentType: "PlainText",
				Content:     "Health check successful",
			},
		},
	}
}

// BuildSSML wraps text in SSML with prosody settings for the persona
func BuildSSML(text string, persona *models.Persona) string {
	// Escape special characters for SSML
	escapedText := html.EscapeString(text)

	// Default prosody if no persona
	rate := "100%"
	pitch := "medium"

	if persona != nil && persona.VoiceConfig.Prosody.Rate != "" {
		rate = persona.VoiceConfig.Prosody.Rate
	}
	if persona != nil && persona.VoiceConfig.Prosody.Pitch != "" {
		pitch = persona.VoiceConfig.Prosody.Pitch
	}

	return fmt.Sprintf(`<speak><prosody rate="%s" pitch="%s"><amazon:domain name="conversational">%s</amazon:domain></prosody></speak>`,
		rate, pitch, escapedText)
}
