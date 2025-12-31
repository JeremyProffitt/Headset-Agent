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
		"persona_id":     input.Persona.PersonaID,
		"persona_name":   input.Persona.DisplayName,
		"persona_voice":  input.Persona.VoiceConfig.PollyVoiceID,
		"system_context": buildSystemContext(input.Persona),
	}

	// Invoke the agent
	output, err := c.client.InvokeAgent(ctx, &bedrockagentruntime.InvokeAgentInput{
		AgentId:      aws.String(input.AgentID),
		AgentAliasId: aws.String(input.AgentAliasID),
		SessionId:    aws.String(input.SessionID),
		InputText:    aws.String(input.InputText),
		SessionState: &types.SessionState{
			PromptSessionAttributes: sessionAttrs,
		},
	})
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
		}
	}

	if err := stream.Close(); err != nil {
		return nil, fmt.Errorf("error closing stream: %w", err)
	}

	return &models.AgentResponse{
		OutputText: responseText.String(),
		SessionID:  input.SessionID,
		Metadata:   sessionAttrs,
	}, nil
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
