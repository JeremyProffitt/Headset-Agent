// Package session provides a DynamoDB-backed store for conversation sessions
// used by the Headset Support Agent.
//
// Design notes:
//   - Store mirrors the dynamoAPI interface pattern from internal/persona so
//     the concrete *dynamodb.Client can be swapped with a mock in tests.
//   - Session.Attributes is a free-form map[string]string that carries both
//     the TriageState projections (using the triage.Attr* key consts) and the
//     VX-layer scalars (pace_rate, low_asr_count, no_match_count) defined as
//     additional key consts below.
//   - Save uses a conditional write on last_activity so a stale concurrent
//     turn cannot clobber a newer write; callers detect this via ErrConcurrentUpdate.
package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/headset-support-agent/internal/models"
	"github.com/headset-support-agent/internal/triage"
)

// SessionTTL is the maximum lifetime of a session in DynamoDB (SEC-6: ≤ 24 h).
const SessionTTL = 24 * time.Hour

// ErrConcurrentUpdate is returned by Save when the conditional write fails
// because another Lambda invocation updated last_activity after this one read
// the session. The caller should reload the session and retry.
var ErrConcurrentUpdate = errors.New("session: concurrent update detected — reload and retry")

// ---------------------------------------------------------------------------
// Session-attribute key constants
// ---------------------------------------------------------------------------
//
// Keys that exist in internal/triage (Attr* consts) are re-exported here as
// aliases so that handlers (B-02) and the triage engine share one vocabulary
// without importing triage solely for string constants.
//
// The VX-layer keys (pace_rate, low_asr_count, no_match_count, last_response,
// attempted_steps, unclear_streak, reboot_count, driver_reinstalled) are NOT
// in triage's Attr* set (those are Lex-mirror hot-path consts only) so they
// are defined here.

// Keys re-exported from triage for handler convenience.
const (
	KeyCurrentTree      = triage.AttrCurrentTree      // "current_tree"
	KeyCurrentStep      = triage.AttrCurrentStep      // "current_step"
	KeySymptom          = triage.AttrSymptom          // "symptom"
	KeyFailedSteps      = triage.AttrFailedSteps      // "failed_steps"
	KeyFrustrationCount = triage.AttrFrustrationCount // "frustration_count"
	KeyRebootCount      = triage.AttrRebootCount      // "reboot_count"
	KeyEscalationReason = triage.AttrEscalationReason // "escalation_reason"
	KeyResolved         = triage.AttrResolved         // "resolved"
	KeyEscalated        = triage.AttrEscalated        // "escalated"
)

// Keys that live in the broader session but are not part of the Lex-mirror
// hot subset (managed here, not in triage).
const (
	// KeyAttemptedSteps stores the ordered []StepID list as a JSON array string.
	KeyAttemptedSteps = "attempted_steps"
	// KeyDriverReinstalled stores a bool ("true"/"false").
	KeyDriverReinstalled = "driver_reinstalled"
	// KeyUnclearStreak counts consecutive unclear answers on the current step.
	KeyUnclearStreak = "unclear_streak"
	// KeyLastResponse is the last rendered step text (for repeat/pace replay).
	KeyLastResponse = "last_response"
	// KeyPaceRate is the user's preferred speech pace (int, 50–200 %).
	KeyPaceRate = "pace_rate"
	// KeyLowASRCount counts low-confidence ASR turns (VX-5).
	KeyLowASRCount = "low_asr_count"
	// KeyNoMatchCount counts intent no-match events (VX-6).
	KeyNoMatchCount = "no_match_count"

	// B-07 Lex-slot values (frozen A-02 controlled vocabulary). Persisted so a
	// value captured on one turn keeps filtering retrieval on later turns.
	// KeyConnectionType is one of usb/bluetooth/dect/wireless_dongle ("" = unknown).
	KeyConnectionType = "connection_type"
	// KeyBrand is one of jabra/poly/logitech/epos/yealink ("" = unknown). This
	// matches the KB sidecar metadata key "brand" (the Lex slot is named
	// "headset_brand"; the handler normalizes to this key).
	KeyBrand = "brand"
	// KeyIssueType is the symptom class resolved from the issue_type slot
	// (maps 1:1 onto triage SymptomClass values; "" = unknown).
	KeyIssueType = "issue_type"
)

// ---------------------------------------------------------------------------
// DynamoDB interface (testability)
// ---------------------------------------------------------------------------

// dynamoAPI is the subset of the DynamoDB client used by Store. The interface
// allows tests to inject a mock without hitting AWS, mirroring the pattern in
// internal/persona/persona.go.
type dynamoAPI interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

// Store is a DynamoDB-backed session store.
type Store struct {
	client    dynamoAPI
	tableName string
}

// NewStore creates a Store backed by a real *dynamodb.Client derived from cfg.
// The tableName is typically read from the SESSION_TABLE_NAME environment variable.
func NewStore(cfg aws.Config, tableName string) *Store {
	return &Store{
		client:    dynamodb.NewFromConfig(cfg),
		tableName: tableName,
	}
}

