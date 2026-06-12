package handlers

import (
	"regexp"
	"strings"
	"testing"

	"github.com/headset-support-agent/internal/models"
)

// digitRunPattern matches any run of 13 or more consecutive digits.
// Used in tests to verify that the refusal response never echoes card numbers.
var digitRunPattern = regexp.MustCompile(`\d{13,}`)

// ---------------------------------------------------------------------------
// DetectPaymentSolicitation — true cases
// ---------------------------------------------------------------------------

func TestDetectPaymentSolicitation_CreditCardPhrase(t *testing.T) {
	if !DetectPaymentSolicitation("my credit card number is 4111 1111 1111 1111") {
		t.Error("expected true for credit card number phrase with digits")
	}
}

func TestDetectPaymentSolicitation_PayWithCard(t *testing.T) {
	if !DetectPaymentSolicitation("I want to pay with my card") {
		t.Error("expected true for 'pay with my card'")
	}
}

func TestDetectPaymentSolicitation_CVV(t *testing.T) {
	if !DetectPaymentSolicitation("let me give you my cvv") {
		t.Error("expected true for CVV mention")
	}
}

func TestDetectPaymentSolicitation_RawSixteenDigitRun(t *testing.T) {
	if !DetectPaymentSolicitation("4111111111111111") {
		t.Error("expected true for raw 16-digit card number")
	}
}

func TestDetectPaymentSolicitation_SpacedCardNumber(t *testing.T) {
	if !DetectPaymentSolicitation("4111 1111 1111 1111") {
		t.Error("expected true for spaced card number '4111 1111 1111 1111'")
	}
}

func TestDetectPaymentSolicitation_DashedCardNumber(t *testing.T) {
	if !DetectPaymentSolicitation("my number is 4111-1111-1111-1111") {
		t.Error("expected true for dashed card number")
	}
}

func TestDetectPaymentSolicitation_ThirteenDigitRun(t *testing.T) {
	if !DetectPaymentSolicitation("4111111111111") {
		t.Error("expected true for 13-digit card number")
	}
}

func TestDetectPaymentSolicitation_MakeAPayment(t *testing.T) {
	if !DetectPaymentSolicitation("I need to make a payment") {
		t.Error("expected true for 'make a payment'")
	}
}

func TestDetectPaymentSolicitation_BillingPhrase(t *testing.T) {
	if !DetectPaymentSolicitation("I have a billing question") {
		t.Error("expected true for 'billing'")
	}
}

func TestDetectPaymentSolicitation_CardNumberIs(t *testing.T) {
	if !DetectPaymentSolicitation("card number is 4111 1111 1111 1111") {
		t.Error("expected true for 'card number is' phrase")
	}
}

func TestDetectPaymentSolicitation_CaseInsensitive(t *testing.T) {
	if !DetectPaymentSolicitation("I WANT TO PAY WITH MY CARD") {
		t.Error("expected true for uppercase payment phrase")
	}
	if !DetectPaymentSolicitation("CREDIT CARD") {
		t.Error("expected true for uppercase CREDIT CARD")
	}
}

// ---------------------------------------------------------------------------
// DetectPaymentSolicitation — false cases (no false positives)
// ---------------------------------------------------------------------------

func TestDetectPaymentSolicitation_HeadsetNotWorking(t *testing.T) {
	if DetectPaymentSolicitation("my headset isn't working") {
		t.Error("expected false for unrelated headset complaint")
	}
}

func TestDetectPaymentSolicitation_USB3(t *testing.T) {
	if DetectPaymentSolicitation("I have a USB 3.0 port") {
		t.Error("expected false for 'USB 3.0 port'")
	}
}

func TestDetectPaymentSolicitation_TwoReboots(t *testing.T) {
	if DetectPaymentSolicitation("I tried 2 reboots") {
		t.Error("expected false for reboot count mention")
	}
}

func TestDetectPaymentSolicitation_TreeStep(t *testing.T) {
	if DetectPaymentSolicitation("Tree 8 step 3") {
		t.Error("expected false for 'Tree 8 step 3'")
	}
}

func TestDetectPaymentSolicitation_EmptyString(t *testing.T) {
	if DetectPaymentSolicitation("") {
		t.Error("expected false for empty string")
	}
}

func TestDetectPaymentSolicitation_ShortDigitRun(t *testing.T) {
	// 12-digit run should not trigger (below 13-digit minimum).
	if DetectPaymentSolicitation("123456789012") {
		t.Error("expected false for 12-digit run (below card threshold)")
	}
}

func TestDetectPaymentSolicitation_ModelNumber(t *testing.T) {
	// Model numbers like "HW301" or "EncorePro 540" should not trigger.
	if DetectPaymentSolicitation("I have the EncorePro 540 headset") {
		t.Error("expected false for model number in headset description")
	}
}

func TestDetectPaymentSolicitation_FirmwareVersion(t *testing.T) {
	// Firmware/version strings should not trigger.
	if DetectPaymentSolicitation("running firmware version 1.2.3") {
		t.Error("expected false for firmware version string")
	}
}

