package agents

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
	"github.com/headset-support-agent/internal/logging"
	"github.com/headset-support-agent/internal/models"
)

// agentRuntimeAPI is the subset of bedrockagentruntime.Client used by BedrockClient.
// It exists so the real SDK client can be swapped for a test mock.
type agentRuntimeAPI interface {
	InvokeAgent(ctx context.Context, params *bedrockagentruntime.InvokeAgentInput, optFns ...func(*bedrockagentruntime.Options)) (*bedrockagentruntime.InvokeAgentOutput, error)
	RetrieveAndGenerate(ctx context.Context, params *bedrockagentruntime.RetrieveAndGenerateInput, optFns ...func(*bedrockagentruntime.Options)) (*bedrockagentruntime.RetrieveAndGenerateOutput, error)
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

// ---------------------------------------------------------------------------
// A-08: Lambda-side knowledge-base retrieval (RetrieveAndGenerate)
// ---------------------------------------------------------------------------

// kbNumberOfResults is how many source chunks the vector search returns for
// generation. The KB docs are retrieval-sized splits (A-01), so a handful of
// chunks comfortably covers a single question without flooding the prompt.
const kbNumberOfResults = 6

// RetrieveAndGenerateRequest contains parameters for a KB-grounded answer.
type RetrieveAndGenerateRequest struct {
	// KnowledgeBaseID is the Bedrock Knowledge Base to query (required).
	KnowledgeBaseID string
	// ModelID is the foundation model ID, inference-profile ID, or full ARN
	// used for generation (required) — e.g. BEDROCK_MODEL_SUPERVISOR.
	ModelID string
	// Query is the user's question (required).
	Query string
	// Persona styles the generated answer (optional).
	Persona *models.Persona
	// Filters are optional metadata equality filters applied to the vector
	// search (e.g. tree_id, brand, connection_type — the keys written by the
	// KB ingestion sidecars). Multiple entries are ANDed. Nil/empty = none.
	Filters map[string]string
	// FilterAnyOf are optional metadata "in" filters: the document's value for
	// the key must be one of the listed values. B-07 uses this for
	// connection_type / brand with the caller's value PLUS "any", because most
	// KB docs are tagged brand:"any" / connection_type:"any" — an exact-equals
	// filter on a specific brand would exclude all generic docs and starve
	// retrieval. Each entry is ANDed with the others and with Filters.
	FilterAnyOf map[string][]string
	// GuardrailID is the Bedrock Guardrail identifier (A-09). When BOTH
	// GuardrailID and GuardrailVersion are non-empty, the guardrail is attached
	// to the GenerationConfiguration so it applies on the RetrieveAndGenerate
	// path (the primary/effective answer path). Leave empty to skip (graceful).
	GuardrailID string
	// GuardrailVersion is the published version string of the guardrail (A-09).
	// Must be non-empty when GuardrailID is set; ignored when GuardrailID is empty.
	GuardrailVersion string
}

// KBAnswer is a generated, KB-grounded answer plus best-effort citations.
type KBAnswer struct {
	// Text is the generated answer.
	Text string
	// Citations lists the distinct source URIs the answer was grounded in
	// (best effort — empty when the service returns no references).
	Citations []string
}

// RetrieveAndGenerate queries the knowledge base and generates a grounded
// answer with the given model. The generation prompt template (see
// groundedPromptTemplate) enforces that the model answers ONLY from the
// retrieved results, admits when the KB lacks the answer (offering
// escalation), and speaks concisely in the caller's persona tone.
func (c *BedrockClient) RetrieveAndGenerate(ctx context.Context, req RetrieveAndGenerateRequest) (*KBAnswer, error) {
	if req.KnowledgeBaseID == "" {
		return nil, fmt.Errorf("knowledge base id is required")
	}
	if req.ModelID == "" {
		return nil, fmt.Errorf("model id is required")
	}
	if strings.TrimSpace(req.Query) == "" {
		return nil, fmt.Errorf("query is required")
	}

	// Same Lambda-budget timeout policy as InvokeAgent.
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	slog.Info("retrieve-and-generate against knowledge base",
		slog.String("kb_id", req.KnowledgeBaseID),
		slog.String("model", req.ModelID),
		slog.Int("filter_count", len(req.Filters)+len(req.FilterAnyOf)),
		slog.Bool("guardrail", req.GuardrailID != ""),
	)
	slog.Debug("retrieve-and-generate query",
		slog.String("query_preview", logging.Truncate(req.Query, 80)),
	)

	// A-09: attach the guardrail to GenerationConfiguration when both fields
	// are provided. Only the RetrieveAndGenerate path is the live answer path,
	// so this is where the guardrail must be wired to actually take effect.
	genCfg := &types.GenerationConfiguration{
		PromptTemplate: &types.PromptTemplate{
			TextPromptTemplate: aws.String(groundedPromptTemplate(req.Persona)),
		},
	}
	if req.GuardrailID != "" && req.GuardrailVersion != "" {
		genCfg.GuardrailConfiguration = &types.GuardrailConfiguration{
			GuardrailId:      aws.String(req.GuardrailID),
			GuardrailVersion: aws.String(req.GuardrailVersion),
		}
	}

	out, err := c.client.RetrieveAndGenerate(ctx, &bedrockagentruntime.RetrieveAndGenerateInput{
		Input: &types.RetrieveAndGenerateInput{Text: aws.String(req.Query)},
		RetrieveAndGenerateConfiguration: &types.RetrieveAndGenerateConfiguration{
			Type: types.RetrieveAndGenerateTypeKnowledgeBase,
			KnowledgeBaseConfiguration: &types.KnowledgeBaseRetrieveAndGenerateConfiguration{
				KnowledgeBaseId:         aws.String(req.KnowledgeBaseID),
				ModelArn:                aws.String(req.ModelID),
				GenerationConfiguration: genCfg,
				RetrievalConfiguration: &types.KnowledgeBaseRetrievalConfiguration{
					VectorSearchConfiguration: &types.KnowledgeBaseVectorSearchConfiguration{
						NumberOfResults: aws.Int32(kbNumberOfResults),
						Filter:          buildRetrievalFilter(req.Filters, req.FilterAnyOf),
					},
				},
			},
		},
	})
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			slog.Error("retrieve-and-generate timed out", slog.String("kb_id", req.KnowledgeBaseID))
			return nil, fmt.Errorf("knowledge base retrieval timed out")
		}
		slog.Error("error calling retrieve-and-generate",
			slog.String("kb_id", req.KnowledgeBaseID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	if out == nil || out.Output == nil || out.Output.Text == nil || strings.TrimSpace(*out.Output.Text) == "" {
		slog.Warn("empty retrieve-and-generate response", slog.String("kb_id", req.KnowledgeBaseID))
		return nil, fmt.Errorf("empty response from knowledge base retrieval")
	}

	answer := &KBAnswer{
		Text:      strings.TrimSpace(*out.Output.Text),
		Citations: citationURIs(out.Citations),
	}
	slog.Debug("retrieve-and-generate answer",
		slog.String("answer_preview", logging.Truncate(answer.Text, 200)),
		slog.Int("citation_count", len(answer.Citations)),
	)
	return answer, nil
}

// groundedPromptTemplate builds the generation prompt template for
// RetrieveAndGenerate. The $search_results$ placeholder is required by the
// service and is replaced with the retrieved KB chunks; the user's query is
// supplied separately. The rules pin the model to the retrieved results so it
// never invents troubleshooting steps.
func groundedPromptTemplate(p *models.Persona) string {
	name := "a friendly headset support assistant"
	style := "warm, patient, and conversational"
	if p != nil {
		if p.DisplayName != "" {
			name = p.DisplayName + ", a friendly headset support assistant"
		}
		if p.Personality.SpeechStyle != "" {
			style = p.Personality.SpeechStyle
		}
	}
	return "You are " + name + " helping a caller troubleshoot their headset. " +
		"Your speaking style is " + style + ".\n\n" +
		"Here are the headset support knowledge-base search results:\n" +
		"$search_results$\n\n" +
		"Follow these rules strictly when answering the caller's question:\n" +
		"- Answer ONLY from the search results above. Never invent steps, settings, menu paths, or product details that are not in the results.\n" +
		"- If the search results do not contain the answer, say plainly that you don't have that information in your guides and offer to connect the caller to a human specialist. Do not guess.\n" +
		"- Be concise: two to three short sentences, phrased so they can be read aloud naturally.\n" +
		"- Respond in plain conversational text only — no markdown, bullet lists, headings, or citation markers.\n" +
		"- Never ask for or accept payment or card details."
}

// buildRetrievalFilter converts exact (equals) and any-of (in) metadata
// constraints into a single Bedrock retrieval filter: nil when there are none,
// the bare condition when there is exactly one, and an andAll otherwise. Keys
// are sorted within each group (equals first, then in) so the constructed
// filter is deterministic.
func buildRetrievalFilter(exact map[string]string, anyOf map[string][]string) types.RetrievalFilter {
	conds := make([]types.RetrievalFilter, 0, len(exact)+len(anyOf))

	exactKeys := make([]string, 0, len(exact))
	for k := range exact {
		exactKeys = append(exactKeys, k)
	}
	sort.Strings(exactKeys)
	for _, k := range exactKeys {
		conds = append(conds, &types.RetrievalFilterMemberEquals{
			Value: types.FilterAttribute{
				Key:   aws.String(k),
				Value: document.NewLazyDocument(exact[k]),
			},
		})
	}

	inKeys := make([]string, 0, len(anyOf))
	for k, vs := range anyOf {
		if len(vs) > 0 {
			inKeys = append(inKeys, k)
		}
	}
	sort.Strings(inKeys)
	for _, k := range inKeys {
		conds = append(conds, &types.RetrievalFilterMemberIn{
			Value: types.FilterAttribute{
				Key:   aws.String(k),
				Value: document.NewLazyDocument(anyOf[k]),
			},
		})
	}

	switch len(conds) {
	case 0:
		return nil
	case 1:
		return conds[0]
	default:
		return &types.RetrievalFilterMemberAndAll{Value: conds}
	}
}

// citationURIs extracts the distinct S3 source URIs from the citations of a
// RetrieveAndGenerate response (best effort — non-S3 locations are skipped).
func citationURIs(citations []types.Citation) []string {
	var uris []string
	seen := make(map[string]bool)
	for _, c := range citations {
		for _, ref := range c.RetrievedReferences {
			if ref.Location == nil || ref.Location.S3Location == nil || ref.Location.S3Location.Uri == nil {
				continue
			}
			u := *ref.Location.S3Location.Uri
			if u != "" && !seen[u] {
				seen[u] = true
				uris = append(uris, u)
			}
		}
	}
	return uris
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
