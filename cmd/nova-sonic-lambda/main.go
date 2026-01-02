package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/headset-support-agent/internal/agents"
	"github.com/headset-support-agent/internal/models"
	"github.com/headset-support-agent/internal/persona"
)

var (
	agentClient   *agents.BedrockClient
	personaLoader *persona.Loader
	ssmClient     *ssm.Client
)

// NovaSonicRequest represents an incoming Nova Sonic streaming request
type NovaSonicRequest struct {
	SessionID   string               `json:"sessionId"`
	PersonaID   string               `json:"personaId"`
	AudioChunk  *models.AudioChunk   `json:"audioChunk,omitempty"`
	TextInput   string               `json:"textInput,omitempty"`
	Action      string               `json:"action"` // "start", "audio", "text", "end"
	Config      *models.NovaSonicConfig `json:"config,omitempty"`
}

// NovaSonicResponse represents the streaming response
type NovaSonicResponse struct {
	SessionID    string             `json:"sessionId"`
	AudioChunk   *models.AudioChunk `json:"audioChunk,omitempty"`
	TextOutput   string             `json:"textOutput,omitempty"`
	Transcript   string             `json:"transcript,omitempty"`
	Status       string             `json:"status"` // "processing", "speaking", "complete", "error"
	Error        string             `json:"error,omitempty"`
}

func init() {
	ctx := context.Background()
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}

	agentClient = agents.NewBedrockClient(cfg)
	ssmClient = ssm.NewFromConfig(cfg)

	tableName := os.Getenv("PERSONA_TABLE_NAME")
	if tableName == "" {
		tableName = "PersonaConfigurations"
	}
	personaLoader = persona.NewLoader(cfg, tableName)
}

// getDefaultNovaSonicConfig returns default Nova Sonic configuration
func getDefaultNovaSonicConfig() *models.NovaSonicConfig {
	return &models.NovaSonicConfig{
		ModelID:        "amazon.nova-sonic-v1:0",
		VoiceID:        "tiffany",
		SampleRate:     24000,
		SampleSizeBits: 16,
		ChannelCount:   1,
		MaxTokens:      1024,
		Temperature:    0.7,
		TopP:           0.9,
	}
}

// mapPersonaToNovaSonicVoice maps persona IDs to Nova Sonic voice IDs
func mapPersonaToNovaSonicVoice(personaID string) string {
	voiceMap := map[string]string{
		"tangerine": "amy",     // British female (closest to Irish)
		"joseph":    "matthew", // US male
		"jennifer":  "tiffany", // US female
	}
	if voice, ok := voiceMap[personaID]; ok {
		return voice
	}
	return "tiffany" // default
}

// handleStreamingRequest handles Nova Sonic streaming requests
func handleStreamingRequest(ctx context.Context, request NovaSonicRequest) (*NovaSonicResponse, error) {
	log.Printf("=== NOVA SONIC HANDLER START ===")
	log.Printf("SessionID: %s, Action: %s, PersonaID: %s",
		request.SessionID, request.Action, request.PersonaID)

	// Load persona configuration
	personaID := request.PersonaID
	if personaID == "" {
		personaID = os.Getenv("DEFAULT_PERSONA")
		if personaID == "" {
			personaID = "tangerine"
		}
	}

	p, err := personaLoader.Load(ctx, personaID)
	if err != nil {
		log.Printf("Error loading persona %s: %v, using default", personaID, err)
		p = persona.DefaultPersona()
	}

	// Get Nova Sonic config
	novaSonicConfig := request.Config
	if novaSonicConfig == nil {
		novaSonicConfig = getDefaultNovaSonicConfig()
	}

	// Map persona to voice
	if p.VoiceConfig.NovaSonicVoice != "" {
		novaSonicConfig.VoiceID = p.VoiceConfig.NovaSonicVoice
	} else {
		novaSonicConfig.VoiceID = mapPersonaToNovaSonicVoice(personaID)
	}

	switch request.Action {
	case "start":
		return handleSessionStart(ctx, request, p, novaSonicConfig)
	case "audio":
		return handleAudioInput(ctx, request, p, novaSonicConfig)
	case "text":
		return handleTextInput(ctx, request, p, novaSonicConfig)
	case "end":
		return handleSessionEnd(ctx, request)
	default:
		return &NovaSonicResponse{
			SessionID: request.SessionID,
			Status:    "error",
			Error:     "Invalid action: " + request.Action,
		}, nil
	}
}

// handleSessionStart initializes a new Nova Sonic session
func handleSessionStart(ctx context.Context, request NovaSonicRequest, p *models.Persona, config *models.NovaSonicConfig) (*NovaSonicResponse, error) {
	log.Printf("Starting Nova Sonic session: %s", request.SessionID)

	// Generate welcome message audio
	welcomeMessage := "Hello! I'm your headset support assistant. How can I help you today?"
	if len(p.Phrases.Greeting) > 0 {
		welcomeMessage = p.Phrases.Greeting[0]
	}

	// For now, return text response - actual Nova Sonic streaming would be implemented here
	return &NovaSonicResponse{
		SessionID:  request.SessionID,
		TextOutput: welcomeMessage,
		Status:     "speaking",
	}, nil
}

