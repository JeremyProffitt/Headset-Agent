package persona

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/headset-support-agent/internal/models"
)

// Loader handles persona loading from DynamoDB
type Loader struct {
	client    *dynamodb.Client
	tableName string
}

// NewLoader creates a new persona loader
func NewLoader(cfg aws.Config, tableName string) *Loader {
	return &Loader{
		client:    dynamodb.NewFromConfig(cfg),
		tableName: tableName,
	}
}

// Load retrieves a persona configuration from DynamoDB
func (l *Loader) Load(ctx context.Context, personaID string) (*models.Persona, error) {
	result, err := l.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(l.tableName),
		Key: map[string]types.AttributeValue{
			"persona_id": &types.AttributeValueMemberS{Value: personaID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get persona %s: %w", personaID, err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("persona %s not found", personaID)
	}

	var persona models.Persona
	err = attributevalue.UnmarshalMap(result.Item, &persona)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal persona: %w", err)
	}

	return &persona, nil
}

// DefaultPersona returns a fallback persona configuration
func DefaultPersona() *models.Persona {
	return &models.Persona{
		PersonaID:   "default",
		DisplayName: "Support Agent",
		VoiceConfig: models.VoiceConfig{
			PollyVoiceID: "Joanna",
			PollyEngine:  "neural",
			LanguageCode: "en-US",
			Prosody: models.Prosody{
				Rate:  "100%",
				Pitch: "medium",
			},
		},
		Personality: models.Personality{
			Origin:      "USA",
			Age:         30,
			Gender:      "female",
			Traits:      []string{"helpful", "professional"},
			SpeechStyle: "neutral",
			Pace:        "normal",
		},
		Phrases: models.Phrases{
			Greeting:      []string{"Hello! I'm here to help you with your headset."},
			Confirmation:  []string{"Great, that worked!"},
			Encouragement: []string{"You're doing well."},
			Empathy:       []string{"I understand that can be frustrating."},
			Escalation:    []string{"Let me connect you with a specialist."},
		},
		SystemPrompt: `You are a helpful technical support agent specializing in headset troubleshooting.
Be friendly, patient, and guide users through troubleshooting steps one at a time.
Keep responses concise and clear for voice interactions.`,
		FillerPhrases: []string{},
	}
}

// SavePersona stores a persona configuration to DynamoDB
func (l *Loader) SavePersona(ctx context.Context, persona *models.Persona) error {
	item, err := attributevalue.MarshalMap(persona)
	if err != nil {
		return fmt.Errorf("failed to marshal persona: %w", err)
	}

	_, err = l.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(l.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to save persona: %w", err)
	}

	return nil
}
