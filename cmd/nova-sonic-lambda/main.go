package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/headset-support-agent/internal/agents"
	"github.com/headset-support-agent/internal/logging"
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
	SessionID  string                  `json:"sessionId"`
	PersonaID  string                  `json:"personaId"`
	AudioChunk *models.AudioChunk      `json:"audioChunk,omitempty"`
	TextInput  string                  `json:"textInput,omitempty"`
	Action     string                  `json:"action"` // "start", "audio", "text", "end"
	Config     *models.NovaSonicConfig `json:"config,omitempty"`
}

// ConnectContactFlowEvent represents an Amazon Connect contact flow Lambda invocation
type ConnectContactFlowEvent struct {
	Name    string                    `json:"Name"`
	Details ConnectContactFlowDetails `json:"Details"`
}

// ConnectContactFlowDetails contains the contact details from Connect
type ConnectContactFlowDetails struct {
	ContactData ConnectContactData `json:"ContactData"`
	Parameters  map[string]string  `json:"Parameters"`
}

// ConnectContactData contains contact information
type ConnectContactData struct {
	Attributes        map[string]string `json:"Attributes"`
	Channel           string            `json:"Channel"`
	ContactID         string            `json:"ContactId"`
	CustomerEndpoint  *ConnectEndpoint  `json:"CustomerEndpoint"`
	InitialContactID  string            `json:"InitialContactId"`
	InitiationMethod  string            `json:"InitiationMethod"`
	InstanceARN       string            `json:"InstanceARN"`
	MediaStreams      *MediaStreams     `json:"MediaStreams"`
	PreviousContactID string            `json:"PreviousContactId"`
	Queue             *ConnectQueue     `json:"Queue"`
	SystemEndpoint    *ConnectEndpoint  `json:"SystemEndpoint"`
}

// ConnectEndpoint represents a phone endpoint
type ConnectEndpoint struct {
	Address string `json:"Address"`
	Type    string `json:"Type"`
}

// ConnectQueue represents a Connect queue
type ConnectQueue struct {
	ARN  string `json:"ARN"`
	Name string `json:"Name"`
}

// MediaStreams contains media streaming information
type MediaStreams struct {
	Customer CustomerMediaStreams `json:"Customer"`
}

// CustomerMediaStreams contains customer audio stream info
type CustomerMediaStreams struct {
	Audio CustomerAudio `json:"Audio"`
}

// CustomerAudio contains audio stream details
type CustomerAudio struct {
	StartFragmentNumber string `json:"StartFragmentNumber"`
	StartTimestamp      string `json:"StartTimestamp"`
	StreamARN           string `json:"StreamARN"`
}

// ConnectLambdaResponse is the response format for Connect contact flows
type ConnectLambdaResponse struct {
	TextOutput          string `json:"textOutput,omitempty"`
	Status              string `json:"status,omitempty"`
	EscalationRequested string `json:"escalation_requested,omitempty"`
}

// NovaSonicResponse represents the streaming response
type NovaSonicResponse struct {
	SessionID  string             `json:"sessionId"`
	AudioChunk *models.AudioChunk `json:"audioChunk,omitempty"`
	TextOutput string             `json:"textOutput,omitempty"`
	Transcript string             `json:"transcript,omitempty"`
	Status     string             `json:"status"` // "processing", "speaking", "complete", "error"
	Error      string             `json:"error,omitempty"`
}

func init() {
	logging.Init()

	ctx := context.Background()
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		slog.Error("failed to load AWS config", slog.String("error", err.Error()))
		panic("failed to load AWS config: " + err.Error())
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
	slog.Info("nova sonic handler start",
		slog.String("session_id", request.SessionID),
		slog.String("action", request.Action),
		slog.String("persona_id", request.PersonaID),
	)

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
		slog.Warn("error loading persona; using default",
			slog.String("persona_id", personaID),
			slog.String("error", err.Error()),
		)
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
	slog.Info("starting nova sonic session", slog.String("session_id", request.SessionID))

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

	slog.Info("processing audio chunk", slog.String("session_id", request.SessionID))

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

	slog.Info("processing text input", slog.String("session_id", request.SessionID))
	slog.Debug("text input content",
		slog.String("session_id", request.SessionID),
		slog.String("text", logging.Truncate(request.TextInput, 80)),
	)

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
		slog.Error("error invoking bedrock agent",
			slog.String("session_id", request.SessionID),
			slog.String("error", err.Error()),
		)
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
	slog.Info("ending nova sonic session", slog.String("session_id", request.SessionID))

	return &NovaSonicResponse{
		SessionID: request.SessionID,
		Status:    "complete",
	}, nil
}

