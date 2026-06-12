package handlers

import (
	"strings"
	"testing"

	"github.com/headset-support-agent/internal/models"
)

// ---------------------------------------------------------------------------
// Helper builders
// ---------------------------------------------------------------------------

func makePersonaWithProsody(rate, pitch string) *models.Persona {
	return &models.Persona{
		PersonaID: "test-persona",
		VoiceConfig: models.VoiceConfig{
			Prosody: models.Prosody{
				Rate:  rate,
				Pitch: pitch,
			},
		},
	}
}

// ---------------------------------------------------------------------------
// BuildSSML tests
// ---------------------------------------------------------------------------

func TestBuildSSML_HTMLEscapesSpecialChars(t *testing.T) {
	input := `Hello <script>alert("xss")</script> & goodbye`

	persona := makePersonaWithProsody("medium", "medium")
	result := BuildSSML(input, persona)

	// Raw dangerous characters must not survive unescaped inside the SSML payload.
	if strings.Contains(result, "<script>") {
		t.Errorf("SSML must not contain raw <script> tag; got: %s", result)
	}
	if strings.Contains(result, `alert("xss")`) && !strings.Contains(result, `alert(&#34;xss&#34;)`) &&
		!strings.Contains(result, `alert(&quot;xss&quot;)`) {
		t.Errorf("double-quote inside script payload must be escaped; got: %s", result)
	}

	// html.EscapeString replacements
	escaped := map[string]string{
		"<":  "&lt;",
		">":  "&gt;",
		"&":  "&amp;",
		"\"": "&#34;",
	}
	for raw, want := range escaped {
		// Only check characters that appear in the input (the outer SSML tags are allowed)
		if raw == "<" || raw == ">" {
			// The input text's < and > should be escaped. We verify the string
			// does NOT contain the raw form from the text portion.
			// Since the surrounding SSML uses <speak> / <prosody> we need a more
			// targeted check: the escaped text payload should not contain the raw
			// "<script>" or other injection patterns.
			if strings.Contains(result, "<script>") {
				t.Errorf("raw %q must be escaped in SSML output, found in: %s", raw, result)
			}
		}
		if raw == "&" {
			if !strings.Contains(result, want) {
				t.Errorf("expected %q to be escaped to %q in SSML, got: %s", raw, want, result)
			}
		}
	}

	// Verify expected escaped entities ARE present in the output
	if !strings.Contains(result, "&lt;script&gt;") {
		t.Errorf("expected &lt;script&gt; in SSML output, got: %s", result)
	}
	if !strings.Contains(result, "&amp;") {
		t.Errorf("expected &amp; in SSML output, got: %s", result)
	}
}

func TestBuildSSML_LtGtAmpersandAndQuoteEscaped(t *testing.T) {
	// Dedicated targeted test for all four special chars.
	inputs := []struct {
		char    string
		escaped string
	}{
		{"<", "&lt;"},
		{">", "&gt;"},
		{"&", "&amp;"},
		{`"`, "&#34;"},
	}
	for _, tc := range inputs {
		result := BuildSSML("x"+tc.char+"y", nil)
		if !strings.Contains(result, tc.escaped) {
			t.Errorf("char %q must be escaped to %q in SSML; got: %s", tc.char, tc.escaped, result)
		}
	}
}

func TestBuildSSML_NilPersonaUsesDefaults(t *testing.T) {
	result := BuildSSML("hello", nil)

	if !strings.Contains(result, `rate="100%"`) {
		t.Errorf("expected default rate=100%% in SSML; got: %s", result)
	}
	if !strings.Contains(result, `pitch="medium"`) {
		t.Errorf("expected default pitch=medium in SSML; got: %s", result)
	}
}

func TestBuildSSML_PersonaWithEmptyProsodyUsesDefaults(t *testing.T) {
	persona := &models.Persona{
		PersonaID: "empty-prosody",
		// VoiceConfig.Prosody is zero-value (empty strings)
	}
	result := BuildSSML("hello", persona)

	if !strings.Contains(result, `rate="100%"`) {
		t.Errorf("expected default rate=100%% when prosody empty; got: %s", result)
	}
	if !strings.Contains(result, `pitch="medium"`) {
		t.Errorf("expected default pitch=medium when prosody empty; got: %s", result)
	}
}

func TestBuildSSML_PersonaProsodyApplied(t *testing.T) {
	persona := makePersonaWithProsody("slow", "low")
	result := BuildSSML("test text", persona)

	if !strings.Contains(result, `rate="slow"`) {
		t.Errorf("expected rate=slow from persona; got: %s", result)
	}
	if !strings.Contains(result, `pitch="low"`) {
		t.Errorf("expected pitch=low from persona; got: %s", result)
	}
}

