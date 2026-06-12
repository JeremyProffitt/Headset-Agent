package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/headset-support-agent/internal/models"
	"github.com/headset-support-agent/internal/triage"
)

// ---------------------------------------------------------------------------
// Mock DynamoDB client
// ---------------------------------------------------------------------------

type mockDynamo struct {
	getItemOutput *dynamodb.GetItemOutput
	getItemErr    error

	putItemInput *dynamodb.PutItemInput // captures the most recent call
	putItemErr   error
}

func (m *mockDynamo) GetItem(_ context.Context, params *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return m.getItemOutput, m.getItemErr
}

func (m *mockDynamo) PutItem(_ context.Context, params *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	m.putItemInput = params
	return &dynamodb.PutItemOutput{}, m.putItemErr
}

// storeWithMock returns a Store that bypasses NewStore's real AWS client.
func storeWithMock(mock dynamoAPI) *Store {
	return &Store{
		client:    mock,
		tableName: "test-session-table",
	}
}

// ---------------------------------------------------------------------------
// Load tests
// ---------------------------------------------------------------------------

// TestLoad_Hit verifies that a found DynamoDB item is unmarshalled and returned.
func TestLoad_Hit(t *testing.T) {
	want := &models.Session{
		SessionID:    "session-abc",
		PersonaID:    "tangerine",
		Attributes:   map[string]string{"current_tree": "tree2"},
		TTL:          time.Now().Add(10 * time.Hour).Unix(),
		CreatedAt:    "2026-06-12T10:00:00Z",
		LastActivity: "2026-06-12T10:05:00Z",
	}

	item, err := attributevalue.MarshalMap(want)
	if err != nil {
		t.Fatalf("test setup: MarshalMap failed: %v", err)
	}

	store := storeWithMock(&mockDynamo{
		getItemOutput: &dynamodb.GetItemOutput{Item: item},
	})

	got, err := store.Load(context.Background(), "session-abc")
	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("Load returned nil session")
	}
	if got.SessionID != want.SessionID {
		t.Errorf("SessionID = %q; want %q", got.SessionID, want.SessionID)
	}
	if got.PersonaID != want.PersonaID {
		t.Errorf("PersonaID = %q; want %q", got.PersonaID, want.PersonaID)
	}
	if got.Attributes["current_tree"] != "tree2" {
		t.Errorf("Attributes[current_tree] = %q; want %q", got.Attributes["current_tree"], "tree2")
	}
}

// TestLoad_Miss verifies that a missing item returns a fresh Session (not an error).
func TestLoad_Miss(t *testing.T) {
	store := storeWithMock(&mockDynamo{
		getItemOutput: &dynamodb.GetItemOutput{Item: nil},
	})

	got, err := store.Load(context.Background(), "brand-new-session")
	if err != nil {
		t.Fatalf("Load returned unexpected error on miss: %v", err)
	}
	if got == nil {
		t.Fatal("Load returned nil session on miss")
	}
	if got.SessionID != "brand-new-session" {
		t.Errorf("SessionID = %q; want %q", got.SessionID, "brand-new-session")
	}
	if got.Attributes == nil {
		t.Error("fresh session Attributes must not be nil")
	}
	if got.CreatedAt == "" {
		t.Error("fresh session CreatedAt must not be empty")
	}
}

// TestLoad_Miss_AttributesNotNil verifies the nil-map guard on a round-tripped item
// whose Attributes map was empty (DynamoDB omits empty maps in some scenarios).
func TestLoad_Miss_AttributesNotNil(t *testing.T) {
	// Marshal a Session with a nil Attributes map; the round-trip should still
	// give a non-nil map.
	sess := &models.Session{
		SessionID: "nils",
		CreatedAt: "2026-06-12T00:00:00Z",
	}
	item, err := attributevalue.MarshalMap(sess)
	if err != nil {
		t.Fatalf("test setup: MarshalMap failed: %v", err)
	}

	store := storeWithMock(&mockDynamo{
		getItemOutput: &dynamodb.GetItemOutput{Item: item},
	})

	got, err := store.Load(context.Background(), "nils")
	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}
	if got.Attributes == nil {
		t.Error("Load must ensure Attributes is non-nil after round-trip")
	}
}

