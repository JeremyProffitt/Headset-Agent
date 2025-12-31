package handlers

import (
	"strings"

	"github.com/headset-support-agent/internal/models"
)

// EscapeKeywords that trigger human escalation
var EscapeKeywords = []string{
	"agent",
	"human",
	"representative",
	"speak to someone",
	"real person",
	"transfer me",
	"manager",
	"supervisor",
	"talk to a person",
}

// FrustrationIndicators suggest user is frustrated
var FrustrationIndicators = []string{
	"this is ridiculous",
	"doesn't work",
	"still not working",
	"frustrated",
	"waste of time",
	"useless",
	"terrible",
	"not helping",
	"give up",
}

// DetectEscalation analyzes input for escalation triggers
func DetectEscalation(transcript string, frustrationCount int, failedSteps int) models.EscalationDecision {
	lowerTranscript := strings.ToLower(transcript)

	// Check for explicit escape keywords
	for _, keyword := range EscapeKeywords {
		if strings.Contains(lowerTranscript, keyword) {
			return models.EscalationDecision{
				ShouldEscalate: true,
				Reason:         "user_requested",
				Priority:       "high",
			}
		}
	}

	// Check frustration indicators
	for _, indicator := range FrustrationIndicators {
		if strings.Contains(lowerTranscript, indicator) {
			frustrationCount++
		}
	}

	// Escalate if frustration threshold exceeded
	if frustrationCount > 2 {
		return models.EscalationDecision{
			ShouldEscalate: true,
			Reason:         "user_frustrated",
			Priority:       "medium",
		}
	}

	// Escalate if too many failed troubleshooting steps
	if failedSteps > 4 {
		return models.EscalationDecision{
			ShouldEscalate: true,
			Reason:         "troubleshooting_exhausted",
			Priority:       "medium",
		}
	}

	return models.EscalationDecision{
		ShouldEscalate: false,
	}
}

// BuildEscalationResponse creates a response for escalation scenarios
func BuildEscalationResponse(p *models.Persona, decision models.EscalationDecision, sessionAttrs map[string]string) (LexV2Response, error) {
	var message string

	// Select appropriate escalation message based on persona and reason
	switch decision.Reason {
	case "user_requested":
		message = getEscalationMessage(p, "requested")
	case "user_frustrated":
		message = getEscalationMessage(p, "frustrated")
	case "troubleshooting_exhausted":
		message = getEscalationMessage(p, "exhausted")
	default:
		message = getEscalationMessage(p, "default")
	}

	// Update session attributes for escalation
	if sessionAttrs == nil {
		sessionAttrs = make(map[string]string)
	}
	sessionAttrs["escalation_requested"] = "true"
	sessionAttrs["escalation_reason"] = decision.Reason
	sessionAttrs["escalation_priority"] = decision.Priority

	ssmlMessage := BuildSSML(p, message)

	return LexV2Response{
		SessionState: &SessionState{
			DialogAction: &DialogAction{
				Type: "Close",
			},
			Intent: &Intent{
				Name:  "EscalateToAgent",
				State: "Fulfilled",
			},
			SessionAttributes: sessionAttrs,
		},
		Messages: []Message{
			{
				ContentType: "SSML",
				Content:     ssmlMessage,
			},
		},
	}, nil
}

// getEscalationMessage returns persona-appropriate escalation message
func getEscalationMessage(p *models.Persona, reason string) string {
	// Use persona's escalation phrases if available
	if len(p.Phrases.Escalation) > 0 {
		return p.Phrases.Escalation[0]
	}

	// Default messages by persona ID
	switch p.PersonaID {
	case "tangerine":
		switch reason {
		case "requested":
			return "Absolutely! I'll get you to someone straight away. Just hold on one moment and I'll transfer you now."
		case "frustrated":
			return "Ah, I can hear this has been a pain. Let me get you to a specialist who can sort this right out. Transferring you now."
		case "exhausted":
			return "Right, we've tried a good few things here. I think a specialist can help you better. Let me transfer you over."
		default:
			return "Let me get you to someone who can help. Transferring you now!"
		}
	case "joseph":
		switch reason {
		case "requested":
			return "Absolutely, I'll connect you with a support specialist right away. I'm transferring you now. Please hold for just a moment."
		case "frustrated":
			return "I understand this has been frustrating. Let me get you to a specialist who can help resolve this directly. Transferring you now."
		case "exhausted":
			return "We've tried several steps without success. I think a specialist can better assist you. Let me transfer you to someone who can help."
		default:
			return "Let me connect you with someone who can help. Transferring now."
		}
	case "jennifer":
		switch reason {
		case "requested":
			return "You got it! Let me get you over to a specialist right quick. Hang tight, I'm transferring you now!"
		case "frustrated":
			return "I hear ya, this is no fun at all. Tell ya what, let me get you to someone who can really dig into this. Transferring you now!"
		case "exhausted":
			return "Alright, we've given this a good go. Let me get you to a specialist who can take a closer look. Transferring!"
		default:
			return "Let me get you to someone who can help. Transferring you now!"
		}
	default:
		return "I'll connect you with a specialist who can help. Please hold while I transfer you."
	}
}
