package main

import (
	"context"
	"encoding/json"
	"errors"
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
	"github.com/headset-support-agent/internal/models"
	"github.com/headset-support-agent/internal/persona"
	"github.com/headset-support-agent/internal/session"
	"github.com/headset-support-agent/internal/triage"
)

// sessionStorer is the interface the handler uses for session persistence.
// The real *session.Store satisfies it; tests inject a mock implementation.
type sessionStorer interface {
	Load(ctx context.Context, sessionID string) (*models.Session, error)
	Save(ctx context.Context, sess *models.Session) error
}

// agentInvoker is the interface the handler uses to call the Bedrock agent.
// The real *agents.BedrockClient satisfies it; tests inject a stub.
type agentInvoker interface {
	InvokeAgent(ctx context.Context, input agents.InvokeAgentInput) (*models.AgentResponse, error)
}

// personaLoaderI is the interface the handler uses to load a persona.
// The real *persona.Loader satisfies it; tests inject a stub.
type personaLoaderI interface {
	Load(ctx context.Context, personaID string) (*models.Persona, error)
}

var (
	agentClient   agentInvoker   // set in init(); overridable in tests
	personaLoader personaLoaderI // set in init(); overridable in tests
	ssmClient     *ssm.Client
	sessStore     sessionStorer // set in init(); overridable in tests

	// B-05: the deterministic triage engine, built once over the compiled-in
	// trees (no I/O). Stateless and safe for concurrent invocations.
	triageEngine = triage.NewDefaultEngine()
	// B-05: keyword-only symptom classifier (nil LLM fallback — a supported,
	// deterministic configuration per classify.go). B-07 may inject a Bedrock
	// Converse LLM for ambiguous openers.
	triageClassifier = triage.NewClassifier(nil)

	agentConfig struct {
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

	// B-02: construct session store from SESSION_TABLE_NAME env.
	sessionTableName := os.Getenv("SESSION_TABLE_NAME")
	if sessionTableName == "" {
		sessionTableName = "HeadsetSupportSessions"
	}
	sessStore = session.NewStore(cfg, sessionTableName)
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

// loadAndMergeSession loads the stored session and produces a merged session
// object that reflects both the durable store state and the Lex/API per-turn
// attributes.
//
// Merge is a two-pass operation:
//
//	Pass 1 — stored fills gaps in turnAttrs:
//	  Stored attributes are copied into turnAttrs for any key not already
//	  present, so Lex-provided keys (persona_id, etc.) are not overridden by
//	  the store.
//
//	Pass 2 — turnAttrs (now including stored gaps) are applied to sess.Attributes:
//	  Every key in turnAttrs is written into sess.Attributes, with turnAttrs
//	  winning on any collision.  This means:
//	    - Lex-provided keys override any stale stored value.
//	    - Counter keys (frustration_count, failed_steps) end up in
//	      sess.Attributes with the stored value, because they weren't present
//	      in the incoming Lex attrs and were gap-filled in Pass 1.
//
// After this function returns, sess.Attributes is the single source of truth
// for the rest of the turn.  The handler reads counters from sess via the
// named accessors (session.GetFrustrationCount etc.) and writes updates back
// via the same accessors.  saveSession persists sess.Attributes; it only fills
// in additional keys from turnAttrs that were not already set (i.e. escalation
// attrs set by BuildEscalationResponse after the load).
//
// On Load failure the function logs at WARN and returns a fresh empty session
// so the turn continues (graceful degrade).
func loadAndMergeSession(ctx context.Context, sessionID string, turnAttrs map[string]string) *models.Session {
	sess, err := sessStore.Load(ctx, sessionID)
	if err != nil {
		slog.Warn("session load failed — continuing with empty session",
			slog.String("session_id", sessionID),
			slog.String("error", err.Error()),
		)
		sess = &models.Session{
			SessionID:  sessionID,
			Attributes: make(map[string]string),
		}
	}

	// Pass 1: stored keys fill gaps in turnAttrs (Lex-provided keys win).
	for k, v := range sess.Attributes {
		if _, alreadyPresent := turnAttrs[k]; !alreadyPresent {
			turnAttrs[k] = v
		}
	}

	// Pass 2: apply the merged turnAttrs back into sess.Attributes so that
	// the session object is the canonical state for this turn. Lex-provided
	// values win on collision (e.g. persona_id from the front-end overrides
	// a stale stored value).
	for k, v := range turnAttrs {
		sess.Attributes[k] = v
	}

	return sess
}

// saveSession writes the session back to the store after the turn is complete.
// Save errors (including ErrConcurrentUpdate) are logged at WARN only — the
// turn response is returned regardless so we never fail the caller.
//
// Merge on save: sess.Attributes holds values the handler explicitly wrote
// (via session.Set* accessors), such as the updated frustration_count. Those
// WIN over turnAttrs on any collision. turnAttrs only fills keys that the
// handler did not explicitly write to sess.Attributes. This is the inverse of
// the load-merge, where Lex/incoming attrs won; here the handler's persisted
// state wins so that counter updates are never clobbered by stale Lex attrs.
func saveSession(ctx context.Context, sess *models.Session, turnAttrs map[string]string) {
	// Copy turnAttrs into sess.Attributes, but only for keys not already present.
	// Keys the handler set explicitly (e.g. frustration_count via SetFrustrationCount)
	// already live in sess.Attributes and must not be overwritten.
	for k, v := range turnAttrs {
		if _, exists := sess.Attributes[k]; !exists {
			sess.Attributes[k] = v
		}
	}

	if err := sessStore.Save(ctx, sess); err != nil {
		if errors.Is(err, session.ErrConcurrentUpdate) {
			slog.Warn("session save skipped — concurrent update",
				slog.String("session_id", sess.SessionID),
			)
		} else {
			slog.Warn("session save failed",
				slog.String("session_id", sess.SessionID),
				slog.String("error", err.Error()),
			)
		}
	}
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

	// Initialize session attributes if nil.
	if chatReq.SessionState.SessionAttributes == nil {
		chatReq.SessionState.SessionAttributes = make(map[string]string)
	}

	// B-02: load stored session and merge persisted attributes (stored keys fill
	// gaps; incoming request attrs win on collision).
	sess := loadAndMergeSession(ctx, chatReq.SessionID, chatReq.SessionState.SessionAttributes)

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
		// Save session even on refusal path (no counter updates needed here).
		saveSession(ctx, sess, chatReq.SessionState.SessionAttributes)
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

	// Save session before returning.
	saveSession(ctx, sess, chatReq.SessionState.SessionAttributes)

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

	// B-02: load stored session and merge persisted attributes.
	// Stored keys fill gaps; Lex sessionAttributes win on collision.
	// Counter keys (frustration_count, failed_steps) are read from the session
	// object directly (via named accessors) so the Lex-wins rule for those keys
	// in sessionAttributes is irrelevant to accumulation correctness.
	sess := loadAndMergeSession(ctx, event.SessionID, event.SessionState.SessionAttributes)

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
		resp := handlers.BuildSuccessResponse(
			p,
			"Hi there! I'm your headset support assistant. Please describe your issue and I'll help you troubleshoot.",
			event.SessionState.SessionAttributes,
		)
		saveSession(ctx, sess, event.SessionState.SessionAttributes)
		return resp, nil
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
		resp := handlers.BuildPaymentRefusalResponse(p, event.SessionState.SessionAttributes)
		saveSession(ctx, sess, event.SessionState.SessionAttributes)
		return resp, nil
	}

	// B-06: read accumulated counters from the session store (not from Lex
	// sessionAttributes) so they reflect the persisted running totals.
	frustrationCount := session.GetFrustrationCount(sess)
	failedSteps := session.GetFailedSteps(sess)

	// Check for escalation triggers
	escalationDecision := handlers.DetectEscalation(
		transcript,
		frustrationCount,
		failedSteps,
	)

	// B-06: persist the frustration delta so it accumulates across turns.
	// Do this regardless of whether escalation fired (if it fired the session
	// will be saved with escalation attrs; if not the incremented counter
	// ensures the next turn starts from the correct accumulated value).
	if escalationDecision.FrustrationDelta > 0 {
		session.SetFrustrationCount(sess, frustrationCount+escalationDecision.FrustrationDelta)
		// Mirror the fresh value into the Lex turn attributes: the load-merge
		// lets echoed Lex attrs win on collision, so a stale echoed counter
		// would otherwise clobber the store value on the next turn.
		event.SessionState.SessionAttributes[session.KeyFrustrationCount] = sess.Attributes[session.KeyFrustrationCount]
	}

	if escalationDecision.ShouldEscalate {
		// Mark the triage flow ended so a resumed session does not keep
		// navigating the tree after the warm transfer.
		session.SetBool(sess, session.KeyEscalated, true)
		session.SetString(sess, session.KeyEscalationReason, escalationDecision.Reason)
		resp := handlers.BuildEscalationResponse(p, escalationDecision, event.SessionState.SessionAttributes)
		// BuildEscalationResponse mutates sessionAttrs in-place (sets
		// escalation_requested / reason / priority). Those mutations are
		// reflected in event.SessionState.SessionAttributes and will be
		// written into the session by saveSession below.
		saveSession(ctx, sess, event.SessionState.SessionAttributes)
		return resp, nil
	}

	// B-05 (CC-3): drive the deterministic triage engine. The engine owns
	// navigation (which step / transition / terminal); Bedrock only answers
	// free-form side questions, so the flow keeps working when Bedrock is
	// unavailable. Falls through to the generic agent path when the turn is
	// not a triage turn (no symptom classified yet, or the flow ended).
	freeForm := func(ffCtx context.Context, sessionID, question string, kbDoc triage.KBDocRef) (string, error) {
		aID, aAlias := loadAgentConfig(ffCtx)
		if aID == "" || aID == "PLACEHOLDER" || aAlias == "" || aAlias == "PLACEHOLDER" {
			return "", errors.New("supervisor agent not configured")
		}
		prompt := question
		if kbDoc != "" {
			prompt = "Answer the user's question briefly, in at most two spoken sentences, grounded in the knowledge-base document \"" + string(kbDoc) + "\": " + question
		}
		r, err := agentClient.InvokeAgent(ffCtx, agents.InvokeAgentInput{
			AgentID:      aID,
			AgentAliasID: aAlias,
			SessionID:    sessionID,
			InputText:    prompt,
			Persona:      p,
		})
		if err != nil {
			return "", err
		}
		return r.OutputText, nil
	}
	if resp, handled := handlers.HandleTriageTurn(ctx, handlers.TriageDeps{
		Engine:     triageEngine,
		Classifier: triageClassifier,
		FreeForm:   freeForm,
	}, sess, transcript, p, event.SessionState.SessionAttributes); handled {
		saveSession(ctx, sess, event.SessionState.SessionAttributes)
		return resp, nil
	}

	// Load supervisor agent configuration from SSM
	agentID, agentAlias := loadAgentConfig(ctx)

	if agentID == "" || agentID == "PLACEHOLDER" || agentAlias == "" || agentAlias == "PLACEHOLDER" {
		slog.Warn("supervisor agent not configured",
			slog.String("session_id", event.SessionID),
			slog.String("agent_id", agentID),
			slog.String("alias_id", agentAlias),
		)
		resp := handlers.BuildSuccessResponse(
			p,
			"Hello! I'm setting up right now. The system is being configured. Please try again in a few minutes.",
			event.SessionState.SessionAttributes,
		)
		saveSession(ctx, sess, event.SessionState.SessionAttributes)
		return resp, nil
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
		resp := handlers.BuildErrorResponse(p, "I'm having a bit of trouble connecting. Let me try that again.")
		saveSession(ctx, sess, event.SessionState.SessionAttributes)
		return resp, nil
	}

	if response == nil || response.OutputText == "" {
		slog.Warn("empty response from bedrock agent", slog.String("session_id", event.SessionID))
		resp := handlers.BuildErrorResponse(p, "I'm sorry, I didn't catch that. Could you please rephrase your question?")
		saveSession(ctx, sess, event.SessionState.SessionAttributes)
		return resp, nil
	}

	resp := handlers.BuildSuccessResponse(p, response.OutputText, event.SessionState.SessionAttributes)
	saveSession(ctx, sess, event.SessionState.SessionAttributes)
	return resp, nil
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

func main() {
	lambda.Start(handleRequest)
}
