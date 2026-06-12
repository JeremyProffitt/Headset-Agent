package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/headset-support-agent/internal/agents"
	"github.com/headset-support-agent/internal/handlers"
	"github.com/headset-support-agent/internal/logging"
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
	MessageVersion   string           `json:"messageVersion"`
	InvocationSource string           `json:"invocationSource"`
	InputMode        string           `json:"inputMode"`
	SessionID        string           `json:"sessionId"`
	InputTranscript  string           `json:"inputTranscript"`
	Bot              LexBot           `json:"bot"`
	SessionState     SessionState     `json:"sessionState"`
	Transcriptions   []Transcription  `json:"transcriptions"`
	Interpretations  []Interpretation `json:"interpretations"`
}

// LexBot contains bot information
type LexBot struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	AliasID  string `json:"aliasId"`
	LocaleID string `json:"localeId"`
}

// Transcription contains speech-to-text results
type Transcription struct {
	Transcription           string  `json:"transcription"`
	TranscriptionConfidence float64 `json:"transcriptionConfidence"`
}

// Interpretation contains intent interpretation
type Interpretation struct {
	Intent        IntentResult `json:"intent"`
	NluConfidence float64      `json:"nluConfidence"`
}

// IntentResult contains intent details
type IntentResult struct {
	Name              string                 `json:"name"`
	Slots             map[string]interface{} `json:"slots"`
	State             string                 `json:"state"`
	ConfirmationState string                 `json:"confirmationState"`
}

// SessionState represents session state from Lex
type SessionState struct {
	SessionAttributes map[string]string `json:"sessionAttributes"`
	Intent            *IntentResult     `json:"intent"`
	DialogAction      *DialogAction     `json:"dialogAction"`
}

// DialogAction represents the dialog action
type DialogAction struct {
	Type string `json:"type"`
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
		slog.Warn("SSM parameter paths not configured — SUPERVISOR_AGENT_ID_PARAM/SUPERVISOR_AGENT_ALIAS_PARAM unset")
		return "", ""
	}

	// Read agent ID from SSM
	idResult, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name: &agentIDParam,
	})
	if err != nil {
		slog.Error("failed to get agent ID from SSM",
			slog.String("param", agentIDParam),
			slog.String("error", err.Error()),
		)
		return "", ""
	}
	agentConfig.agentID = *idResult.Parameter.Value

	// Read agent alias from SSM
	aliasResult, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name: &agentAliasParam,
	})
	if err != nil {
		slog.Error("failed to get agent alias from SSM",
			slog.String("param", agentAliasParam),
			slog.String("error", err.Error()),
		)
		return "", ""
	}
	agentConfig.agentAlias = *aliasResult.Parameter.Value

	// Only mark as loaded if both values are valid (not PLACEHOLDER)
	if agentConfig.agentID != "" && agentConfig.agentID != "PLACEHOLDER" &&
		agentConfig.agentAlias != "" && agentConfig.agentAlias != "PLACEHOLDER" {
		agentConfig.loaded = true
	}

	slog.Info("loaded agent config from SSM",
		slog.String("agent_id", agentConfig.agentID),
		slog.String("alias_id", agentConfig.agentAlias),
	)
	return agentConfig.agentID, agentConfig.agentAlias
}

