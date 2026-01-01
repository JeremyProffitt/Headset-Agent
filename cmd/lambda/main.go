package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/headset-support-agent/internal/agents"
	"github.com/headset-support-agent/internal/handlers"
	"github.com/headset-support-agent/internal/persona"
)

var (
	agentClient   *agents.BedrockClient
	personaLoader *persona.Loader
	ssmClient     *ssm.Client
	agentConfig   struct {
		sync.RWMutex
		agentID    string
		agentAlias string
		loaded     bool
	}
)

// LexV2Event represents the incoming Lex V2 event
type LexV2Event struct {
	SessionID       string       `json:"sessionId"`
	InputTranscript string       `json:"inputTranscript"`
	SessionState    SessionState `json:"sessionState"`
}

// SessionState represents session state from Lex
type SessionState struct {
	SessionAttributes map[string]string `json:"sessionAttributes"`
}

// ChatRequest represents the incoming chat API request
type ChatRequest struct {
	SessionID       string       `json:"sessionId"`
	InputTranscript string       `json:"inputTranscript"`
	SessionState    SessionState `json:"sessionState"`
}

// ChatResponse represents the outgoing chat API response
type ChatResponse struct {
	Messages []ChatMessage `json:"messages"`
}

// ChatMessage represents a single message in the response
type ChatMessage struct {
	Content string `json:"content"`
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

// loadAgentConfig reads agent configuration from SSM Parameter Store
func loadAgentConfig(ctx context.Context) (agentID, agentAlias string) {
	agentConfig.RLock()
	if agentConfig.loaded {
		id, alias := agentConfig.agentID, agentConfig.agentAlias
		agentConfig.RUnlock()
		return id, alias
	}
	agentConfig.RUnlock()

	agentConfig.Lock()
	defer agentConfig.Unlock()

	// Double-check after acquiring write lock
	if agentConfig.loaded {
		return agentConfig.agentID, agentConfig.agentAlias
	}

	agentIDParam := os.Getenv("SUPERVISOR_AGENT_ID_PARAM")
	agentAliasParam := os.Getenv("SUPERVISOR_AGENT_ALIAS_PARAM")

	if agentIDParam == "" || agentAliasParam == "" {
		log.Printf("SSM parameter paths not configured")
		return "", ""
	}

	// Read agent ID from SSM
	idResult, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name: &agentIDParam,
	})
	if err != nil {
		log.Printf("Failed to get agent ID from SSM: %v", err)
		return "", ""
	}
	agentConfig.agentID = *idResult.Parameter.Value

	// Read agent alias from SSM
	aliasResult, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name: &agentAliasParam,
	})
	if err != nil {
		log.Printf("Failed to get agent alias from SSM: %v", err)
		return "", ""
	}
	agentConfig.agentAlias = *aliasResult.Parameter.Value

	// Only mark as loaded if both values are valid (not PLACEHOLDER)
	if agentConfig.agentID != "" && agentConfig.agentID != "PLACEHOLDER" &&
		agentConfig.agentAlias != "" && agentConfig.agentAlias != "PLACEHOLDER" {
		agentConfig.loaded = true
	}

	log.Printf("Loaded agent config from SSM: ID=%s, Alias=%s", agentConfig.agentID, agentConfig.agentAlias)
	return agentConfig.agentID, agentConfig.agentAlias
}

