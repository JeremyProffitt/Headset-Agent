# Voice-Based Headset Troubleshooting Agent POC
## Architecture & Cost Analysis

**Version:** 1.0  
**Date:** December 31, 2025  
**Stack:** 100% AWS (Lambda, API Gateway, Go)

---

## Executive Summary

This document outlines the architecture for a proof-of-concept (POC) voice-based AI agent that guides users through headset audio troubleshooting via phone call. The system leverages Amazon Connect for telephony, Amazon Lex for conversational AI orchestration, Amazon Bedrock for intelligent reasoning, and AWS Lambda (Go) for business logic.

---

## Architecture Overview

### High-Level Flow

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Phone     │────▶│   Amazon    │────▶│   Amazon    │────▶│   Lambda    │
│   Call      │     │   Connect   │     │     Lex     │     │   (Go)      │
└─────────────┘     └─────────────┘     └─────────────┘     └──────┬──────┘
                                                                    │
                    ┌─────────────┐     ┌─────────────┐            │
                    │  Knowledge  │◀────│   Amazon    │◀───────────┘
                    │    Base     │     │   Bedrock   │
                    │  (S3/OSS)   │     │   Agent     │
                    └─────────────┘     └─────────────┘
```

### Component Breakdown

| Component | Purpose | AWS Service |
|-----------|---------|-------------|
| Telephony | Inbound call handling, DID numbers | Amazon Connect |
| Speech-to-Text | Real-time voice transcription | Amazon Connect (built-in) |
| Conversation Mgmt | Intent recognition, dialog flow | Amazon Lex V2 |
| AI Reasoning | Troubleshooting logic, RAG | Amazon Bedrock (Claude Haiku 3.5) |
| Knowledge Base | Headset troubleshooting docs | Bedrock Knowledge Bases + S3 |
| Business Logic | Orchestration, escape hatch | AWS Lambda (Go) |
| Text-to-Speech | Voice response generation | Amazon Polly (via Connect) |
| Vector Storage | Knowledge embeddings | OpenSearch Serverless |
| API Layer | Webhook endpoints, admin | API Gateway |

---

## Detailed Architecture

### 1. Amazon Connect (Telephony Layer)

Amazon Connect serves as the entry point for all phone calls.

**Configuration:**
- Provision a DID (Direct Inward Dial) phone number
- Create a Contact Flow that:
  1. Plays a welcome greeting
  2. Routes to Amazon Lex bot
  3. Handles escalation to human agents
  4. Loops for multi-turn conversation

**Contact Flow Design:**
```
Start ──▶ Set Logging ──▶ Set Voice (Neural) ──▶ Get Customer Input (Lex)
                                                         │
                              ┌─────────────────────────┴───────────────┐
                              ▼                                         ▼
                        Agent Request?                           Continue Lex
                              │                                         │
                              ▼                                         │
                     Transfer to Queue                                  │
                              │                                         │
                              └─────────────────────────────────────────┘
                                              ▼
                                        End/Disconnect
```

### 2. Amazon Lex V2 (Conversational AI)

Lex handles intent recognition and dialog management.

**Intents:**
| Intent | Description | Slots |
|--------|-------------|-------|
| `StartTroubleshooting` | Begin troubleshooting session | `HeadsetBrand`, `IssueType` |
| `DescribeProblem` | User describes audio issue | `ProblemDescription` |
| `ConfirmStep` | User confirms/denies step completion | `Confirmation` |
| `RequestAgent` | User wants human agent (escape hatch) | None |
| `EndSession` | User wants to end call | None |
| `FallbackIntent` | Catch-all for unrecognized input | None |

**Dialog Flow:**
```go
// Lex triggers Lambda for fulfillment
// Lambda calls Bedrock for intelligent responses
```

### 3. AWS Lambda (Go) - Orchestration Layer

The Lambda function serves as the bridge between Lex and Bedrock.

**Function Structure:**
```go
package main

import (
    "context"
    "encoding/json"
    
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
)

type LexEvent struct {
    SessionState    SessionState    `json:"sessionState"`
    InterpretedIntent InterpretedIntent `json:"interpretations"`
    InputTranscript string          `json:"inputTranscript"`
    SessionId       string          `json:"sessionId"`
}

type SessionState struct {
    Intent       Intent                 `json:"intent"`
    SessionAttrs map[string]string      `json:"sessionAttributes"`
}

