package handlers

import (
	"strings"

	"github.com/headset-support-agent/internal/models"
)

// EscapeKeywords are words that trigger escalation to a human agent
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
	"live agent",
	"customer service",
	"operator",
}

// FrustrationIndicators are phrases that indicate user frustration
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
	"stupid",
	"hate this",
	"worst",
	"never works",
	"broken",
	"garbage",
}

// DetectEscalation analyzes user input for escalation triggers
func DetectEscalation(transcript string, frustrationCount int, failedSteps int) *models.EscalationDecision {
	lowerTranscript := strings.ToLower(transcript)

	// Check for explicit human request
	for _, keyword := range EscapeKeywords {
		if strings.Contains(lowerTranscript, keyword) {
			return &models.EscalationDecision{
				ShouldEscalate: true,
				Reason:         "user_requested",
				Priority:       "high",
			}
		}
	}

	// Check for frustration indicators
	currentFrustration := 0
	for _, indicator := range FrustrationIndicators {
		if strings.Contains(lowerTranscript, indicator) {
			currentFrustration++
		}
	}

	// Escalate if frustration threshold exceeded
	totalFrustration := frustrationCount + currentFrustration
	if totalFrustration >= 3 {
		return &models.EscalationDecision{
			ShouldEscalate: true,
			Reason:         "user_frustrated",
			Priority:       "medium",
		}
	}

	// Escalate if too many failed troubleshooting steps
	if failedSteps >= 5 {
		return &models.EscalationDecision{
			ShouldEscalate: true,
			Reason:         "troubleshooting_exhausted",
			Priority:       "medium",
		}
	}

	return &models.EscalationDecision{
		ShouldEscalate: false,
		Reason:         "",
		Priority:       "",
	}
}

// BuildEscalationResponse creates a persona-appropriate escalation response
func BuildEscalationResponse(persona *models.Persona, decision *models.EscalationDecision, sessionAttrs map[string]string) LexV2Response {
	message := getEscalationMessage(persona, decision.Reason)

	// Set escalation attributes for Connect transfer
	if sessionAttrs == nil {
		sessionAttrs = make(map[string]string)
	}
	sessionAttrs["escalation_requested"] = "true"
	sessionAttrs["escalation_reason"] = decision.Reason
	sessionAttrs["escalation_priority"] = decision.Priority

	return BuildCloseResponse(persona, message, sessionAttrs)
}

// getEscalationMessage returns a persona-specific escalation message
func getEscalationMessage(persona *models.Persona, reason string) string {
	if persona == nil {
		return "Let me transfer you to a specialist who can help you further."
	}

	// Use persona's escalation phrases if available
	if len(persona.Phrases.Escalation) > 0 {
		return persona.Phrases.Escalation[0]
	}

	// Persona-specific fallback messages
	switch persona.PersonaID {
	case "tangerine":
		switch reason {
		case "user_requested":
			return "No bother at all! Let me get you connected with one of our brilliant specialists who can help you out. Just one moment!"
		case "user_frustrated":
			return "I'm so sorry this has been such a hassle for you. Let me get you through to someone who can sort this out properly. You're in good hands!"
		default:
			return "Right so, we've tried a good few things here. Let me connect you with a specialist who might have some other tricks up their sleeve!"
		}
	case "joseph":
		switch reason {
		case "user_requested":
			return "Absolutely, I understand. Let me get you connected with a specialist. They'll take good care of you."
		case "user_frustrated":
			return "I hear you, and I'm sorry we haven't been able to resolve this yet. Let me transfer you to someone who can dig a little deeper into this issue."
		default:
			return "Alright, we've worked through quite a few steps here. I think it's time to bring in a specialist who might have some additional solutions. Let me transfer you now."
		}
	case "jennifer":
		switch reason {
		case "user_requested":
			return "You got it! Let me get you over to one of our specialists right quick. They're real good at what they do!"
		case "user_frustrated":
			return "Well shoot, I'm real sorry this has been giving you so much trouble. Tell ya what, let me get you connected with someone who can really dig into this for you."
		default:
			return "Alright, we've been through quite a bit here! Let me hand you off to one of our specialists - they've got a few more tricks they can try."
		}
	default:
		return "Let me connect you with a specialist who can help you further."
	}
}