// handleAPIRequest handles HTTP API Gateway requests
func handleAPIRequest(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	log.Printf("Received API request: path=%s, method=%s", request.RawPath, request.RequestContext.HTTP.Method)

	// Parse the request body
	var chatReq ChatRequest
	if err := json.Unmarshal([]byte(request.Body), &chatReq); err != nil {
		log.Printf("Error parsing request body: %v", err)
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"error": "Invalid request body"}`,
		}, nil
	}

	// Get persona from request headers or body
	personaID := request.Headers["x-persona-id"]
	if personaID == "" && chatReq.SessionState.SessionAttributes != nil {
		personaID = chatReq.SessionState.SessionAttributes["persona_id"]
	}
	if personaID == "" {
		personaID = os.Getenv("DEFAULT_PERSONA")
		if personaID == "" {
			personaID = "tangerine"
		}
	}

	// Load persona configuration
	p, err := personaLoader.Load(ctx, personaID)
	if err != nil {
		log.Printf("Error loading persona %s: %v, using default", personaID, err)
		p = persona.DefaultPersona()
	}

	// Load supervisor agent configuration from SSM
	agentID, agentAlias := loadAgentConfig(ctx)

	var responseMessage string
	if agentID == "" || agentID == "PLACEHOLDER" || agentAlias == "" || agentAlias == "PLACEHOLDER" {
		log.Printf("Supervisor agent not configured (ID: %s, Alias: %s)", agentID, agentAlias)
		responseMessage = "Hello! I'm setting up right now. The system is being configured. Please try again in a few minutes."
	} else {
		// Invoke Bedrock supervisor agent
		response, err := agentClient.InvokeAgent(ctx, agents.InvokeAgentInput{
			AgentID:      agentID,
			AgentAliasID: agentAlias,
			SessionID:    chatReq.SessionID,
			InputText:    chatReq.InputTranscript,
			Persona:      p,
		})
		if err != nil {
			log.Printf("Error invoking Bedrock agent: %v", err)
			responseMessage = "I'm having a bit of trouble connecting. Let me try that again."
		} else {
			responseMessage = response.OutputText
		}
	}

	// Build response
	chatResp := ChatResponse{
		Messages: []ChatMessage{
			{Content: responseMessage},
		},
	}

	respBody, _ := json.Marshal(chatResp)
	return events.APIGatewayV2HTTPResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":                 "application/json",
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "POST, OPTIONS",
			"Access-Control-Allow-Headers": "Content-Type, X-Session-Id, X-Persona-Id",
		},
		Body: string(respBody),
	}, nil
}

// handleLexRequest handles Lex V2 requests
func handleLexRequest(ctx context.Context, event LexV2Event) (handlers.LexV2Response, error) {
	log.Printf("Received Lex event: sessionId=%s, transcript=%s", event.SessionID, event.InputTranscript)

	// Initialize session attributes if nil
	if event.SessionState.SessionAttributes == nil {
		event.SessionState.SessionAttributes = make(map[string]string)
	}

	// Check for test invocation
	if event.InputTranscript == "" && event.SessionState.SessionAttributes["test"] == "true" {
		return handlers.BuildTestResponse(), nil
	}

	// Handle empty transcript - this happens for initial dialog hook invocations
	// or when voice transcription fails
	if event.InputTranscript == "" {
		log.Printf("Empty transcript received - returning welcome prompt")
		// Load default persona for welcome message
		personaID := event.SessionState.SessionAttributes["persona_id"]
		if personaID == "" {
			personaID = os.Getenv("DEFAULT_PERSONA")
			if personaID == "" {
				personaID = "tangerine"
			}
		}
		p, _ := personaLoader.Load(ctx, personaID)
		if p == nil {
			p = persona.DefaultPersona()
		}
		return handlers.BuildSuccessResponse(
			p,
			"Hi there! I'm your headset support assistant. Please describe your issue and I'll help you troubleshoot.",
			event.SessionState.SessionAttributes,
		), nil
	}

	// Get persona from session attributes, default to configured default
	personaID := event.SessionState.SessionAttributes["persona_id"]
	if personaID == "" {
		personaID = os.Getenv("DEFAULT_PERSONA")
		if personaID == "" {
			personaID = "tangerine"
		}
	}

	// Load persona configuration
	p, err := personaLoader.Load(ctx, personaID)
	if err != nil {
		log.Printf("Error loading persona %s: %v, using default", personaID, err)
		p = persona.DefaultPersona()
	}

	// Check for escalation triggers
	escalationDecision := handlers.DetectEscalation(
		event.InputTranscript,
		getIntAttr(event.SessionState.SessionAttributes, "frustration_count"),
		getIntAttr(event.SessionState.SessionAttributes, "failed_steps"),
	)

	if escalationDecision.ShouldEscalate {
		return handlers.BuildEscalationResponse(p, escalationDecision, event.SessionState.SessionAttributes)
	}

	// Load supervisor agent configuration from SSM
	agentID, agentAlias := loadAgentConfig(ctx)

	if agentID == "" || agentID == "PLACEHOLDER" || agentAlias == "" || agentAlias == "PLACEHOLDER" {
		// Agent not yet configured, return a helpful message
		log.Printf("Supervisor agent not configured (ID: %s, Alias: %s)", agentID, agentAlias)
		return handlers.BuildSuccessResponse(
			p,
			"Hello! I'm setting up right now. The system is being configured. Please try again in a few minutes.",
			event.SessionState.SessionAttributes,
		), nil
	}

	// Invoke Bedrock supervisor agent
	response, err := agentClient.InvokeAgent(ctx, agents.InvokeAgentInput{
		AgentID:      agentID,
		AgentAliasID: agentAlias,
		SessionID:    event.SessionID,
		InputText:    event.InputTranscript,
		Persona:      p,
	})
	if err != nil {
		log.Printf("Error invoking Bedrock agent: %v", err)
		return handlers.BuildErrorResponse(p, "I'm having a bit of trouble connecting. Let me try that again."), nil
	}

	return handlers.BuildSuccessResponse(p, response.OutputText, event.SessionState.SessionAttributes), nil
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// handleRequest is a unified handler that routes to the appropriate handler
func handleRequest(ctx context.Context, event json.RawMessage) (interface{}, error) {
	// Try to detect event type by unmarshaling into different structures
	var apiEvent events.APIGatewayV2HTTPRequest
	if err := json.Unmarshal(event, &apiEvent); err == nil && apiEvent.RequestContext.HTTP.Method != "" {
		log.Printf("Detected API Gateway V2 HTTP event")
		return handleAPIRequest(ctx, apiEvent)
	}

	// Log raw event for debugging Lex V2 issues
	log.Printf("Non-API event received. Raw event (first 1000 chars): %s", truncateString(string(event), 1000))

	// Fall back to Lex V2 event
	var lexEvent LexV2Event
	if err := json.Unmarshal(event, &lexEvent); err != nil {
		log.Printf("Error parsing event: %v", err)
		return nil, err
	}
	log.Printf("Detected Lex V2 event")
	return handleLexRequest(ctx, lexEvent)
}

func getIntAttr(attrs map[string]string, key string) int {
	if val, ok := attrs[key]; ok {
		var n int
		json.Unmarshal([]byte(val), &n)
		return n
	}
	return 0
}

func main() {
	lambda.Start(handleRequest)
}