func TestBuildSSML_WrapsInSpeakAndProsodyAndDomain(t *testing.T) {
	result := BuildSSML("some text", nil)

	if !strings.HasPrefix(result, "<speak>") {
		t.Errorf("SSML must start with <speak>; got: %s", result)
	}
	if !strings.HasSuffix(result, "</speak>") {
		t.Errorf("SSML must end with </speak>; got: %s", result)
	}
	if !strings.Contains(result, "<prosody") {
		t.Errorf("SSML must contain <prosody>; got: %s", result)
	}
	if !strings.Contains(result, `<amazon:domain name="conversational">`) {
		t.Errorf("SSML must contain conversational domain; got: %s", result)
	}
	if !strings.Contains(result, "some text") {
		t.Errorf("SSML must contain the original text; got: %s", result)
	}
}

func TestBuildSSML_OnlyRateSetUsesDefaultPitch(t *testing.T) {
	persona := &models.Persona{
		PersonaID: "rate-only",
		VoiceConfig: models.VoiceConfig{
			Prosody: models.Prosody{
				Rate:  "fast",
				Pitch: "", // empty → default
			},
		},
	}
	result := BuildSSML("hi", persona)
	if !strings.Contains(result, `rate="fast"`) {
		t.Errorf("expected rate=fast; got: %s", result)
	}
	if !strings.Contains(result, `pitch="medium"`) {
		t.Errorf("expected default pitch=medium when pitch empty; got: %s", result)
	}
}

func TestBuildSSML_OnlyPitchSetUsesDefaultRate(t *testing.T) {
	persona := &models.Persona{
		PersonaID: "pitch-only",
		VoiceConfig: models.VoiceConfig{
			Prosody: models.Prosody{
				Rate:  "", // empty → default
				Pitch: "high",
			},
		},
	}
	result := BuildSSML("hi", persona)
	if !strings.Contains(result, `rate="100%"`) {
		t.Errorf("expected default rate=100%%; got: %s", result)
	}
	if !strings.Contains(result, `pitch="high"`) {
		t.Errorf("expected pitch=high; got: %s", result)
	}
}

// ---------------------------------------------------------------------------
// BuildSuccessResponse tests
// ---------------------------------------------------------------------------

func TestBuildSuccessResponse_DialogActionElicitIntent(t *testing.T) {
	persona := makePersonaWithProsody("medium", "medium")
	attrs := map[string]string{"key": "val"}

	resp := BuildSuccessResponse(persona, "Hello!", attrs)

	if resp.SessionState.DialogAction.Type != "ElicitIntent" {
		t.Errorf("expected DialogAction=ElicitIntent, got %q", resp.SessionState.DialogAction.Type)
	}
}

func TestBuildSuccessResponse_ContentTypeSSML(t *testing.T) {
	resp := BuildSuccessResponse(nil, "text", nil)

	if len(resp.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(resp.Messages))
	}
	if resp.Messages[0].ContentType != "SSML" {
		t.Errorf("expected ContentType=SSML, got %q", resp.Messages[0].ContentType)
	}
}

func TestBuildSuccessResponse_PassesSessionAttributes(t *testing.T) {
	attrs := map[string]string{"session_id": "abc123", "step": "2"}
	resp := BuildSuccessResponse(nil, "text", attrs)

	if resp.SessionState.SessionAttributes["session_id"] != "abc123" {
		t.Errorf("session_id not passed through; got %v", resp.SessionState.SessionAttributes)
	}
	if resp.SessionState.SessionAttributes["step"] != "2" {
		t.Errorf("step not passed through; got %v", resp.SessionState.SessionAttributes)
	}
}

func TestBuildSuccessResponse_NilSessionAttrs(t *testing.T) {
	// Must not panic
	resp := BuildSuccessResponse(nil, "text", nil)
	// SessionAttributes may be nil; just ensure the response is well-formed
	if resp.SessionState.DialogAction.Type != "ElicitIntent" {
		t.Errorf("expected ElicitIntent; got %q", resp.SessionState.DialogAction.Type)
	}
}

func TestBuildSuccessResponse_NoIntentSet(t *testing.T) {
	resp := BuildSuccessResponse(nil, "text", nil)
	if resp.SessionState.Intent != nil {
		t.Errorf("BuildSuccessResponse should not set Intent, got %+v", resp.SessionState.Intent)
	}
}

// ---------------------------------------------------------------------------
// BuildErrorResponse tests
// ---------------------------------------------------------------------------

func TestBuildErrorResponse_DialogActionElicitIntent(t *testing.T) {
	resp := BuildErrorResponse(nil, "Something went wrong")

	if resp.SessionState.DialogAction.Type != "ElicitIntent" {
		t.Errorf("expected ElicitIntent, got %q", resp.SessionState.DialogAction.Type)
	}
}

func TestBuildErrorResponse_ContentTypeSSML(t *testing.T) {
	resp := BuildErrorResponse(nil, "error text")

	if len(resp.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(resp.Messages))
	}
	if resp.Messages[0].ContentType != "SSML" {
		t.Errorf("expected ContentType=SSML, got %q", resp.Messages[0].ContentType)
	}
}

