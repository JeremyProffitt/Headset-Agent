package persona

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/headset-support-agent/internal/models"
)

// mockDynamo satisfies dynamoAPI for testing.
type mockDynamo struct {
	getItemOutput *dynamodb.GetItemOutput
	getItemErr    error

	putItemInput *dynamodb.PutItemInput // records the most recent call
	putItemErr   error
}

func (m *mockDynamo) GetItem(_ context.Context, params *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return m.getItemOutput, m.getItemErr
}

func (m *mockDynamo) PutItem(_ context.Context, params *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	m.putItemInput = params
	return &dynamodb.PutItemOutput{}, m.putItemErr
}

// loaderWithMock constructs a Loader that bypasses NewLoader's real AWS client.
func loaderWithMock(mock dynamoAPI) *Loader {
	return &Loader{
		client:    mock,
		tableName: "test-persona-table",
	}
}

// ---- DefaultPersona tests -----------------------------------------------

func TestDefaultPersona_Fields(t *testing.T) {
	p := DefaultPersona()

	if p.PersonaID != "default" {
		t.Errorf("PersonaID = %q; want %q", p.PersonaID, "default")
	}
	if p.VoiceConfig.PollyVoiceID != "Joanna" {
		t.Errorf("PollyVoiceID = %q; want %q", p.VoiceConfig.PollyVoiceID, "Joanna")
	}
	if p.VoiceConfig.Prosody.Rate != "100%" {
		t.Errorf("Prosody.Rate = %q; want %q", p.VoiceConfig.Prosody.Rate, "100%")
	}
	if len(p.Phrases.Greeting) == 0 {
		t.Error("Phrases.Greeting must not be empty")
	}
	if len(p.Phrases.Confirmation) == 0 {
		t.Error("Phrases.Confirmation must not be empty")
	}
	if len(p.Phrases.Encouragement) == 0 {
		t.Error("Phrases.Encouragement must not be empty")
	}
	if len(p.Phrases.Empathy) == 0 {
		t.Error("Phrases.Empathy must not be empty")
	}
	if len(p.Phrases.Escalation) == 0 {
		t.Error("Phrases.Escalation must not be empty")
	}
	if p.SystemPrompt == "" {
		t.Error("SystemPrompt must not be empty")
	}
	if len(p.FillerPhrases) == 0 {
		t.Error("FillerPhrases must not be empty")
	}
}

// ---- Load tests ----------------------------------------------------------

// TestLoad_Hit verifies that a found DynamoDB item is unmarshalled and returned.
func TestLoad_Hit(t *testing.T) {
	want := &models.Persona{
		PersonaID:    "tangerine",
		DisplayName:  "Tangerine Agent",
		SystemPrompt: "You are a friendly assistant.",
		VoiceConfig: models.VoiceConfig{
			PollyVoiceID: "Matthew",
			PollyEngine:  "neural",
			LanguageCode: "en-US",
			Prosody:      models.Prosody{Rate: "90%", Pitch: "low"},
		},
	}

	item, err := attributevalue.MarshalMap(want)
	if err != nil {
		t.Fatalf("test setup: MarshalMap failed: %v", err)
	}

	loader := loaderWithMock(&mockDynamo{
		getItemOutput: &dynamodb.GetItemOutput{Item: item},
	})

	got, err := loader.Load(context.Background(), "tangerine")
	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("Load returned nil persona")
	}
	if got.PersonaID != want.PersonaID {
		t.Errorf("PersonaID = %q; want %q", got.PersonaID, want.PersonaID)
	}
	if got.DisplayName != want.DisplayName {
		t.Errorf("DisplayName = %q; want %q", got.DisplayName, want.DisplayName)
	}
	if got.VoiceConfig.PollyVoiceID != want.VoiceConfig.PollyVoiceID {
		t.Errorf("PollyVoiceID = %q; want %q", got.VoiceConfig.PollyVoiceID, want.VoiceConfig.PollyVoiceID)
	}
}

// TestLoad_Miss verifies that a missing item (nil Item map) returns DefaultPersona.
func TestLoad_Miss(t *testing.T) {
	loader := loaderWithMock(&mockDynamo{
		getItemOutput: &dynamodb.GetItemOutput{Item: nil},
	})

	got, err := loader.Load(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Load returned unexpected error on miss: %v", err)
	}
	if got == nil {
		t.Fatal("Load returned nil persona on miss")
	}
	// A miss must return the default persona.
	dflt := DefaultPersona()
	if got.PersonaID != dflt.PersonaID {
		t.Errorf("miss: PersonaID = %q; want default %q", got.PersonaID, dflt.PersonaID)
	}
}

