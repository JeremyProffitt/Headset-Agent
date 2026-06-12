// Package logging provides structured, PII-safe logging helpers for the
// Headset Support Agent.  All log output is emitted as JSON via log/slog.
//
// PII conventions enforced by this package:
//   - Phone numbers / ANI: always pass through HashANI before logging.
//   - Transcript text: only log at DEBUG level via slog.Debug; truncate to ~80
//     chars using Truncate(text, 80).  Never log transcript text at INFO+.
//   - contactId / sessionId are NOT PII and may be logged at any level.
package logging

import (
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
)

// saltWarnOnce ensures the "ANI_HASH_SALT not set" warning is emitted at most once.
var saltWarnOnce sync.Once

// fallbackSalt is used when ANI_HASH_SALT env var is empty.
// Using a constant is intentional (WS-C-13 will wire the real SSM-sourced salt
// into the env var; until then we degrade gracefully rather than fail).
const fallbackSalt = "headset-agent-default-salt-v1"

// Init configures the global slog default logger with a JSON handler writing
// to stdout.  Level is read from the LOG_LEVEL env var (DEBUG/INFO/WARN/ERROR,
// case-insensitive); defaults to INFO.
//
// Call Init() from each Lambda's init() function before any logging occurs.
func Init() {
	level := parseLevel(os.Getenv("LOG_LEVEL"))
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(h))
}

// parseLevel converts a LOG_LEVEL string to a slog.Level.
// Returns slog.LevelInfo for any unrecognised or empty value.
func parseLevel(s string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// HashANI returns a deterministic, salted SHA-256 hex digest of the given ANI
// (phone number).  Returns "" for empty input so callers can safely pass
// optional ANI fields.
//
// Salt source (in priority order):
//  1. ANI_HASH_SALT environment variable
//  2. Compiled-in fallbackSalt constant (a WARN is emitted once)
func HashANI(ani string) string {
	if ani == "" {
		return ""
	}

	salt := os.Getenv("ANI_HASH_SALT")
	if salt == "" {
		saltWarnOnce.Do(func() {
			slog.Warn("ANI_HASH_SALT env var is not set; using built-in fallback salt — set the variable for production deployments")
		})
		salt = fallbackSalt
	}

	h := sha256.Sum256([]byte(salt + ani))
	return fmt.Sprintf("%x", h)
}

// Truncate returns the first n bytes of s.  If s is longer than n, it appends
// "…" (UTF-8 ellipsis, 3 bytes) to signal truncation.  If n <= 0, returns "".
func Truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