func handler(ctx context.Context, event LexEvent) (LexResponse, error) {
    // 1. Check for escape hatch (agent request)
    if isAgentRequest(event) {
        return escalateToAgent(event)
    }
    
    // 2. Call Bedrock Agent with context
    response, err := callBedrockAgent(ctx, event)
    if err != nil {
        return fallbackResponse(err)
    }
    
    // 3. Format response for Lex
    return formatLexResponse(response, event)
}

func main() {
    lambda.Start(handler)
}
```

**Key Functions:**

```go
// Escape Hatch Detection
func isAgentRequest(event LexEvent) bool {
    keywords := []string{"agent", "human", "representative", "speak to someone", "transfer"}
    transcript := strings.ToLower(event.InputTranscript)
    
    for _, keyword := range keywords {
        if strings.Contains(transcript, keyword) {
            return true
        }
    }
    return event.SessionState.Intent.Name == "RequestAgent"
}

// Bedrock Agent Invocation
func callBedrockAgent(ctx context.Context, event LexEvent) (*AgentResponse, error) {
    cfg, _ := config.LoadDefaultConfig(ctx)
    client := bedrockagentruntime.NewFromConfig(cfg)
    
    // Build conversation history from session
    history := buildConversationHistory(event.SessionState.SessionAttrs)
    
    // Invoke agent with RAG
    input := &bedrockagentruntime.InvokeAgentInput{
        AgentId:      aws.String(os.Getenv("BEDROCK_AGENT_ID")),
        AgentAliasId: aws.String(os.Getenv("BEDROCK_AGENT_ALIAS")),
        SessionId:    aws.String(event.SessionId),
        InputText:    aws.String(event.InputTranscript),
    }
    
    return client.InvokeAgent(ctx, input)
}
```

### 4. Amazon Bedrock Agent (AI Reasoning)

The Bedrock Agent provides intelligent troubleshooting guidance.

**Agent Configuration:**
- **Model:** Claude 3.5 Haiku (cost-effective, fast responses)
- **Knowledge Base:** Headset troubleshooting documentation
- **Guardrails:** Content filtering, stay on topic

**Agent Instructions (System Prompt):**
```
You are a friendly and patient technical support agent specializing in headset 
audio troubleshooting. Your role is to help users diagnose and resolve audio 
issues with their headsets.

INTERACTION RULES:
1. Always confirm understanding before proceeding to next step
2. Use simple, non-technical language unless user indicates technical expertise
3. Guide through ONE troubleshooting step at a time
4. Wait for user confirmation before moving forward
5. If a step doesn't work, try the next logical troubleshooting approach
6. After 3-4 failed attempts on an issue, suggest escalation to human agent

TROUBLESHOOTING HIERARCHY:
1. Physical connections (cables, USB, audio jack)
2. Volume and mute settings (hardware and software)
3. Default audio device configuration
4. Driver and firmware issues
5. Application-specific audio settings
6. Hardware diagnostics

ESCAPE TRIGGERS:
- If user says "agent", "human", "representative", or similar → Immediately 
  acknowledge and prepare for transfer
- If user expresses frustration more than twice → Proactively offer human agent

RESPONSE FORMAT:
- Keep responses conversational and suitable for voice
- Avoid lists - speak in natural sentences
- Limit responses to 2-3 sentences maximum for voice clarity
- Always end with a clear question or instruction
```

### 5. Amazon Bedrock Knowledge Base

The Knowledge Base provides RAG (Retrieval-Augmented Generation) for accurate troubleshooting.

**Data Source Structure:**
```
s3://headset-kb-bucket/
├── common-issues/
│   ├── no-audio.md
│   ├── one-side-audio.md
│   ├── microphone-not-working.md
│   ├── static-crackling.md
│   └── bluetooth-connection.md
├── brand-specific/
│   ├── jabra/
│   ├── plantronics-poly/
│   ├── logitech/
│   └── microsoft/
├── platform-guides/
│   ├── windows-11.md
│   ├── windows-10.md
│   ├── macos.md
│   └── linux.md
└── application-specific/
    ├── microsoft-teams.md
    ├── zoom.md
    └── webex.md