// TestLoad_Error verifies that a DynamoDB error is propagated as-is.
func TestLoad_Error(t *testing.T) {
	sentinel := errors.New("dynamo connection refused")
	loader := loaderWithMock(&mockDynamo{
		getItemErr: sentinel,
	})

	got, err := loader.Load(context.Background(), "any")
	if got != nil {
		t.Errorf("Load should return nil persona on error; got %+v", got)
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("Load error = %v; want sentinel %v", err, sentinel)
	}
}

// TestLoad_Malformed verifies that a malformed DynamoDB item triggers an UnmarshalMap
// error, which is returned as nil persona + non-nil error.
// We produce a malformed item by forcing the "age" field (int in the model) to be a
// String AttributeValue — attributevalue.UnmarshalMap will reject the type mismatch.
func TestLoad_Malformed(t *testing.T) {
	// Build a minimally valid item but set the nested "age" to a String type,
	// which is incompatible with the int field in models.Personality.
	//
	// The Personality field is mapped as a DynamoDB Map; override it so that
	// age (an int) carries an S (string) value — UnmarshalMap will return an error.
	badPersonality := map[string]types.AttributeValue{
		"age": &types.AttributeValueMemberS{Value: "not-a-number"},
	}
	item := map[string]types.AttributeValue{
		"persona_id":  &types.AttributeValueMemberS{Value: "bad"},
		"personality": &types.AttributeValueMemberM{Value: badPersonality},
	}

	loader := loaderWithMock(&mockDynamo{
		getItemOutput: &dynamodb.GetItemOutput{Item: item},
	})

	got, err := loader.Load(context.Background(), "bad")
	if got != nil {
		t.Errorf("Load should return nil persona on malformed item; got %+v", got)
	}
	if err == nil {
		t.Error("Load should return an error for malformed item; got nil")
	}
}

// ---- SavePersona tests ---------------------------------------------------

// TestSavePersona verifies that SavePersona calls PutItem with the correct
// TableName and a properly marshalled item.
func TestSavePersona(t *testing.T) {
	mock := &mockDynamo{}
	loader := loaderWithMock(mock)

	p := &models.Persona{
		PersonaID:   "blue",
		DisplayName: "Blue Agent",
		VoiceConfig: models.VoiceConfig{
			PollyVoiceID: "Kendra",
			PollyEngine:  "standard",
			LanguageCode: "en-US",
		},
	}

	if err := loader.SavePersona(context.Background(), p); err != nil {
		t.Fatalf("SavePersona returned unexpected error: %v", err)
	}

	if mock.putItemInput == nil {
		t.Fatal("PutItem was never called")
	}

	// Verify TableName
	if aws.ToString(mock.putItemInput.TableName) != "test-persona-table" {
		t.Errorf("TableName = %q; want %q",
			aws.ToString(mock.putItemInput.TableName), "test-persona-table")
	}

	// Verify the round-trip: unmarshal the stored item back into a Persona
	var roundTrip models.Persona
	if err := attributevalue.UnmarshalMap(mock.putItemInput.Item, &roundTrip); err != nil {
		t.Fatalf("UnmarshalMap on stored item failed: %v", err)
	}
	if roundTrip.PersonaID != p.PersonaID {
		t.Errorf("round-trip PersonaID = %q; want %q", roundTrip.PersonaID, p.PersonaID)
	}
	if roundTrip.DisplayName != p.DisplayName {
		t.Errorf("round-trip DisplayName = %q; want %q", roundTrip.DisplayName, p.DisplayName)
	}
	if roundTrip.VoiceConfig.PollyVoiceID != p.VoiceConfig.PollyVoiceID {
		t.Errorf("round-trip PollyVoiceID = %q; want %q",
			roundTrip.VoiceConfig.PollyVoiceID, p.VoiceConfig.PollyVoiceID)
	}
}

// TestSavePersona_Error verifies that a PutItem error is propagated.
func TestSavePersona_Error(t *testing.T) {
	sentinel := errors.New("put failed")
	loader := loaderWithMock(&mockDynamo{putItemErr: sentinel})

	err := loader.SavePersona(context.Background(), DefaultPersona())
	if !errors.Is(err, sentinel) {
		t.Errorf("SavePersona error = %v; want sentinel %v", err, sentinel)
	}
}