// ---------------------------------------------------------------------------
// Load
// ---------------------------------------------------------------------------

// Load retrieves the session identified by sessionID from DynamoDB.
// If the item does not exist a fresh zero-value Session with that ID is
// returned (not an error) so callers can treat first-turn and resume-turn
// uniformly. A genuine DynamoDB error is propagated as-is.
func (s *Store) Load(ctx context.Context, sessionID string) (*models.Session, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"session_id": &types.AttributeValueMemberS{Value: sessionID},
		},
	})
	if err != nil {
		slog.Error("session.Load: GetItem failed", "session_id", sessionID, "err", err)
		return nil, fmt.Errorf("session.Load %s: %w", sessionID, err)
	}

	// Item absent → fresh session (first turn or expired TTL).
	if out.Item == nil {
		slog.Info("session.Load: no existing session; creating fresh", "session_id", sessionID)
		now := time.Now().UTC().Format(time.RFC3339)
		return &models.Session{
			SessionID:  sessionID,
			Attributes: make(map[string]string),
			CreatedAt:  now,
		}, nil
	}

	var sess models.Session
	if err := attributevalue.UnmarshalMap(out.Item, &sess); err != nil {
		slog.Error("session.Load: UnmarshalMap failed", "session_id", sessionID, "err", err)
		return nil, fmt.Errorf("session.Load %s: unmarshal: %w", sessionID, err)
	}

	// Ensure Attributes is never nil after a round-trip.
	if sess.Attributes == nil {
		sess.Attributes = make(map[string]string)
	}

	return &sess, nil
}

// ---------------------------------------------------------------------------
// Save
// ---------------------------------------------------------------------------

// Save persists the session to DynamoDB, refreshing TTL and LastActivity.
//
// Concurrent-turn safety: a conditional PutItem is used so that a stale write
// (one whose loaded LastActivity is older than what is already in the table)
// fails fast instead of clobbering a newer update. The condition is:
//
//	attribute_not_exists(session_id)
//	OR last_activity = :loaded_last_activity
//
// On a ConditionalCheckFailedException the function returns ErrConcurrentUpdate
// so the caller can reload and retry.
func (s *Store) Save(ctx context.Context, sess *models.Session) error {
	now := time.Now().UTC()
	sess.TTL = now.Add(SessionTTL).Unix()
	sess.LastActivity = now.Format(time.RFC3339)

	if sess.Attributes == nil {
		sess.Attributes = make(map[string]string)
	}

	item, err := attributevalue.MarshalMap(sess)
	if err != nil {
		return fmt.Errorf("session.Save %s: marshal: %w", sess.SessionID, err)
	}

	// Conditional write: only succeed if the row is new OR the last_activity
	// value we loaded is still the current value in the table.
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(session_id) OR last_activity = :loaded_last"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			// Use the LastActivity we set above — if another writer already
			// bumped it, the condition will not match.
			":loaded_last": &types.AttributeValueMemberS{Value: sess.LastActivity},
		},
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			slog.Warn("session.Save: conditional check failed — concurrent update",
				"session_id", sess.SessionID)
			return ErrConcurrentUpdate
		}
		slog.Error("session.Save: PutItem failed", "session_id", sess.SessionID, "err", err)
		return fmt.Errorf("session.Save %s: %w", sess.SessionID, err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Generic typed accessors over Session.Attributes
// ---------------------------------------------------------------------------

// GetString returns the raw string value for key, or "" if absent.
func GetString(sess *models.Session, key string) string {
	if sess.Attributes == nil {
		return ""
	}
	return sess.Attributes[key]
}

// SetString stores a string value for key.
func SetString(sess *models.Session, key, value string) {
	if sess.Attributes == nil {
		sess.Attributes = make(map[string]string)
	}
	sess.Attributes[key] = value
}

// GetInt returns the integer stored at key, or 0 if absent or unparseable.
func GetInt(sess *models.Session, key string) int {
	v := GetString(sess, key)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}

// SetInt stores an integer value for key.
func SetInt(sess *models.Session, key string, value int) {
	SetString(sess, key, strconv.Itoa(value))
}

// GetBool returns the bool stored at key ("true"/"false"), or false if absent.
func GetBool(sess *models.Session, key string) bool {
	return GetString(sess, key) == "true"
}

// SetBool stores a bool value for key.
func SetBool(sess *models.Session, key string, value bool) {
	if value {
		SetString(sess, key, "true")
	} else {
		SetString(sess, key, "false")
	}
}

// GetStringSlice decodes a JSON-array string stored at key into a []string.
// Returns nil if the key is absent or decoding fails.
func GetStringSlice(sess *models.Session, key string) []string {
	raw := GetString(sess, key)
	if raw == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		slog.Warn("session.GetStringSlice: JSON decode failed",
			"key", key, "err", err)
		return nil
	}
	return out
}

