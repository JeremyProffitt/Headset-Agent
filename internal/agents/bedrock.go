package agents

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
	"github.com/headset-support-agent/internal/logging"
	"github.com/headset-support-agent/internal/models"
)

// agentRuntimeAPI is the subset of bedrockagentruntime.Client used by BedrockClient.
// It exists so the real SDK client can be swapped for a test mock.
type agentRuntimeAPI interface {
	InvokeAgent(ctx context.Context, params *bedrockagentruntime.InvokeAgentInput, optFns ...func(*bedrockagentruntime.Options)) (*bedrockagentruntime.InvokeAgentOutput, error)
}

// BedrockClient wraps the Bedrock Agent Runtime client
type BedrockClient struct {
	client agentRuntimeAPI
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

	slog.Info("invoking bedrock agent",
		slog.String("agent_id", input.AgentID),
		slog.String("alias_id", input.AgentAliasID),
		slog.String("session_id", input.SessionID),
	)

	result, err := c.client.InvokeAgent(ctx, &bedrockagentruntime.InvokeAgentInput{
		AgentId:      aws.String(input.AgentID),
		AgentAliasId: aws.String(input.AgentAliasID),
		SessionId:    aws.String(input.SessionID),
		InputText:    aws.String(enhancedInput),
		EnableTrace:  aws.Bool(true),
		SessionState: &types.SessionState{
			PromptSessionAttributes: sessionAttrs,
		},
	})
	if err != nil {
		// Check for context deadline exceeded
		if ctx.Err() == context.DeadlineExceeded {
			slog.Error("bedrock agent invocation timed out", slog.String("session_id", input.SessionID))
			return nil, fmt.Errorf("agent invocation timed out")
		}
		slog.Error("error invoking bedrock agent",
			slog.String("session_id", input.SessionID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	// Process the streaming response
	responseText, err := processResponseStream(result.GetStream().Reader)
	if err != nil {
		return nil, err
	}

	slog.Debug("agent response",
		slog.String("session_id", input.SessionID),
		slog.String("response_preview", logging.Truncate(responseText, 200)),
	)

	return &models.AgentResponse{
		OutputText: responseText,
		SessionID:  input.SessionID,
		Metadata:   sessionAttrs,
	}, nil
}

// processResponseStream reads all events from a ResponseStreamReader, accumulates
// the text chunks, and returns the cleaned response string.
// Extracted as a standalone function so it can be unit-tested independently of
// the live SDK event-stream machinery.
func processResponseStream(reader bedrockagentruntime.ResponseStreamReader) (string, error) {
	var outputText strings.Builder

	for event := range reader.Events() {
		switch v := event.(type) {
		case *types.ResponseStreamMemberChunk:
			chunk := string(v.Value.Bytes)
			outputText.WriteString(chunk)
			slog.Debug("received response chunk",
				slog.String("chunk_preview", logging.Truncate(chunk, 100)),
			)
		case *types.ResponseStreamMemberTrace:
			// Log trace events for debugging
			if v.Value.Trace != nil {
				slog.Debug("trace event received")
			}
		default:
			slog.Warn("unknown stream event type", slog.String("type", fmt.Sprintf("%T", v)))
		}
	}

	if err := reader.Err(); err != nil {
		slog.Error("response stream error", slog.String("error", err.Error()))
		return "", err
	}

	return cleanResponse(outputText.String()), nil
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
		slog.Error("agent health check failed",
			slog.String("agent_id", agentID),
			slog.String("alias_id", aliasID),
			slog.String("error", err.Error()),
		)
		return false, err
	}

	return true, nil
}