// handleAPIRequest handles HTTP API Gateway requests
func handleAPIRequest(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	slog.Info("received API request",
		slog.String("path", request.RawPath),
		slog.String("method", request.RequestContext.HTTP.Method),
	)

	// Parse the request body
	var chatReq ChatRequest
	if err := json.Unmarshal([]byte(request.Body), &chatReq); err != nil {
		slog.Error("error parsing request body", slog.String("error", err.Error()))
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
		slog.Warn("error loading persona; using default",
			slog.String("persona_id", personaID),
			slog.String("error", err.Error()),
		)
		p = persona.DefaultPersona()
	}

	// Check for payment solicitation BEFORE calling Bedrock.
	// Return a safe refusal without echoing any digits or storing card data.
	if handlers.DetectPaymentSolicitation(chatReq.InputTranscript) {
		slog.Info("payment solicitation detected — returning refusal",
			slog.String("session_id", chatReq.SessionID),
			slog.Bool("payment_blocked", true),
		)
		refusalMsg := "For your security, I can't take any payment or card details — I only help with headset troubleshooting. Please don't share card numbers here. If you need billing help I can connect you to a person."
		chatResp := ChatResponse{
			Messages: []ChatMessage{
				{Content: refusalMsg},
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

	// Load supervisor agent configuration from SSM
	agentID, agentAlias := loadAgentConfig(ctx)

	var responseMessage string
	if agentID == "" || agentID == "PLACEHOLDER" || agentAlias == "" || agentAlias == "PLACEHOLDER" {
		slog.Warn("supervisor agent not configured",
			slog.String("agent_id", agentID),
			slog.String("alias_id", agentAlias),
		)
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
			slog.Error("error invoking Bedrock agent",
				slog.String("session_id", chatReq.SessionID),
				slog.String("error", err.Error()),
			)
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
	slog.Info("lex handler start",
		slog.String("session_id", event.SessionID),
		slog.String("input_mode", event.InputMode),
		slog.String("invocation_source", event.InvocationSource),
	)
	slog.Debug("lex input transcript",
		slog.String("session_id", event.SessionID),
		slog.String("transcript", logging.Truncate(event.InputTranscript, 80)),
	)

	// Extract transcript
	transcript := event.InputTranscript
	if transcript == "" && len(event.Transcriptions) > 0 {
		bestTranscription := event.Transcriptions[0]
		for _, t := range event.Transcriptions[1:] {
			if t.TranscriptionConfidence > bestTranscription.TranscriptionConfidence {
				bestTranscription = t
			}
		}
		transcript = bestTranscription.Transcription
	}

	// Initialize session attributes if nil
	if event.SessionState.SessionAttributes == nil {
		event.SessionState.SessionAttributes = make(map[string]string)
	}

	// Check for test invocation
	if transcript == "" && event.SessionState.SessionAttributes["test"] == "true" {
		return handlers.BuildTestResponse(), nil
	}

	// Handle empty transcript
	if transcript == "" {
		slog.Info("empty transcript received — returning welcome prompt",
			slog.String("session_id", event.SessionID),
		)
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

	// Get persona
	personaID := event.SessionState.SessionAttributes["persona_id"]
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

	// Check for payment solicitation BEFORE escalation and BEFORE calling Bedrock.
	// If detected, return a safe refusal without echoing any digits.
	if handlers.DetectPaymentSolicitation(transcript) {
		slog.Info("payment solicitation detected — returning refusal",
			slog.String("session_id", event.SessionID),
			slog.Bool("payment_blocked", true),
		)
		return handlers.BuildPaymentRefusalResponse(p, event.SessionState.SessionAttributes), nil
	}

	// Check for escalation triggers
	escalationDecision := handlers.DetectEscalation(
		transcript,
		getIntAttr(event.SessionState.SessionAttributes, "frustration_count"),
		getIntAttr(event.SessionState.SessionAttributes, "failed_steps"),
	)

	if escalationDecision.ShouldEscalate {
		return handlers.BuildEscalationResponse(p, escalationDecision, event.SessionState.SessionAttributes), nil
	}

	// Load supervisor agent configuration from SSM
	agentID, agentAlias := loadAgentConfig(ctx)

	if agentID == "" || agentID == "PLACEHOLDER" || agentAlias == "" || agentAlias == "PLACEHOLDER" {
		slog.Warn("supervisor agent not configured",
			slog.String("session_id", event.SessionID),
			slog.String("agent_id", agentID),
			slog.String("alias_id", agentAlias),
		)
		return handlers.BuildSuccessResponse(
			p,
			"Hello! I'm setting up right now. The system is being configured. Please try again in a few minutes.",
			event.SessionState.SessionAttributes,
		), nil
	}

	// Invoke Bedrock supervisor agent — log transcript at DEBUG only (PII risk)
	slog.Info("invoking bedrock agent for lex turn",
		slog.String("session_id", event.SessionID),
	)
	slog.Debug("lex turn transcript",
		slog.String("session_id", event.SessionID),
		slog.String("transcript", logging.Truncate(transcript, 80)),
	)
	response, err := agentClient.InvokeAgent(ctx, agents.InvokeAgentInput{
		AgentID:      agentID,
		AgentAliasID: agentAlias,
		SessionID:    event.SessionID,
		InputText:    transcript,
		Persona:      p,
	})
	if err != nil {
		slog.Error("error invoking bedrock agent",
			slog.String("session_id", event.SessionID),
			slog.String("error", err.Error()),
		)
		return handlers.BuildErrorResponse(p, "I'm having a bit of trouble connecting. Let me try that again."), nil
	}

	if response == nil || response.OutputText == "" {
		slog.Warn("empty response from bedrock agent", slog.String("session_id", event.SessionID))
		return handlers.BuildErrorResponse(p, "I'm sorry, I didn't catch that. Could you please rephrase your question?"), nil
	}

	return handlers.BuildSuccessResponse(p, response.OutputText, event.SessionState.SessionAttributes), nil
}

func handleRequest(ctx context.Context, event json.RawMessage) (interface{}, error) {
	// Try to detect event type
	var apiEvent events.APIGatewayV2HTTPRequest
	if err := json.Unmarshal(event, &apiEvent); err == nil && apiEvent.RequestContext.HTTP.Method != "" {
		return handleAPIRequest(ctx, apiEvent)
	}

	// Fall back to Lex V2 event
	var lexEvent LexV2Event
	if err := json.Unmarshal(event, &lexEvent); err != nil {
		slog.Error("error parsing lambda event", slog.String("error", err.Error()))
		return nil, err
	}
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
