package handlers

import (
	"log/slog"
	"regexp"
	"strings"

	"github.com/headset-support-agent/internal/models"
)

// cardDigitPattern matches 13–16 consecutive digits, optionally separated by
// single spaces or dashes (e.g. "4111 1111 1111 1111" or "4111-1111-1111-1111").
// The pattern requires at least 13 digit characters total when separators are
// removed, which avoids false-positive matches on short numeric strings like
// model numbers or USB version identifiers.
var cardDigitPattern = regexp.MustCompile(
	`\b\d{4}[\s-]\d{4}[\s-]\d{4}[\s-]\d{2,4}\b` + // spaced/dashed 4-4-4-{2,4}
		`|\b\d{13,16}\b`, // or a solid 13–16-digit run
)

// paymentPhrases are explicit payment/billing phrases whose presence alone (even
// without a card-digit sequence) indicates a payment attempt. All are lower-case
// for case-insensitive comparison.
var paymentPhrases = []string{
	"my card number",
	"card number is",
	"pay with my card",
	"pay with card",
	"credit card",
	"debit card",
	"cvv",
	"make a payment",
	"billing",
	"card details",
	"card information",
	"card info",
	"payment method",
}

// DetectPaymentSolicitation returns true when the transcript indicates that the
// caller is offering payment details or asking to make a payment.
//
// Detection logic (conservative to minimise false positives):
//   - An explicit payment phrase alone is sufficient, OR
//   - A card-digit-like sequence alone is sufficient (≥13 contiguous digits, or
//     a 4-4-4-{2,4} hyphen/space-separated sequence).
//
// Importantly, this function NEVER logs or returns the transcript content — it
// only returns a boolean so that card digits are never echoed or persisted.
func DetectPaymentSolicitation(transcript string) bool {
	lower := strings.ToLower(transcript)

	// Check for explicit payment phrases first — fast path.
	for _, phrase := range paymentPhrases {
		if strings.Contains(lower, phrase) {
			slog.Info("payment solicitation detected via phrase match",
				slog.Bool("payment_detected", true),
			)
			return true
		}
	}

	// Check for a card-digit-like sequence.
	if cardDigitPattern.MatchString(transcript) {
		slog.Info("payment solicitation detected via digit pattern",
			slog.Bool("payment_detected", true),
		)
		return true
	}

	return false
}

// BuildPaymentRefusalResponse returns a persona-appropriate LexV2Response that
// refuses to handle payment or card data. It never repeats or echoes any digits
// from the caller's input. The session attribute "payment_blocked" is set to
// "true" so downstream contact-flow logic can route to a billing specialist.
func BuildPaymentRefusalResponse(persona *models.Persona, sessionAttrs map[string]string) LexV2Response {
	if sessionAttrs == nil {
		sessionAttrs = make(map[string]string)
	}
	sessionAttrs["payment_blocked"] = "true"

	msg := getPaymentRefusalMessage(persona)
	return BuildCloseResponse(persona, msg, sessionAttrs)
}

// getPaymentRefusalMessage returns a persona-flavoured refusal message that
// never contains digits or card information.
func getPaymentRefusalMessage(persona *models.Persona) string {
	if persona == nil {
		return "For your security, I can't take any payment or card details — I only help with headset troubleshooting. Please don't share card numbers here. If you need billing help I can connect you to a person."
	}

	switch persona.PersonaID {
	case "tangerine":
		return "Oh, I'm so sorry — I can't take any payment or card details here at all, I'm afraid! I'm only here for headset troubleshooting. For your security, please don't share card numbers with me. If you need billing help I can get you over to someone who can sort that out properly!"
	case "joseph":
		return "I appreciate you letting me know, but I'm not set up to handle any payment or card information — that's not something I'm able to do. For your security, please don't share card numbers here. I'm here just for headset support. If billing is the issue, I can get you connected with the right person."
	case "jennifer":
		return "Oh goodness, I can't take any payment or card details — that's just not something I'm able to help with here! I'm all about headset troubleshooting. For your security, please don't share card numbers with me. If you need billing help, tell ya what — I'll get you over to someone who can take care of that!"
	default:
		return "For your security, I can't take any payment or card details — I only help with headset troubleshooting. Please don't share card numbers here. If you need billing help I can connect you to a person."
	}
}