// SetStringSlice JSON-encodes a []string and stores it at key.
func SetStringSlice(sess *models.Session, key string, values []string) {
	if values == nil {
		SetString(sess, key, "")
		return
	}
	b, err := json.Marshal(values)
	if err != nil {
		slog.Error("session.SetStringSlice: JSON encode failed",
			"key", key, "err", err)
		return
	}
	SetString(sess, key, string(b))
}

// ---------------------------------------------------------------------------
// Named convenience accessors (B-02 / engine calling surface)
// ---------------------------------------------------------------------------

// GetCurrentTree returns the active tree ID (e.g. "preflight", "tree2").
func GetCurrentTree(sess *models.Session) string { return GetString(sess, KeyCurrentTree) }

// SetCurrentTree sets the active tree ID.
func SetCurrentTree(sess *models.Session, tree string) { SetString(sess, KeyCurrentTree, tree) }

// GetCurrentStep returns the active StepID within the current tree.
func GetCurrentStep(sess *models.Session) string { return GetString(sess, KeyCurrentStep) }

// SetCurrentStep sets the active StepID.
func SetCurrentStep(sess *models.Session, step string) { SetString(sess, KeyCurrentStep, step) }

// GetAttemptedSteps returns the ordered list of StepIDs already presented.
// The slice is decoded from a JSON-array string stored in Attributes.
func GetAttemptedSteps(sess *models.Session) []string {
	return GetStringSlice(sess, KeyAttemptedSteps)
}

// SetAttemptedSteps replaces the attempted-steps list.
func SetAttemptedSteps(sess *models.Session, steps []string) {
	SetStringSlice(sess, KeyAttemptedSteps, steps)
}

// AppendAttemptedStep appends stepID to the attempted-steps list.
func AppendAttemptedStep(sess *models.Session, stepID string) {
	existing := GetAttemptedSteps(sess)
	SetAttemptedSteps(sess, append(existing, stepID))
}

// GetFailedSteps returns the count of steps that did not resolve the issue.
func GetFailedSteps(sess *models.Session) int { return GetInt(sess, KeyFailedSteps) }

// SetFailedSteps sets the failed-steps count.
func SetFailedSteps(sess *models.Session, n int) { SetInt(sess, KeyFailedSteps, n) }

// GetFrustrationCount returns the accumulated frustration delta.
func GetFrustrationCount(sess *models.Session) int { return GetInt(sess, KeyFrustrationCount) }

// SetFrustrationCount sets the frustration counter.
func SetFrustrationCount(sess *models.Session, n int) { SetInt(sess, KeyFrustrationCount, n) }

// GetRebootCount returns the number of full reboots performed this session.
func GetRebootCount(sess *models.Session) int { return GetInt(sess, KeyRebootCount) }

// SetRebootCount sets the reboot count.
func SetRebootCount(sess *models.Session, n int) { SetInt(sess, KeyRebootCount, n) }

// GetDriverReinstalled returns whether a driver reinstall has been done.
func GetDriverReinstalled(sess *models.Session) bool { return GetBool(sess, KeyDriverReinstalled) }

// SetDriverReinstalled sets the driver-reinstalled flag.
func SetDriverReinstalled(sess *models.Session, v bool) { SetBool(sess, KeyDriverReinstalled, v) }

// GetUnclearStreak returns the count of consecutive unclear answers on the current step.
func GetUnclearStreak(sess *models.Session) int { return GetInt(sess, KeyUnclearStreak) }

// SetUnclearStreak sets the unclear-answer streak counter.
func SetUnclearStreak(sess *models.Session, n int) { SetInt(sess, KeyUnclearStreak, n) }

// GetLastResponse returns the last rendered step text (for repeat/pace replay).
func GetLastResponse(sess *models.Session) string { return GetString(sess, KeyLastResponse) }

// SetLastResponse sets the last rendered step text.
func SetLastResponse(sess *models.Session, text string) { SetString(sess, KeyLastResponse, text) }

// GetPaceRate returns the user's preferred speech pace percentage (VX-5).
func GetPaceRate(sess *models.Session) int { return GetInt(sess, KeyPaceRate) }

// SetPaceRate sets the speech pace rate.
func SetPaceRate(sess *models.Session, rate int) { SetInt(sess, KeyPaceRate, rate) }

// GetLowASRCount returns the low-confidence ASR turn count (VX-5).
func GetLowASRCount(sess *models.Session) int { return GetInt(sess, KeyLowASRCount) }

// SetLowASRCount sets the low-ASR count.
func SetLowASRCount(sess *models.Session, n int) { SetInt(sess, KeyLowASRCount, n) }

// GetNoMatchCount returns the intent no-match event count (VX-6).
func GetNoMatchCount(sess *models.Session) int { return GetInt(sess, KeyNoMatchCount) }

// SetNoMatchCount sets the no-match count.
func SetNoMatchCount(sess *models.Session, n int) { SetInt(sess, KeyNoMatchCount, n) }