```

**Chunking Strategy:**
- Chunk size: 300 tokens
- Overlap: 50 tokens
- Embedding model: Amazon Titan Embeddings G1

**Vector Store Options:**

| Option | Monthly Cost | Recommended For |
|--------|--------------|-----------------|
| OpenSearch Serverless (2 OCU) | ~$360/month | Production |
| Pinecone Starter | Free (up to 100K vectors) | POC |
| Aurora PostgreSQL pgvector | Variable | Existing Aurora users |

---

## Escape Hatch Implementation

### Trigger Keywords
The system monitors for these phrases to initiate human agent transfer:

```go
var escapeKeywords = []string{
    "agent",
    "human",
    "representative",
    "speak to someone",
    "real person",
    "transfer me",
    "manager",
    "supervisor",
}
```

### Escalation Flow
```
User says "agent" ──▶ Lambda detects keyword ──▶ Returns transfer action
                                                        │
                                                        ▼
                              Connect receives DialogAction: TransferToQueue
                                                        │
                                                        ▼
                              Call transferred to human agent queue
```

### Lambda Escalation Response:
```go
func escalateToAgent(event LexEvent) (LexResponse, error) {
    return LexResponse{
        SessionState: SessionState{
            DialogAction: DialogAction{
                Type: "Close",
            },
            Intent: Intent{
                Name:  event.SessionState.Intent.Name,
                State: "Fulfilled",
            },
            SessionAttributes: map[string]string{
                "escalationReason": "User requested human agent",
                "conversationSummary": buildSummary(event),
            },
        },
        Messages: []Message{
            {
                ContentType: "PlainText",
                Content: "I understand you'd like to speak with a human agent. " +
                    "Let me transfer you now. Please hold for a moment.",
            },
        },
    }, nil
}
```

---

## API Gateway Integration

API Gateway provides REST endpoints for:

1. **Admin Dashboard** - Knowledge base management
2. **Webhooks** - External system notifications
3. **Analytics** - Call metrics and outcomes

**Endpoints:**
```
POST /api/v1/kb/sync          # Trigger knowledge base sync
GET  /api/v1/analytics/calls  # Get call statistics
POST /api/v1/webhooks/outcome # Record call outcomes
```

---

## Cost Analysis

### Static Monthly Costs (Infrastructure)

| Service | Description | Monthly Cost |
|---------|-------------|--------------|
| Amazon Connect | Phone number (DID, US) | ~$0.90 |
| OpenSearch Serverless | 2 OCU minimum (no HA) | ~$360.00 |
| S3 | Knowledge base storage (~1GB) | ~$0.02 |
| CloudWatch | Logs and monitoring | ~$10.00 |
| Lambda | Baseline (minimal idle) | ~$0.00 |
| **TOTAL** | | **~$371/month** |

**POC-Optimized Option (using Pinecone Free Tier):**

| Service | Description | Monthly Cost |
|---------|-------------|--------------|
| Amazon Connect | Phone number (DID, US) | ~$0.90 |
| Pinecone | Free tier (100K vectors) | $0.00 |
| S3 | Knowledge base storage | ~$0.02 |
| CloudWatch | Logs and monitoring | ~$5.00 |
| **TOTAL** | | **~$6/month** |

### Usage Costs per 1,000 Minutes of Calls

**Assumptions:**
- Average call duration: 5 minutes
- Calls per 1,000 minutes: 200 calls
- Average Lex turns per call: 10 (voice requests)
- Average Bedrock tokens per call: 2,000 input + 500 output
- Lambda invocations per call: 12

| Service | Calculation | Cost |
|---------|-------------|------|
| **Amazon Connect** | 1,000 min × $0.038/min | $38.00 |
| **Connect Telephony (DID inbound)** | 1,000 min × $0.0022/min | $2.20 |
| **Amazon Lex (voice)** | 2,000 requests × $0.004/req | $8.00 |
| **Amazon Bedrock (Claude 3.5 Haiku)** | | |
| - Input tokens | 400K tokens × ($0.00025/1K) | $0.10 |
| - Output tokens | 100K tokens × ($0.00125/1K) | $0.13 |
| **Bedrock KB Embedding queries** | 2,000 queries × $0.00002/query | $0.04 |
| **Lambda** | 2,400 invocations (128MB, 500ms) | ~$0.01 |
| **Data Transfer** | Minimal internal traffic | ~$0.00 |
| **TOTAL per 1,000 minutes** | | **~$48.48** |

### Cost Summary Table

| Volume | Static Cost | Usage Cost | Total Monthly |
|--------|-------------|------------|---------------|
| POC (1,000 min) | $6 (Pinecone) | $48 | **$54** |
| Low (5,000 min) | $371 | $242 | **$613** |
| Medium (25,000 min) | $371 | $1,212 | **$1,583** |
| High (100,000 min) | $371 | $4,848 | **$5,219** |

### Cost Optimization Strategies

1. **Use Claude 3.5 Haiku** - 5-10x cheaper than Sonnet/Opus
2. **Implement Prompt Caching** - 90% discount on cached tokens
3. **Intelligent Routing** - Route simple queries to smaller models
4. **Batch Processing** - Use batch API for analytics (50% discount)
5. **Pinecone for POC** - Free tier eliminates OpenSearch costs

---

## Implementation Roadmap

### Phase 1: Foundation (Week 1-2)
- [ ] Set up Amazon Connect instance
- [ ] Provision DID phone number
- [ ] Create basic Contact Flow
- [ ] Deploy Lambda function skeleton (Go)

### Phase 2: Conversational AI (Week 3-4)
- [ ] Design and build Amazon Lex bot
- [ ] Implement all intents and slots
- [ ] Integrate Lambda with Lex
- [ ] Test basic conversation flow

### Phase 3: AI Agent (Week 5-6)
- [ ] Create Bedrock Knowledge Base
- [ ] Upload troubleshooting documentation
- [ ] Configure Bedrock Agent
- [ ] Connect Lambda to Bedrock Agent

### Phase 4: Polish & Test (Week 7-8)
- [ ] Implement escape hatch logic
- [ ] Add conversation context management
- [ ] Conduct end-to-end testing
- [ ] Performance optimization

---

## Go Project Structure

```
headset-troubleshooter/
├── cmd/
│   └── lambda/
│       └── main.go              # Lambda entry point
├── internal/
│   ├── agent/
│   │   ├── bedrock.go           # Bedrock agent client
│   │   └── prompts.go           # System prompts
│   ├── handlers/
│   │   ├── lex.go               # Lex event handler
│   │   └── escalation.go        # Agent transfer logic
│   ├── models/
│   │   ├── lex.go               # Lex request/response types
│   │   └── session.go           # Session management
│   └── kb/
│       └── knowledge.go         # Knowledge base queries
├── go.mod
├── go.sum
├── Makefile
├── template.yaml                # SAM template
└── README.md
```

### SAM Template (template.yaml)

```yaml
AWSTemplateFormatVersion: '2010-09-09'
Transform: AWS::Serverless-2016-10-31