// ---------------------------------------------------------------------------
// BuildPaymentRefusalResponse tests
// ---------------------------------------------------------------------------

func TestBuildPaymentRefusalResponse_SetsPaymentBlockedAttr(t *testing.T) {
	persona := &models.Persona{PersonaID: "joseph"}
	attrs := map[string]string{"existing": "value"}

	resp := BuildPaymentRefusalResponse(persona, attrs)

	sa := resp.SessionState.SessionAttributes
	if sa["payment_blocked"] != "true" {
		t.Errorf("expected payment_blocked=true, got %q", sa["payment_blocked"])
	}
	// Existing attributes must be preserved.
	if sa["existing"] != "value" {
		t.Errorf("expected existing attribute to be preserved, got %q", sa["existing"])
	}
}

func TestBuildPaymentRefusalResponse_NilSessionAttrs(t *testing.T) {
	// Must not panic when session attrs are nil.
	resp := BuildPaymentRefusalResponse(nil, nil)

	sa := resp.SessionState.SessionAttributes
	if sa == nil {
		t.Fatal("expected session attributes map to be initialised, got nil")
	}
	if sa["payment_blocked"] != "true" {
		t.Errorf("expected payment_blocked=true, got %q", sa["payment_blocked"])
	}
}

func TestBuildPaymentRefusalResponse_NonEmptyMessage(t *testing.T) {
	personas := []*models.Persona{
		nil,
		{PersonaID: "tangerine"},
		{PersonaID: "joseph"},
		{PersonaID: "jennifer"},
		{PersonaID: "unknown"},
	}

	for _, p := range personas {
		id := "nil"
		if p != nil {
			id = p.PersonaID
		}

		resp := BuildPaymentRefusalResponse(p, nil)

		if len(resp.Messages) == 0 {
			t.Errorf("persona=%s: expected at least one message", id)
			continue
		}
		if resp.Messages[0].Content == "" {
			t.Errorf("persona=%s: message content is empty", id)
		}
	}
}

func TestBuildPaymentRefusalResponse_MessageContainsNoCardDigits(t *testing.T) {
	// The refusal message must never echo a card-digit sequence.
	// We call with a session attr that could theoretically carry digits, and
	// verify the response message itself contains no 13+ digit run.
	cardInput := "4111111111111111"

	attrs := map[string]string{"raw_input": cardInput} // should not be echoed

	resp := BuildPaymentRefusalResponse(&models.Persona{PersonaID: "tangerine"}, attrs)

	for _, msg := range resp.Messages {
		if digitRunPattern.MatchString(msg.Content) {
			t.Errorf("refusal message must not contain a digit run; got: %s", msg.Content)
		}
	}
}

func TestBuildPaymentRefusalResponse_ReturnsCloseDialogAction(t *testing.T) {
	resp := BuildPaymentRefusalResponse(&models.Persona{PersonaID: "tangerine"}, nil)

	if resp.SessionState.DialogAction.Type != "Close" {
		t.Errorf("expected DialogAction=Close, got %q", resp.SessionState.DialogAction.Type)
	}
}

func TestBuildPaymentRefusalResponse_MessageContentTypeSSML(t *testing.T) {
	resp := BuildPaymentRefusalResponse(nil, nil)

	if len(resp.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(resp.Messages))
	}
	if resp.Messages[0].ContentType != "SSML" {
		t.Errorf("expected ContentType=SSML, got %q", resp.Messages[0].ContentType)
	}
}

func TestBuildPaymentRefusalResponse_MessageMentionsSecurity(t *testing.T) {
	// All persona messages should contain a security/safety indication.
	personas := []*models.Persona{
		nil,
		{PersonaID: "tangerine"},
		{PersonaID: "joseph"},
		{PersonaID: "jennifer"},
	}

	keywords := []string{"security", "can't", "billing", "card"}

	for _, p := range personas {
		id := "nil"
		if p != nil {
			id = p.PersonaID
		}

		resp := BuildPaymentRefusalResponse(p, nil)

		if len(resp.Messages) == 0 {
			t.Errorf("persona=%s: no messages returned", id)
			continue
		}

		content := strings.ToLower(resp.Messages[0].Content)
		foundAny := false
		for _, kw := range keywords {
			if strings.Contains(content, kw) {
				foundAny = true
				break
			}
		}
		if !foundAny {
			t.Errorf("persona=%s: refusal message should mention security/billing context; got: %s",
				id, resp.Messages[0].Content)
		}
	}
}

func TestBuildPaymentRefusalResponse_TangerinePersonaVoice(t *testing.T) {
	persona := &models.Persona{PersonaID: "tangerine"}
	resp := BuildPaymentRefusalResponse(persona, nil)

	if len(resp.Messages) == 0 {
		t.Fatal("expected a message")
	}
	// Tangerine is Irish/warm — message should be non-empty and persona-flavoured.
	content := resp.Messages[0].Content
	if content == "" {
		t.Error("expected non-empty message for tangerine persona")
	}
}
