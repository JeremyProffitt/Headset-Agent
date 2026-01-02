package agents

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

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

// InvokeAgent invokes the Bedrock supervisor agent
func (c *BedrockClient) InvokeAgent(ctx context.Context, input InvokeAgentInput) (*models.AgentResponse, error) {
	// Set a timeout to ensure we don't exceed Lambda's limit
	// Leave 5 seconds buffer for response processing
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	// Build session attributes with persona context
	sessionAttrs := make(map[string]string)
	if input.Persona != nil {
		sessionAttrs["persona_id"] = input.Persona.PersonaID
		sessionAttrs["persona_name"] = input.Persona.DisplayName
		sessionAttrs["persona_style"] = input.Persona.Personality.SpeechStyle
	}

	// Build enhanced system context
	enhancedInput := buildSystemContext(input.InputText, input.Persona)

	log.Printf("Invoking Bedrock agent: agentId=%s, aliasId=%s, sessionId=%s",
		input.AgentID, input.AgentAliasID, input.SessionID)

	result, err := c.client.InvokeAgent(ctx, &bedrockagentruntime.InvokeAgentInput{
		AgentId:        aws.String(input.AgentID),
		AgentAliasId:   aws.String(input.AgentAliasID),
		SessionId:      aws.String(input.SessionID),
		InputText:      aws.String(enhancedInput),
		EnableTrace:    aws.Bool(true),
		SessionState: &types.SessionState{
			PromptSessionAttributes: sessionAttrs,
		},
	})
	if err != nil {
		// Check for context deadline exceeded
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("Bedrock agent invocation timed out after 25 seconds")
			return nil, fmt.Errorf("agent invocation timed out")
		}
		log.Printf("Error invoking Bedrock agent: %v", err)
		return nil, err
	}

	// Process the streaming response
	var outputText strings.Builder
	stream := result.GetStream()

	for event := range stream.Events() {
		switch v := event.(type) {
		case *types.ResponseStreamMemberChunk:
			chunk := string(v.Value.Bytes)
			outputText.WriteString(chunk)
			log.Printf("Received chunk: %s", truncateString(chunk, 100))
		case *types.ResponseStreamMemberTrace:
			// Log trace events for debugging
			if v.Value.Trace != nil {
				log.Printf("Trace event received")
			}
		default:
			log.Printf("Unknown event type: %T", v)
		}
	}

	if err := stream.Err(); err != nil {
		log.Printf("Stream error: %v", err)
		return nil, err
	}

	responseText := cleanResponse(outputText.String())
	log.Printf("Agent response: %s", truncateString(responseText, 200))

	return &models.AgentResponse{
		OutputText: responseText,
		SessionID:  input.SessionID,
		Metadata:   sessionAttrs,
	}, nil
}

// buildSystemContext creates an enhanced prompt with persona and troubleshooting context
func buildSystemContext(userInput string, persona *models.Persona) string {
	if persona == nil {
		return userInput
	}

	// For simple inputs, just return the user input
	// The agent already has the persona context from session attributes
	return userInput
}

// cleanResponse removes any JSON artifacts or function call remnants from the response
func cleanResponse(response string) string {
	// Remove common JSON artifacts
	response = strings.TrimSpace(response)

	// Remove function call patterns like {"function": "...", "parameters": {...}}
	jsonPattern := regexp.MustCompile(`\{[^{}]*"function"[^{}]*\}`)
	response = jsonPattern.ReplaceAllString(response, "")

	// Remove any remaining JSON-like artifacts
	response = strings.TrimPrefix(response, "{")
	response = strings.TrimSuffix(response, "}")

	return strings.TrimSpace(response)
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// GetAgentStatus checks if an agent is available
func (c *BedrockClient) GetAgentStatus(ctx context.Context, agentID, aliasID string) (bool, error) {
	// Try a simple invocation with empty input to check availability
	_, err := c.client.InvokeAgent(ctx, &bedrockagentruntime.InvokeAgentInput{
		AgentId:      aws.String(agentID),
		AgentAliasId: aws.String(aliasID),
		SessionId:    aws.String("health-check"),
		InputText:    aws.String("health check"),
	})

	if err != nil {
		log.Printf("Agent health check failed: %v", err)
		return false, err
	}

	return true, nil
}