// handleConnectContactFlow handles Amazon Connect contact flow Lambda invocations
func handleConnectContactFlow(ctx context.Context, event ConnectContactFlowEvent) (*ConnectLambdaResponse, error) {
	// contactId is not PII — safe to log at INFO.
	// CustomerEndpoint.Address IS a phone number (ANI) — hash it before logging.
	var aniHash string
	if event.Details.ContactData.CustomerEndpoint != nil {
		aniHash = logging.HashANI(event.Details.ContactData.CustomerEndpoint.Address)
	}
	slog.Info("connect contact flow handler start",
		slog.String("contact_id", event.Details.ContactData.ContactID),
		slog.String("channel", event.Details.ContactData.Channel),
		slog.String("initiation_method", event.Details.ContactData.InitiationMethod),
		slog.String("ani_hash", aniHash),
	)

	// Extract persona from contact attributes
	personaID := ""
	if event.Details.ContactData.Attributes != nil {
		personaID = event.Details.ContactData.Attributes["persona_id"]
	}
	if personaID == "" {
		personaID = os.Getenv("DEFAULT_PERSONA")
		if personaID == "" {
			personaID = "tangerine"
		}
	}
	slog.Info("using persona", slog.String("persona_id", personaID))

	// Load persona configuration
	p, err := personaLoader.Load(ctx, personaID)
	if err != nil {
		slog.Warn("error loading persona; using default",
			slog.String("persona_id", personaID),
			slog.String("error", err.Error()),
		)
		p = persona.DefaultPersona()
	}

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
		slog.Warn("supervisor agent not configured",
			slog.String("contact_id", event.Details.ContactData.ContactID),
			slog.String("agent_id", agentID),
			slog.String("alias_id", agentAlias),
		)
		// Return a greeting message while agent is being configured
		greeting := "Hello! I'm your headset support assistant. The system is currently being set up. Please try again in a few minutes."
		if p != nil && len(p.Phrases.Greeting) > 0 {
			greeting = p.Phrases.Greeting[0] + " I'm still getting set up. Please call back in a few minutes."
		}
		return &ConnectLambdaResponse{
			TextOutput: greeting,
			Status:     "complete",
		}, nil
	}

	// For initial contact flow invocation, return a greeting
	// In a full implementation, this would handle bidirectional streaming with Nova Sonic
	greeting := "Hello! I'm your headset support assistant. How can I help you today?"
	if p != nil && len(p.Phrases.Greeting) > 0 {
		greeting = p.Phrases.Greeting[0]
	}

	slog.Info("returning greeting to connect", slog.String("contact_id", event.Details.ContactData.ContactID))
	return &ConnectLambdaResponse{
		TextOutput:          greeting,
		Status:              "ready",
		EscalationRequested: "false",
	}, nil
}

// handleRequest is the main Lambda handler
func handleRequest(ctx context.Context, event json.RawMessage) (interface{}, error) {
	// NOTE: Do NOT log the raw event body — it may contain ANI or transcript text.
	// The event type is detected by inspection; each handler logs its own entry point.

	// Try to parse as API Gateway event (for function URL)
	var apiEvent events.APIGatewayV2HTTPRequest
	if err := json.Unmarshal(event, &apiEvent); err == nil && apiEvent.RequestContext.HTTP.Method != "" {
		slog.Info("detected API gateway V2 event")
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

	// Try to parse as Amazon Connect contact flow event
	var connectEvent ConnectContactFlowEvent
	if err := json.Unmarshal(event, &connectEvent); err == nil && connectEvent.Details.ContactData.ContactID != "" {
		slog.Info("detected amazon connect contact flow event")
		return handleConnectContactFlow(ctx, connectEvent)
	}

	// Direct Lambda invocation with NovaSonicRequest
	var request NovaSonicRequest
	if err := json.Unmarshal(event, &request); err != nil {
		slog.Error("error parsing event as nova sonic request", slog.String("error", err.Error()))
		// Return a generic error response that Connect can handle
		return &ConnectLambdaResponse{
			TextOutput: "I'm sorry, I couldn't process that request. Please try again.",
			Status:     "error",
		}, nil
	}

	slog.Info("processing direct nova sonic request invocation")
	return handleStreamingRequest(ctx, request)
}

func main() {
	lambda.Start(handleRequest)
}