// TestLoad_Error verifies that a DynamoDB error is propagated as-is.
func TestLoad_Error(t *testing.T) {
	sentinel := errors.New("dynamo unavailable")
	store := storeWithMock(&mockDynamo{getItemErr: sentinel})

	got, err := store.Load(context.Background(), "any")
	if got != nil {
		t.Errorf("Load should return nil on error; got %+v", got)
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("Load error = %v; want sentinel %v", err, sentinel)
	}
}

// ---------------------------------------------------------------------------
// Save tests
// ---------------------------------------------------------------------------

// TestSave_SetsMetadata verifies Save refreshes TTL and LastActivity and issues PutItem.
func TestSave_SetsMetadata(t *testing.T) {
	mock := &mockDynamo{}
	store := storeWithMock(mock)

	before := time.Now().Unix()
	sess := &models.Session{
		SessionID:  "save-test",
		PersonaID:  "blue",
		Attributes: map[string]string{"key": "val"},
		CreatedAt:  "2026-06-12T09:00:00Z",
	}

	if err := store.Save(context.Background(), sess); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}
	after := time.Now().Unix()

	// TTL must be in the future (> now) and ≤ now + 24h + slack.
	if sess.TTL <= before {
		t.Errorf("TTL %d must be > now %d", sess.TTL, before)
	}
	maxTTL := after + int64(SessionTTL.Seconds()) + 2
	if sess.TTL > maxTTL {
		t.Errorf("TTL %d is too far in the future; expected ≤ %d", sess.TTL, maxTTL)
	}

	// LastActivity must be a parseable RFC3339 timestamp close to now.
	if sess.LastActivity == "" {
		t.Error("LastActivity must not be empty after Save")
	}
	la, err := time.Parse(time.RFC3339, sess.LastActivity)
	if err != nil {
		t.Errorf("LastActivity %q is not valid RFC3339: %v", sess.LastActivity, err)
	}
	if la.Before(time.Unix(before, 0).Add(-time.Second)) {
		t.Errorf("LastActivity %v appears too old", la)
	}

	// PutItem must have been called with the correct table.
	if mock.putItemInput == nil {
		t.Fatal("PutItem was never called")
	}
	if aws.ToString(mock.putItemInput.TableName) != "test-session-table" {
		t.Errorf("TableName = %q; want test-session-table",
			aws.ToString(mock.putItemInput.TableName))
	}

	// Verify the round-trip: unmarshal the stored item back into Session.
	var roundTrip models.Session
	if err := attributevalue.UnmarshalMap(mock.putItemInput.Item, &roundTrip); err != nil {
		t.Fatalf("UnmarshalMap on stored item failed: %v", err)
	}
	if roundTrip.SessionID != sess.SessionID {
		t.Errorf("round-trip SessionID = %q; want %q", roundTrip.SessionID, sess.SessionID)
	}
	if roundTrip.Attributes["key"] != "val" {
		t.Errorf("round-trip Attributes[key] = %q; want %q", roundTrip.Attributes["key"], "val")
	}
}

// TestSave_ConditionalExpression verifies the conditional write is set.
func TestSave_ConditionalExpression(t *testing.T) {
	mock := &mockDynamo{}
	store := storeWithMock(mock)
	sess := &models.Session{SessionID: "cond-test", CreatedAt: "2026-06-12T09:00:00Z"}

	if err := store.Save(context.Background(), sess); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}
	if mock.putItemInput == nil {
		t.Fatal("PutItem was never called")
	}
	if mock.putItemInput.ConditionExpression == nil {
		t.Error("Save must set a ConditionExpression for concurrent-turn safety")
	}
	if mock.putItemInput.ExpressionAttributeValues == nil {
		t.Error("Save must set ExpressionAttributeValues for the condition")
	}
}

// TestSave_ConcurrentUpdate verifies that a ConditionalCheckFailedException is
// translated to ErrConcurrentUpdate.
func TestSave_ConcurrentUpdate(t *testing.T) {
	ccfErr := &types.ConditionalCheckFailedException{
		Message: aws.String("The conditional request failed"),
	}
	mock := &mockDynamo{putItemErr: ccfErr}
	store := storeWithMock(mock)
	sess := &models.Session{SessionID: "conflict", CreatedAt: "2026-06-12T09:00:00Z"}

	err := store.Save(context.Background(), sess)
	if err == nil {
		t.Fatal("Save must return an error on ConditionalCheckFailedException")
	}
	if !errors.Is(err, ErrConcurrentUpdate) {
		t.Errorf("Save error = %v; want ErrConcurrentUpdate", err)
	}
}