Globals:
  Function:
    Timeout: 30
    MemorySize: 128
    Runtime: provided.al2023
    Architectures:
      - arm64

Resources:
  TroubleshootingFunction:
    Type: AWS::Serverless::Function
    Properties:
      Handler: bootstrap
      CodeUri: cmd/lambda/
      Environment:
        Variables:
          BEDROCK_AGENT_ID: !Ref BedrockAgentId
          BEDROCK_AGENT_ALIAS: !Ref BedrockAgentAlias
      Policies:
        - Version: '2012-10-17'
          Statement:
            - Effect: Allow
              Action:
                - bedrock:InvokeAgent
                - bedrock:RetrieveAndGenerate
              Resource: '*'
```

---

## Security Considerations

1. **IAM Least Privilege** - Lambda role with minimal Bedrock permissions
2. **VPC Integration** - Optional private connectivity for Bedrock
3. **Encryption** - Enable encryption at rest for all data stores
4. **PII Handling** - No PII logging, session data auto-expires
5. **Guardrails** - Bedrock Guardrails for content filtering

---

## Monitoring & Observability

**CloudWatch Metrics:**
- Call volume and duration
- Escalation rate
- Intent recognition accuracy
- Bedrock latency and token usage
- Lambda errors and cold starts

**Dashboards:**
- Real-time call activity
- Issue resolution rate
- Customer satisfaction indicators
- Cost tracking

---

## Appendix: Key AWS Documentation

- [Amazon Connect + Lex Integration](https://docs.aws.amazon.com/connect/latest/adminguide/amazon-lex.html)
- [Amazon Bedrock Agents](https://docs.aws.amazon.com/bedrock/latest/userguide/agents.html)
- [Bedrock Knowledge Bases](https://aws.amazon.com/bedrock/knowledge-bases/)
- [AWS SDK for Go v2 - Bedrock](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime)
