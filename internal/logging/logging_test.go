package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// HashANI tests
// ---------------------------------------------------------------------------

func TestHashANI_EmptyReturnsEmpty(t *testing.T) {
	got := HashANI("")
	if got != "" {
		t.Errorf("HashANI(\"\") = %q; want \"\"", got)
	}
}

func TestHashANI_NonEmptyIsNonEmpty(t *testing.T) {
	got := HashANI("+15551234567")
	if got == "" {
		t.Error("HashANI(non-empty) returned empty string")
	}
}

func TestHashANI_NeverEqualsRawInput(t *testing.T) {
	raw := "+15551234567"
	got := HashANI(raw)
	if got == raw {
		t.Errorf("HashANI returned the raw phone number unchanged: %q", got)
	}
}

func TestHashANI_Deterministic(t *testing.T) {
	a := HashANI("+15551234567")
	b := HashANI("+15551234567")
	if a != b {
		t.Errorf("HashANI is not deterministic: %q != %q", a, b)
	}
}

func TestHashANI_DifferentInputsDifferentHashes(t *testing.T) {
	a := HashANI("+15551234567")
	b := HashANI("+15559876543")
	if a == b {
		t.Errorf("different ANIs produced the same hash: %q", a)
	}
}

func TestHashANI_IsHexString(t *testing.T) {
	got := HashANI("+15551234567")
	for _, c := range got {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("HashANI result is not lowercase hex: %q (unexpected char %q)", got, c)
		}
	}
}

// ---------------------------------------------------------------------------
// Truncate tests
// ---------------------------------------------------------------------------

func TestTruncate_ShortStringUnchanged(t *testing.T) {
	got := Truncate("hello", 10)
	if got != "hello" {
		t.Errorf("Truncate(\"hello\", 10) = %q; want \"hello\"", got)
	}
}

func TestTruncate_ExactLengthUnchanged(t *testing.T) {
	got := Truncate("hello", 5)
	if got != "hello" {
		t.Errorf("Truncate(\"hello\", 5) = %q; want \"hello\"", got)
	}
}

func TestTruncate_LongStringTruncated(t *testing.T) {
	got := Truncate("hello world", 5)
	if !strings.HasPrefix(got, "hello") {
		t.Errorf("Truncate(\"hello world\", 5) = %q; want prefix \"hello\"", got)
	}
	if !strings.Contains(got, "…") {
		t.Errorf("Truncate(\"hello world\", 5) = %q; want ellipsis suffix", got)
	}
}

func TestTruncate_ZeroNReturnsEmpty(t *testing.T) {
	got := Truncate("hello", 0)
	if got != "" {
		t.Errorf("Truncate(\"hello\", 0) = %q; want \"\"", got)
	}
}

func TestTruncate_EmptyStringReturnsEmpty(t *testing.T) {
	got := Truncate("", 10)
	if got != "" {
		t.Errorf("Truncate(\"\", 10) = %q; want \"\"", got)
	}
}

func TestTruncate_NegativeNReturnsEmpty(t *testing.T) {
	got := Truncate("hello", -1)
	if got != "" {
		t.Errorf("Truncate(\"hello\", -1) = %q; want \"\"", got)
	}
}

// ---------------------------------------------------------------------------
// PII leak test at INFO level
//
// logTurnAtINFO simulates what the refactored Lambda handlers do for each turn:
//   - session_id logged at INFO (not PII)
//   - ANI logged as hash at INFO (PII-safe)
//   - transcript logged at DEBUG only (truncated)
//
// The test captures the JSON output and asserts the raw phone number is absent
// at INFO, but the hash is present (or the field is simply omitted at INFO).
// ---------------------------------------------------------------------------

// logTurnAtINFO exercises the expected INFO-level log pattern.
// It is a package-level helper so the test can swap in a custom handler.
func logTurnAtINFO(logger *slog.Logger, sessionID, ani, transcript string) {
	// INFO: safe fields only.
	logger.Info("turn received",
		slog.String("session_id", sessionID),
		slog.String("ani_hash", HashANI(ani)),
	)
	// DEBUG: transcript (truncated) — should NOT appear at INFO.
	logger.Debug("transcript",
		slog.String("text", Truncate(transcript, 80)),
	)
}

func TestNoRawPIIInINFOOutput(t *testing.T) {
	const fakePhone = "+15551234567"
	const fakeSession = "sess-abc-123"
	const fakeTranscript = "My headset is crackling and I cannot hear the other party at all."

	// Capture slog output at INFO level into a buffer.
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(h)

	logTurnAtINFO(logger, fakeSession, fakePhone, fakeTranscript)

	output := buf.String()

	// 1. Raw phone number must never appear in INFO-level output.
	if strings.Contains(output, fakePhone) {
		t.Errorf("raw phone number %q found in INFO log output:\n%s", fakePhone, output)
	}

	// 2. The transcript must not appear at INFO (it is only logged at DEBUG).
	if strings.Contains(output, fakeTranscript) {
		t.Errorf("raw transcript found in INFO log output:\n%s", output)
	}

	// 3. session_id IS allowed at INFO.
	if !strings.Contains(output, fakeSession) {
		t.Errorf("session_id %q not found in INFO log output (it should be there):\n%s", fakeSession, output)
	}

	// 4. The ANI hash should appear instead of the raw phone.
	expectedHash := HashANI(fakePhone)
	if !strings.Contains(output, expectedHash) {
		t.Errorf("ANI hash %q not found in INFO log output:\n%s", expectedHash, output)
	}
}

func TestNoTranscriptInINFOEvenAtDebugLevel(t *testing.T) {
	// At DEBUG level the transcript IS emitted. Confirm it appears in DEBUG
	// output but only at that level — this is the positive complement of
	// TestNoRawPIIInINFOOutput.
	const fakePhone = "+15559876543"
	const fakeSession = "sess-debug-456"
	const fakeTranscript = "Short transcript text."

	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(h)

	logTurnAtINFO(logger, fakeSession, fakePhone, fakeTranscript)

	output := buf.String()

	// At DEBUG level the transcript text should appear (it's in a Debug log line).
	if !strings.Contains(output, fakeTranscript) {
		t.Errorf("expected transcript in DEBUG-level output but not found:\n%s", output)
	}

	// Even at DEBUG, raw phone must still not appear (we always hash ANI).
	if strings.Contains(output, fakePhone) {
		t.Errorf("raw phone number %q found in DEBUG log output:\n%s", fakePhone, output)
	}
}
