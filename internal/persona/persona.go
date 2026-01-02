package persona

import (
	"context"
	"log"

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

// Load retrieves a persona by ID from DynamoDB
func (l *Loader) Load(ctx context.Context, personaID string) (*models.Persona, error) {
	result, err := l.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(l.tableName),
		Key: map[string]types.AttributeValue{
			"persona_id": &types.AttributeValueMemberS{Value: personaID},
		},
	})
	if err != nil {
		log.Printf("Failed to get persona %s: %v", personaID, err)
		return nil, err
	}

	if result.Item == nil {
		log.Printf("Persona %s not found, using default", personaID)
		return DefaultPersona(), nil
	}

	var persona models.Persona
	if err := attributevalue.UnmarshalMap(result.Item, &persona); err != nil {
		log.Printf("Failed to unmarshal persona %s: %v", personaID, err)
		return nil, err
	}

	return &persona, nil
}

// SavePersona stores a persona configuration to DynamoDB
func (l *Loader) SavePersona(ctx context.Context, persona *models.Persona) error {
	item, err := attributevalue.MarshalMap(persona)
	if err != nil {
		return err
	}

	_, err = l.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(l.tableName),
		Item:      item,
	})
	return err
}

// DefaultPersona returns a fallback persona configuration
func DefaultPersona() *models.Persona {
	return &models.Persona{
		PersonaID:   "default",
		DisplayName: "Support Agent",
		VoiceConfig: models.VoiceConfig{
			PollyVoiceID:   "Joanna",
			PollyEngine:    "neural",
			LanguageCode:   "en-US",
			UseNovaSonic:   false,
			NovaSonicVoice: "tiffany",
			Prosody: models.Prosody{
				Rate:  "100%",
				Pitch: "medium",
			},
		},
		Personality: models.Personality{
			Origin:      "United States",
			Age:         30,
			Gender:      "female",
			Traits:      []string{"helpful", "professional", "patient"},
			SpeechStyle: "Clear and professional",
			Pace:        "moderate",
		},
		Phrases: models.Phrases{
			Greeting:      []string{"Hello! I'm here to help you with your headset."},
			Confirmation:  []string{"Got it.", "I understand.", "Okay."},
			Encouragement: []string{"You're doing great!", "Almost there!"},
			Empathy:       []string{"I understand that can be frustrating.", "Let's get this sorted out for you."},
			Escalation:    []string{"Let me connect you with a specialist who can help further."},
		},
		SystemPrompt: `You are a helpful headset troubleshooting assistant. Your role is to help users diagnose and fix issues with their headsets, including USB, Bluetooth, and wireless devices. Be patient, clear, and guide users step-by-step through troubleshooting procedures.`,
		FillerPhrases: []string{"Let me check that for you.", "One moment please."},
	}
}
