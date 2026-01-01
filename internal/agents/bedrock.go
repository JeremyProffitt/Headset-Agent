package agents

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
	"github.com/headset-support-agent/internal/models"
)

// BedrockClient wraps the Bedrock Agent Runtime client
type BedrockClient struct {
	client *bedrockagentruntime.Client
}

// NewBedrockClient creates a new Bedrock client
func NewBedrockClient(cfg aws.Config) *BedrockClient {
	return &BedrockClient{
		client: bedrockagentruntime.NewFromConfig(cfg),
	}
}

// InvokeAgentInput contains parameters for agent invocation
type InvokeAgentInput struct {
	AgentID      string
	AgentAliasID string
	SessionID    string
	InputText    string
	Persona      *models.Persona
}

// InvokeAgent invokes the Bedrock supervisor agent with persona context
func (c *BedrockClient) InvokeAgent(ctx context.Context, input InvokeAgentInput) (*models.AgentResponse, error) {
	// Build session attributes with persona context
	sessionAttrs := map[string]string{
		"persona_id":   input.Persona.PersonaID,
		"persona_name": input.Persona.DisplayName,
	}

	// Log the invocation for debugging
	fmt.Printf("Invoking agent %s with input: %s\n", input.AgentID, input.InputText)

	// Invoke the agent
	invokeInput := &bedrockagentruntime.InvokeAgentInput{
		AgentId:      aws.String(input.AgentID),
		AgentAliasId: aws.String(input.AgentAliasID),
		SessionId:    aws.String(input.SessionID),
		InputText:    aws.String(input.InputText),
		EnableTrace:  aws.Bool(true),
	}

	// Only add session state if we have attributes
	if len(sessionAttrs) > 0 {
		invokeInput.SessionState = &types.SessionState{
			PromptSessionAttributes: sessionAttrs,
		}
	}

	output, err := c.client.InvokeAgent(ctx, invokeInput)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke agent: %w", err)
	}

	// Collect response from stream
	var responseText strings.Builder
	stream := output.GetStream()

	for event := range stream.Events() {
		switch v := event.(type) {
		case *types.ResponseStreamMemberChunk:
			responseText.Write(v.Value.Bytes)
		case *types.ResponseStreamMemberTrace:
			// Log trace information for debugging
			fmt.Printf("Trace event received: %+v\n", v.Value)
		}
	}

	// Clean up the response - remove any JSON function call artifacts
	cleanedResponse := cleanResponse(responseText.String())
	fmt.Printf("Agent response: %s\n", cleanedResponse)

	if err := stream.Close(); err != nil {
		return nil, fmt.Errorf("error closing stream: %w", err)
	}

	return &models.AgentResponse{
		OutputText: cleanedResponse,
		SessionID:  input.SessionID,
		Metadata:   sessionAttrs,
	}, nil
}

// cleanResponse removes any JSON function call artifacts from the response
func cleanResponse(response string) string {
	// Remove JSON function calls that some models include
	// These can look like: {"name": ...} or {{"name": ...}}
	patterns := []string{
		"\n{{\"name\":",
		"\n{\"name\":",
		" {{\"name\":",
		" {\"name\":",
		"{{\"name\":",
		"{\"name\":",
	}
	for _, pattern := range patterns {
		if idx := strings.Index(response, pattern); idx != -1 {
			response = strings.TrimSpace(response[:idx])
			break
		}
	}
	return response
}

// buildSystemContext creates the enhanced system prompt with persona and troubleshooting context
func buildSystemContext(p *models.Persona) string {
	return fmt.Sprintf(`%s

TROUBLESHOOTING KNOWLEDGE:
You have access to knowledge bases for:
- USB headset troubleshooting
- Bluetooth headset troubleshooting
- Windows audio configuration
- Genesys Cloud Desktop (Chrome) configuration

BEHAVIOR GUIDELINES:
- Always maintain your persona while delivering technical guidance
- Adapt the technical steps to match your personality and speech patterns
- Keep responses under 3 sentences for voice clarity
- Ask ONE question at a time
- Confirm each step before proceeding
- Use layman's terms unless user indicates technical expertise

VOICE OPTIMIZATION:
- Your responses will be spoken using %s voice
- Avoid technical jargon that sounds awkward when spoken
- Use natural conversation patterns
- Include appropriate pauses with "..."
`, p.SystemPrompt, p.VoiceConfig.PollyVoiceID)
}

// GetAgentStatus checks if the agent is available
func (c *BedrockClient) GetAgentStatus(ctx context.Context) error {
	agentID := os.Getenv("SUPERVISOR_AGENT_ID")
	if agentID == "" {
		return fmt.Errorf("SUPERVISOR_AGENT_ID not configured")
	}
	return nil
}
