package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"

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

func handleRequest(ctx context.Context, event LexV2Event) (handlers.LexV2Response, error) {
	log.Printf("Received Lex event: sessionId=%s, transcript=%s", event.SessionID, event.InputTranscript)

	// Initialize session attributes if nil
	if event.SessionState.SessionAttributes == nil {
		event.SessionState.SessionAttributes = make(map[string]string)
	}

	// Check for test invocation
	if event.InputTranscript == "" && event.SessionState.SessionAttributes["test"] == "true" {
		return handlers.BuildTestResponse(), nil
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