func TestBuildErrorResponse_NoSessionAttributes(t *testing.T) {
	resp := BuildErrorResponse(nil, "error")
	// BuildErrorResponse does not accept session attributes – must not set any
	if len(resp.SessionState.SessionAttributes) != 0 {
		t.Errorf("expected empty session attributes, got %v", resp.SessionState.SessionAttributes)
	}
}

func TestBuildErrorResponse_WithPersona(t *testing.T) {
	persona := makePersonaWithProsody("slow", "low")
	resp := BuildErrorResponse(persona, "An error occurred")

	if len(resp.Messages) == 0 {
		t.Fatal("expected a message")
	}
	ssml := resp.Messages[0].Content
	if !strings.Contains(ssml, `rate="slow"`) {
		t.Errorf("persona prosody not applied in error response; got: %s", ssml)
	}
}

// ---------------------------------------------------------------------------
// BuildCloseResponse tests
// ---------------------------------------------------------------------------

func TestBuildCloseResponse_DialogActionClose(t *testing.T) {
	resp := BuildCloseResponse(nil, "Goodbye", nil)

	if resp.SessionState.DialogAction.Type != "Close" {
		t.Errorf("expected DialogAction=Close, got %q", resp.SessionState.DialogAction.Type)
	}
}

func TestBuildCloseResponse_IntentFulfilled(t *testing.T) {
	resp := BuildCloseResponse(nil, "Goodbye", nil)

	if resp.SessionState.Intent == nil {
		t.Fatal("expected Intent to be set")
	}
	if resp.SessionState.Intent.Name != "TroubleshootIntent" {
		t.Errorf("expected Intent.Name=TroubleshootIntent, got %q", resp.SessionState.Intent.Name)
	}
	if resp.SessionState.Intent.State != "Fulfilled" {
		t.Errorf("expected Intent.State=Fulfilled, got %q", resp.SessionState.Intent.State)
	}
}

func TestBuildCloseResponse_ContentTypeSSML(t *testing.T) {
	resp := BuildCloseResponse(nil, "Goodbye", nil)

	if len(resp.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(resp.Messages))
	}
	if resp.Messages[0].ContentType != "SSML" {
		t.Errorf("expected ContentType=SSML, got %q", resp.Messages[0].ContentType)
	}
}

func TestBuildCloseResponse_PassesSessionAttributes(t *testing.T) {
	attrs := map[string]string{"foo": "bar"}
	resp := BuildCloseResponse(nil, "Bye", attrs)

	if resp.SessionState.SessionAttributes["foo"] != "bar" {
		t.Errorf("session attrs not passed through; got %v", resp.SessionState.SessionAttributes)
	}
}

func TestBuildCloseResponse_WithPersona(t *testing.T) {
	persona := makePersonaWithProsody("fast", "high")
	resp := BuildCloseResponse(persona, "See you!", nil)

	ssml := resp.Messages[0].Content
	if !strings.Contains(ssml, `rate="fast"`) {
		t.Errorf("persona rate not in close response SSML; got: %s", ssml)
	}
}

// ---------------------------------------------------------------------------
// BuildTestResponse tests
// ---------------------------------------------------------------------------

func TestBuildTestResponse_DialogActionClose(t *testing.T) {
	resp := BuildTestResponse()

	if resp.SessionState.DialogAction.Type != "Close" {
		t.Errorf("expected Close, got %q", resp.SessionState.DialogAction.Type)
	}
}

func TestBuildTestResponse_ContentTypePlainText(t *testing.T) {
	resp := BuildTestResponse()

	if len(resp.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(resp.Messages))
	}
	if resp.Messages[0].ContentType != "PlainText" {
		t.Errorf("expected PlainText, got %q", resp.Messages[0].ContentType)
	}
}

func TestBuildTestResponse_HealthCheckContent(t *testing.T) {
	resp := BuildTestResponse()

	if resp.Messages[0].Content != "Health check successful" {
		t.Errorf("expected 'Health check successful', got %q", resp.Messages[0].Content)
	}
}

func TestBuildTestResponse_IntentFulfilled(t *testing.T) {
	resp := BuildTestResponse()

	if resp.SessionState.Intent == nil {
		t.Fatal("expected Intent to be set")
	}
	if resp.SessionState.Intent.State != "Fulfilled" {
		t.Errorf("expected Fulfilled, got %q", resp.SessionState.Intent.State)
	}
}

func TestBuildTestResponse_NoSessionAttributes(t *testing.T) {
	resp := BuildTestResponse()
	if len(resp.SessionState.SessionAttributes) != 0 {
		t.Errorf("expected no session attributes in test response, got %v", resp.SessionState.SessionAttributes)
	}
}