// TestSave_PropagatesOtherErrors verifies that non-conditional DynamoDB errors
// are wrapped and returned (not swallowed or converted to ErrConcurrentUpdate).
func TestSave_PropagatesOtherErrors(t *testing.T) {
	sentinel := errors.New("table not found")
	mock := &mockDynamo{putItemErr: sentinel}
	store := storeWithMock(mock)
	sess := &models.Session{SessionID: "err-test", CreatedAt: "2026-06-12T09:00:00Z"}

	err := store.Save(context.Background(), sess)
	if err == nil {
		t.Fatal("Save must propagate DynamoDB errors")
	}
	if errors.Is(err, ErrConcurrentUpdate) {
		t.Error("non-conditional error must not be ErrConcurrentUpdate")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("Save error = %v; want sentinel wrapped", err)
	}
}

// ---------------------------------------------------------------------------
// Accessor round-trip tests
// ---------------------------------------------------------------------------

func newTestSession() *models.Session {
	return &models.Session{
		SessionID:  "acc-test",
		Attributes: make(map[string]string),
	}
}

func TestGetSetInt(t *testing.T) {
	sess := newTestSession()

	// Default is 0 for absent key.
	if v := GetInt(sess, "count"); v != 0 {
		t.Errorf("GetInt absent = %d; want 0", v)
	}

	SetInt(sess, "count", 42)
	if v := GetInt(sess, "count"); v != 42 {
		t.Errorf("GetInt = %d; want 42", v)
	}

	SetInt(sess, "count", -7)
	if v := GetInt(sess, "count"); v != -7 {
		t.Errorf("GetInt negative = %d; want -7", v)
	}
}

func TestGetSetBool(t *testing.T) {
	sess := newTestSession()

	// Default false for absent key.
	if v := GetBool(sess, "flag"); v {
		t.Error("GetBool absent must be false")
	}

	SetBool(sess, "flag", true)
	if !GetBool(sess, "flag") {
		t.Error("GetBool after SetBool(true) must be true")
	}

	SetBool(sess, "flag", false)
	if GetBool(sess, "flag") {
		t.Error("GetBool after SetBool(false) must be false")
	}
}

func TestGetSetStringSlice_AttemptedSteps(t *testing.T) {
	sess := newTestSession()

	// Absent → nil.
	if s := GetAttemptedSteps(sess); s != nil {
		t.Errorf("GetAttemptedSteps absent = %v; want nil", s)
	}

	steps := []string{"tree2.s1", "tree2.s2", "tree2.s3"}
	SetAttemptedSteps(sess, steps)

	got := GetAttemptedSteps(sess)
	if len(got) != len(steps) {
		t.Fatalf("GetAttemptedSteps len = %d; want %d", len(got), len(steps))
	}
	for i, want := range steps {
		if got[i] != want {
			t.Errorf("GetAttemptedSteps[%d] = %q; want %q", i, got[i], want)
		}
	}
}

func TestAppendAttemptedStep(t *testing.T) {
	sess := newTestSession()

	AppendAttemptedStep(sess, "preflight.s1")
	AppendAttemptedStep(sess, "preflight.s2")
	AppendAttemptedStep(sess, "tree1.s1")

	got := GetAttemptedSteps(sess)
	if len(got) != 3 {
		t.Fatalf("AppendAttemptedStep: len = %d; want 3", len(got))
	}
	if got[2] != "tree1.s1" {
		t.Errorf("AppendAttemptedStep last = %q; want tree1.s1", got[2])
	}
}

func TestGetSetString(t *testing.T) {
	sess := newTestSession()

	if v := GetString(sess, "k"); v != "" {
		t.Errorf("GetString absent = %q; want empty", v)
	}

	SetString(sess, "k", "hello world")
	if v := GetString(sess, "k"); v != "hello world" {
		t.Errorf("GetString = %q; want hello world", v)
	}
}

