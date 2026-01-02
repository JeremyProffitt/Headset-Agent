package handlers

import (
	"fmt"
	"html"
	"strings"

	"github.com/headset-support-agent/internal/models"
)

// LexV2Response represents the response structure for Lex V2
type LexV2Response struct {
	SessionState *SessionState `json:"sessionState,omitempty"`
	Messages     []Message     `json:"messages,omitempty"`
}

// SessionState represents the session state in Lex V2 response
type SessionState struct {
	DialogAction      *DialogAction     `json:"dialogAction,omitempty"`
	Intent            *Intent           `json:"intent,omitempty"`
	SessionAttributes map[string]string `json:"sessionAttributes,omitempty"`
}

// DialogAction represents the dialog action in Lex V2
type DialogAction struct {
	Type string `json:"type"`
}

// Intent represents the intent in Lex V2
type Intent struct {
	Name  string `json:"name,omitempty"`
	State string `json:"state,omitempty"`
}

// Message represents a message in Lex V2 response
type Message struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

// BuildSuccessResponse creates a successful Lex response with persona styling
func BuildSuccessResponse(p *models.Persona, message string, sessionAttrs map[string]string) LexV2Response {
	// Apply persona voice styling via SSML
	ssmlMessage := BuildSSML(p, message)

	return LexV2Response{
		SessionState: &SessionState{
			DialogAction: &DialogAction{
				Type: "ElicitIntent",
			},
			SessionAttributes: sessionAttrs,
		},
		Messages: []Message{
			{
				ContentType: "SSML",
				Content:     ssmlMessage,
			},
		},
	}
}

// BuildErrorResponse creates an error response with persona styling
func BuildErrorResponse(p *models.Persona, message string) LexV2Response {
	ssmlMessage := BuildSSML(p, message)

	return LexV2Response{
		SessionState: &SessionState{
			DialogAction: &DialogAction{
				Type: "ElicitIntent",
			},
		},
		Messages: []Message{
			{
				ContentType: "SSML",
				Content:     ssmlMessage,
			},
		},
	}
}

// BuildTestResponse creates a response for test invocations
func BuildTestResponse() LexV2Response {
	return LexV2Response{
		SessionState: &SessionState{
			DialogAction: &DialogAction{
				Type: "Close",
			},
			Intent: &Intent{
				Name:  "HealthCheck",
				State: "Fulfilled",
			},
		},
		Messages: []Message{
			{
				ContentType: "PlainText",
				Content:     "Health check passed. Lambda function is operational.",
			},
		},
	}
}

// BuildSSML wraps text in SSML with persona voice settings
func BuildSSML(p *models.Persona, text string) string {
	// Validate input text - use fallback if empty to prevent silent failures
	if text == "" {
		text = "I'm sorry, I didn't catch that. Could you please repeat your question?"
	}

	// Escape special XML characters to prevent SSML parsing errors
	// This is critical - if Bedrock returns <, >, &, ", or ', the SSML would be malformed
	escapedText := html.EscapeString(text)

	// Remove any existing SSML tags that might have been included in the response
	// Some models include their own <speak> tags which would break our wrapper
	escapedText = strings.ReplaceAll(escapedText, "&lt;speak&gt;", "")
	escapedText = strings.ReplaceAll(escapedText, "&lt;/speak&gt;", "")

	// Use simple SSML without amazon:domain to ensure compatibility with Lex
	return fmt.Sprintf(`<speak><prosody rate="%s" pitch="%s">%s</prosody></speak>`,
		p.VoiceConfig.Prosody.Rate, p.VoiceConfig.Prosody.Pitch, escapedText)
}

// BuildCloseResponse creates a close dialog response
func BuildCloseResponse(p *models.Persona, message string, intentName string) LexV2Response {
	ssmlMessage := BuildSSML(p, message)

	return LexV2Response{
		SessionState: &SessionState{
			DialogAction: &DialogAction{
				Type: "Close",
			},
			Intent: &Intent{
				Name:  intentName,
				State: "Fulfilled",
			},
		},
		Messages: []Message{
			{
				ContentType: "SSML",
				Content:     ssmlMessage,
			},
		},
	}
}