// handleAudioInput processes incoming audio chunks
func handleAudioInput(ctx context.Context, request NovaSonicRequest, p *models.Persona, config *models.NovaSonicConfig) (*NovaSonicResponse, error) {
	if request.AudioChunk == nil {
		return &NovaSonicResponse{
			SessionID: request.SessionID,
			Status:    "error",
			Error:     "No audio chunk provided",
		}, nil
	}

	log.Printf("Processing audio chunk for session: %s", request.SessionID)

	// Decode audio from base64
	_, err := base64.StdEncoding.DecodeString(request.AudioChunk.Content)
	if err != nil {
		return &NovaSonicResponse{
			SessionID: request.SessionID,
			Status:    "error",
			Error:     "Invalid audio encoding",
		}, nil
	}

	// TODO: Implement actual Nova Sonic bidirectional streaming
	// This would involve:
	// 1. Sending audio to Nova Sonic via InvokeModelWithBidirectionalStream
	// 2. Receiving transcription and response audio
	// 3. Streaming response back to caller

	return &NovaSonicResponse{
		SessionID: request.SessionID,
		Status:    "processing",
	}, nil
}

// handleTextInput processes text input (for testing/fallback)
func handleTextInput(ctx context.Context, request NovaSonicRequest, p *models.Persona, config *models.NovaSonicConfig) (*NovaSonicResponse, error) {
	if request.TextInput == "" {
		return &NovaSonicResponse{
			SessionID: request.SessionID,
			Status:    "error",
			Error:     "No text input provided",
		}, nil
	}

	log.Printf("Processing text input for session: %s - '%s'", request.SessionID, request.TextInput)

	// Get agent config from SSM
	agentIDParam := os.Getenv("SUPERVISOR_AGENT_ID_PARAM")
	agentAliasParam := os.Getenv("SUPERVISOR_AGENT_ALIAS_PARAM")

	var agentID, agentAlias string
	if agentIDParam != "" {
		idResult, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{Name: &agentIDParam})
		if err == nil && idResult.Parameter.Value != nil {
			agentID = *idResult.Parameter.Value
		}
	}
	if agentAliasParam != "" {
		aliasResult, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{Name: &agentAliasParam})
		if err == nil && aliasResult.Parameter.Value != nil {
			agentAlias = *aliasResult.Parameter.Value
		}
	}

	// Check if agent is configured
	if agentID == "" || agentID == "PLACEHOLDER" || agentAlias == "" || agentAlias == "PLACEHOLDER" {
		return &NovaSonicResponse{
			SessionID:  request.SessionID,
			TextOutput: "I'm setting up right now. Please try again in a few minutes.",
			Status:     "complete",
		}, nil
	}

	// Invoke Bedrock agent
	response, err := agentClient.InvokeAgent(ctx, agents.InvokeAgentInput{
		AgentID:      agentID,
		AgentAliasID: agentAlias,
		SessionID:    request.SessionID,
		InputText:    request.TextInput,
		Persona:      p,
	})
	if err != nil {
		log.Printf("Error invoking Bedrock agent: %v", err)
		return &NovaSonicResponse{
			SessionID: request.SessionID,
			Status:    "error",
			Error:     "Failed to process request",
		}, nil
	}

	return &NovaSonicResponse{
		SessionID:  request.SessionID,
		TextOutput: response.OutputText,
		Transcript: request.TextInput,
		Status:     "complete",
	}, nil
}

// handleSessionEnd closes a Nova Sonic session
func handleSessionEnd(ctx context.Context, request NovaSonicRequest) (*NovaSonicResponse, error) {
	log.Printf("Ending Nova Sonic session: %s", request.SessionID)

	return &NovaSonicResponse{
		SessionID: request.SessionID,
		Status:    "complete",
	}, nil
}

// handleRequest is the main Lambda handler
func handleRequest(ctx context.Context, event json.RawMessage) (interface{}, error) {
	// Try to parse as API Gateway event (for function URL)
	var apiEvent events.APIGatewayV2HTTPRequest
	if err := json.Unmarshal(event, &apiEvent); err == nil && apiEvent.RequestContext.HTTP.Method != "" {
		// Parse the request body
		var request NovaSonicRequest
		if err := json.Unmarshal([]byte(apiEvent.Body), &request); err != nil {
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 400,
				Body:       `{"error": "Invalid request body"}`,
			}, nil
		}

		response, err := handleStreamingRequest(ctx, request)
		if err != nil {
			return events.APIGatewayV2HTTPResponse{
				StatusCode: 500,
				Body:       `{"error": "Internal error"}`,
			}, nil
		}

		respBody, _ := json.Marshal(response)
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type":                "application/json",
				"Access-Control-Allow-Origin": "*",
			},
			Body: string(respBody),
		}, nil
	}

	// Direct Lambda invocation
	var request NovaSonicRequest
	if err := json.Unmarshal(event, &request); err != nil {
		log.Printf("Error parsing event: %v", err)
		return nil, err
	}

	return handleStreamingRequest(ctx, request)
}

func main() {
	lambda.Start(handleRequest)
}