func TestGetSetStringSlice_EmptySlice(t *testing.T) {
	sess := newTestSession()
	SetStringSlice(sess, "items", []string{})
	got := GetStringSlice(sess, "items")
	if len(got) != 0 {
		t.Errorf("GetStringSlice empty = %v; want []", got)
	}
}

func TestGetSetStringSlice_NilClearsKey(t *testing.T) {
	sess := newTestSession()
	SetStringSlice(sess, "items", []string{"a", "b"})
	SetStringSlice(sess, "items", nil)
	if v := GetString(sess, "items"); v != "" {
		t.Errorf("SetStringSlice(nil) should clear key; got %q", v)
	}
}

// ---------------------------------------------------------------------------
// Named convenience accessor tests
// ---------------------------------------------------------------------------

func TestConvenienceAccessors(t *testing.T) {
	sess := newTestSession()

	// CurrentTree / CurrentStep
	SetCurrentTree(sess, "tree5")
	if v := GetCurrentTree(sess); v != "tree5" {
		t.Errorf("GetCurrentTree = %q; want tree5", v)
	}
	SetCurrentStep(sess, "tree5.s2")
	if v := GetCurrentStep(sess); v != "tree5.s2" {
		t.Errorf("GetCurrentStep = %q; want tree5.s2", v)
	}

	// FailedSteps
	SetFailedSteps(sess, 3)
	if v := GetFailedSteps(sess); v != 3 {
		t.Errorf("GetFailedSteps = %d; want 3", v)
	}

	// FrustrationCount
	SetFrustrationCount(sess, 2)
	if v := GetFrustrationCount(sess); v != 2 {
		t.Errorf("GetFrustrationCount = %d; want 2", v)
	}

	// RebootCount
	SetRebootCount(sess, 1)
	if v := GetRebootCount(sess); v != 1 {
		t.Errorf("GetRebootCount = %d; want 1", v)
	}

	// DriverReinstalled
	SetDriverReinstalled(sess, true)
	if !GetDriverReinstalled(sess) {
		t.Error("GetDriverReinstalled = false; want true")
	}

	// UnclearStreak
	SetUnclearStreak(sess, 1)
	if v := GetUnclearStreak(sess); v != 1 {
		t.Errorf("GetUnclearStreak = %d; want 1", v)
	}

	// LastResponse
	SetLastResponse(sess, "Please check the volume knob on your headset.")
	if v := GetLastResponse(sess); v != "Please check the volume knob on your headset." {
		t.Errorf("GetLastResponse = %q; unexpected", v)
	}

	// PaceRate
	SetPaceRate(sess, 80)
	if v := GetPaceRate(sess); v != 80 {
		t.Errorf("GetPaceRate = %d; want 80", v)
	}

	// LowASRCount
	SetLowASRCount(sess, 4)
	if v := GetLowASRCount(sess); v != 4 {
		t.Errorf("GetLowASRCount = %d; want 4", v)
	}

	// NoMatchCount
	SetNoMatchCount(sess, 2)
	if v := GetNoMatchCount(sess); v != 2 {
		t.Errorf("GetNoMatchCount = %d; want 2", v)
	}
}

// ---------------------------------------------------------------------------
// Key alignment tests — assert key consts match triage.Attr* exactly
// ---------------------------------------------------------------------------

func TestKeyConsts_AlignWithTriage(t *testing.T) {
	cases := []struct {
		sessionKey string
		triageKey  string
	}{
		{KeyCurrentTree, triage.AttrCurrentTree},
		{KeyCurrentStep, triage.AttrCurrentStep},
		{KeySymptom, triage.AttrSymptom},
		{KeyFailedSteps, triage.AttrFailedSteps},
		{KeyFrustrationCount, triage.AttrFrustrationCount},
		{KeyRebootCount, triage.AttrRebootCount},
		{KeyEscalationReason, triage.AttrEscalationReason},
		{KeyResolved, triage.AttrResolved},
		{KeyEscalated, triage.AttrEscalated},
	}
	for _, c := range cases {
		if c.sessionKey != c.triageKey {
			t.Errorf("key mismatch: session.%s=%q triage.Attr*=%q",
				c.sessionKey, c.sessionKey, c.triageKey)
		}
	}
}
