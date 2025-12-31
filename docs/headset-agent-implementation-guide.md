# Building a Voice-Based Headset Troubleshooting Agent
## Multi-Agent Architecture Implementation Guide

**Version:** 2.0  
**Date:** December 31, 2025  
**Stack:** 100% AWS | Lambda | API Gateway | Go  
**Architecture:** Multi-Agent with Supervisor Pattern

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Technology Stack Selection](#technology-stack-selection)
3. [Multi-Agent Design](#multi-agent-design)
4. [Sub-Agent Specifications](#sub-agent-specifications)
5. [Knowledge Base Setup](#knowledge-base-setup)
6. [Interaction Rules](#interaction-rules)
7. [Escape Hatch Implementation](#escape-hatch-implementation)
8. [Implementation Guide](#implementation-guide)
9. [Deployment Options](#deployment-options)
10. [Cost Analysis](#cost-analysis)

---

## Architecture Overview

This POC implements a **Supervisor Agent Pattern** using Amazon Bedrock's Multi-Agent Collaboration (GA as of March 2025). The architecture uses specialized sub-agents coordinated by a supervisor agent to handle different aspects of headset troubleshooting.

### High-Level Architecture

```
                                    ┌─────────────────────┐
                                    │   Phone Call        │
                                    │   (User)            │
                                    └──────────┬──────────┘
                                               │
                                    ┌──────────▼──────────┐
                                    │   Amazon Connect    │
                                    │   (Telephony)       │
                                    └──────────┬──────────┘
                                               │
                                    ┌──────────▼──────────┐
                                    │   Amazon Lex V2     │
                                    │   (Speech/Intent)   │
                                    └──────────┬──────────┘
                                               │
                              ┌────────────────▼────────────────┐
                              │      SUPERVISOR AGENT           │
                              │   (Troubleshooting Orchestrator)│
                              │   - Routes to sub-agents        │
                              │   - Manages conversation flow   │
                              │   - Synthesizes responses       │
                              └───────────────┬─────────────────┘
                                              │
                    ┌─────────────────────────┼─────────────────────────┐
                    │                         │                         │
          ┌─────────▼─────────┐    ┌─────────▼─────────┐    ┌─────────▼─────────┐
          │   DIAGNOSTIC      │    │   PLATFORM        │    │   ESCALATION      │
          │   SUB-AGENT       │    │   SUB-AGENT       │    │   SUB-AGENT       │
          │                   │    │                   │    │                   │
          │ - Hardware checks │    │ - Windows config  │    │ - Agent transfer  │
          │ - Connection tests│    │ - macOS settings  │    │ - Ticket creation │
          │ - Audio testing   │    │ - App-specific    │    │ - Summary gen     │
          └─────────┬─────────┘    └─────────┬─────────┘    └─────────┬─────────┘
                    │                         │                         │
          ┌─────────▼─────────┐    ┌─────────▼─────────┐    ┌─────────▼─────────┐
          │   Knowledge Base  │    │   Knowledge Base  │    │   Action Groups   │
          │   (Hardware KB)   │    │   (Platform KB)   │    │   (CRM/Ticketing) │
          └───────────────────┘    └───────────────────┘    └───────────────────┘
```

---

## Technology Stack Selection

### Core AWS Services

| Component | Service | Justification |
|-----------|---------|---------------|
| **Multi-Agent Orchestration** | Amazon Bedrock Multi-Agent Collaboration | GA March 2025, native supervisor/sub-agent support |
| **Agent Runtime** | Amazon Bedrock AgentCore Runtime | Serverless, framework-agnostic, session isolation |
| **Foundation Model** | Claude 3.5 Haiku (sub-agents) / Claude 3.5 Sonnet (supervisor) | Cost-optimized with quality balance |
| **Agent Framework** | Strands Agents SDK (optional) | AWS open-source, model-driven approach |
| **Telephony** | Amazon Connect | Native Lex/Bedrock integration |
| **Speech Processing** | Amazon Lex V2 | Conversational AI with voice support |
| **Knowledge Storage** | Amazon Bedrock Knowledge Bases | Managed RAG with S3/OpenSearch |
| **Vector Database** | Pinecone (POC) / OpenSearch Serverless (Prod) | Cost vs. scale tradeoff |
| **Business Logic** | AWS Lambda (Go) | Orchestration layer for Lex↔Bedrock |
| **API Layer** | API Gateway | Admin endpoints, webhooks |
| **Protocol Support** | MCP (Model Context Protocol) | Tool standardization via AgentCore Gateway |

### Latest Technology Integrations (2025)

| Technology | Version/Status | Purpose |
|------------|---------------|---------|
| Bedrock Multi-Agent | GA (March 2025) | Supervisor + sub-agent coordination |
| AgentCore Runtime | GA (October 2025) | Production agent hosting |
| Strands Agents SDK | v1.0 (July 2025) | Multi-agent orchestration patterns |
| A2A Protocol | Supported | Agent-to-agent communication |
| MCP Protocol | Supported | Tool interoperability |

---

## Multi-Agent Design

### Why Multi-Agent?

A single monolithic agent for headset troubleshooting faces several challenges:

1. **Context Window Pollution**: Too much knowledge in one agent dilutes accuracy
2. **Specialization**: Different troubleshooting domains require different expertise
3. **Parallel Processing**: Sub-agents can work concurrently
4. **Maintainability**: Easier to update individual sub-agents
5. **Cost Optimization**: Route simple queries to cheaper models

### Collaboration Modes

Amazon Bedrock Multi-Agent Collaboration supports two modes:

#### 1. Supervisor Mode (Recommended for POC)
```
User Query → Supervisor Agent → Analyzes/Breaks Down → Sub-Agents → Synthesize → Response
```

- Supervisor analyzes input and breaks down complex problems
- Invokes sub-agents serially or in parallel
- Consolidates sub-agent responses into final answer

#### 2. Supervisor with Routing Mode (Recommended for Production)
```
Simple Query → Route directly to relevant sub-agent
Complex Query → Full supervisor orchestration
```

- Simple queries bypass full orchestration
- Complex/ambiguous queries trigger full supervisor mode
- Optimizes latency and cost

### Agent Hierarchy

```yaml
supervisor_agent:
  name: "TroubleshootingOrchestrator"
  model: "anthropic.claude-3-5-sonnet-20241022-v2:0"
  role: "Coordinate troubleshooting workflow"
  capabilities:
    - Route queries to appropriate sub-agents
    - Maintain conversation context
    - Synthesize multi-source responses
    - Handle escalation decisions

sub_agents:
  - name: "DiagnosticAgent"
    model: "anthropic.claude-3-5-haiku-20241022-v1:0"
    role: "Hardware and connection diagnostics"
    knowledge_base: "hardware-kb"
    
  - name: "PlatformAgent"
    model: "anthropic.claude-3-5-haiku-20241022-v1:0"
    role: "OS and application configuration"
    knowledge_base: "platform-kb"
    
  - name: "EscalationAgent"
    model: "anthropic.claude-3-5-haiku-20241022-v1:0"
    role: "Human handoff and ticket creation"
    action_groups: ["transfer-to-agent", "create-ticket"]
```

---

## Sub-Agent Specifications

### 1. Diagnostic Sub-Agent

**Purpose**: Handle physical hardware diagnostics and connection testing

**Instructions**:
```
You are a hardware diagnostic specialist for headsets. Your expertise covers:
- USB and audio jack connections
- Bluetooth pairing and connectivity
- Hardware volume controls and mute switches
- Microphone and speaker testing
- Cable and connector inspection

BEHAVIOR:
- Start with the simplest physical checks first
- Ask ONE question at a time
- Confirm each step before proceeding
- Use layman's terms unless user indicates technical expertise
- If you cannot resolve after 3 attempts, recommend escalation

DIAGNOSTIC SEQUENCE:
1. Physical connection verification
2. Hardware control checks (mute/volume)
3. Basic functionality test (can they hear anything?)
4. Detailed component isolation (left/right, mic, speakers)
```

**Knowledge Base**: Hardware troubleshooting documentation
- Brand-specific guides (Jabra, Poly, Logitech, Microsoft)
- Connection type guides (USB-A, USB-C, 3.5mm, Bluetooth)
- LED indicator meanings
- Reset procedures

**Tools/Action Groups**:
```yaml
action_groups:
  - name: "DiagnosticTools"
    actions:
      - name: "CheckWarrantyStatus"
        description: "Verify if headset is under warranty"
        parameters:
          - name: "serial_number"
            type: "string"
      - name: "LogDiagnosticStep"
        description: "Log completed diagnostic step for session"
        parameters:
          - name: "step_name"
            type: "string"
          - name: "result"
            type: "string"
```

---

### 2. Platform Sub-Agent

**Purpose**: Handle OS-level and application-specific audio configuration

**Instructions**:
```
You are a platform configuration specialist for audio devices. Your expertise covers:
- Windows 10/11 audio device management
- macOS audio preferences and permissions
- Linux audio subsystems (PulseAudio/PipeWire)
- Application-specific audio settings (Teams, Zoom, WebEx, etc.)

BEHAVIOR:
- Identify the user's operating system first
- Guide through settings step-by-step with clear navigation paths
- Explain what each setting does in simple terms
- Verify changes took effect before proceeding

CONFIGURATION SEQUENCE:
1. Identify OS and version
2. Check default audio device settings
3. Verify application permissions (microphone access)
4. Configure application-specific audio settings
5. Test with built-in OS tools before application
```

**Knowledge Base**: Platform configuration guides
- Windows Sound Settings navigation
- macOS System Preferences audio
- Teams/Zoom/WebEx audio configuration
- Driver installation and updates
- Permission troubleshooting

**Tools/Action Groups**:
```yaml
action_groups:
  - name: "PlatformTools"
    actions:
      - name: "GenerateSettingsPath"
        description: "Generate step-by-step navigation to a setting"
        parameters:
          - name: "os_type"
            type: "string"
            enum: ["windows10", "windows11", "macos", "linux"]
          - name: "target_setting"
            type: "string"
      - name: "LookupDriverInfo"
        description: "Get driver download information for headset"
        parameters:
          - name: "headset_brand"
            type: "string"
          - name: "headset_model"
            type: "string"
```

---

### 3. Escalation Sub-Agent

**Purpose**: Handle human agent transfers and ticket creation

**Instructions**:
```
You are an escalation specialist responsible for smooth handoffs to human agents.

TRIGGERS FOR ACTIVATION:
- User explicitly requests human agent
- User expresses repeated frustration
- Technical issue requires physical inspection
- Issue persists after exhausting troubleshooting steps

BEHAVIOR:
- Acknowledge the user's need immediately
- Summarize the troubleshooting steps already attempted
- Gather any missing information needed for the handoff
- Prepare a concise summary for the human agent
- Execute the transfer or create the support ticket

ESCALATION PROTOCOL:
1. Confirm user wants to proceed with escalation
2. Generate conversation summary
3. Collect any additional required information
4. Create ticket or initiate transfer
5. Provide user with reference number and expectations
```

**Tools/Action Groups**:
```yaml
action_groups:
  - name: "EscalationTools"
    actions:
      - name: "TransferToAgent"
        description: "Initiate transfer to human agent queue"
        parameters:
          - name: "reason"
            type: "string"
          - name: "summary"
            type: "string"
          - name: "priority"
            type: "string"
            enum: ["low", "medium", "high"]
      - name: "CreateSupportTicket"
        description: "Create a support ticket in the ticketing system"
        parameters:
          - name: "title"
            type: "string"
          - name: "description"
            type: "string"
          - name: "category"
            type: "string"
          - name: "troubleshooting_summary"
            type: "string"
      - name: "GenerateConversationSummary"
        description: "Generate a summary of the troubleshooting session"
        parameters: []
```

---

## Knowledge Base Setup

### Knowledge Base Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    S3 Data Sources                           │
├─────────────────┬─────────────────┬─────────────────────────┤
│   hardware-kb/  │   platform-kb/  │      common-kb/         │
│                 │                 │                         │
│ - jabra/        │ - windows/      │ - faq.md                │
│ - poly/         │ - macos/        │ - glossary.md           │
│ - logitech/     │ - teams/        │ - escalation-policy.md  │
│ - microsoft/    │ - zoom/         │                         │
│ - generic/      │ - webex/        │                         │
└────────┬────────┴────────┬────────┴────────────┬────────────┘
         │                 │                      │
         ▼                 ▼                      ▼
┌─────────────────────────────────────────────────────────────┐
│              Amazon Bedrock Knowledge Bases                  │
│                                                              │
│  Embedding Model: Amazon Titan Embeddings G1 - Text v1.2    │
│  Chunking: Semantic (300 tokens, 50 overlap)                │
│  Vector Store: Pinecone (POC) / OpenSearch (Production)     │
└─────────────────────────────────────────────────────────────┘
```

### Document Structure Best Practices

```markdown
# Document: no-audio-usb-headset.md
---
headset_type: USB
issue_category: no_audio
platforms: [windows, macos]
brands: [all]
difficulty: beginner
---

## Problem Description
User reports no audio output from USB headset.

## Prerequisites
- Headset is plugged in
- User can access sound settings

## Troubleshooting Steps

### Step 1: Verify Physical Connection
Check that the USB cable is fully inserted into the computer's USB port.
**User Prompt**: "Is the USB cable firmly connected to your computer?"
**If No**: Ask user to disconnect and reconnect, ensuring a firm connection.
**If Yes**: Proceed to Step 2.

### Step 2: Check USB Port
Try a different USB port on the computer.
**User Prompt**: "Can you try plugging the headset into a different USB port?"
...
```

### Creating Knowledge Bases via AWS CLI

```bash
# Create S3 bucket for documents
aws s3 mb s3://headset-kb-${AWS_ACCOUNT_ID}

# Upload documents
aws s3 sync ./knowledge-base-docs s3://headset-kb-${AWS_ACCOUNT_ID}/

# Create Knowledge Base
aws bedrock-agent create-knowledge-base \
  --name "HeadsetHardwareKB" \
  --description "Hardware troubleshooting for headsets" \
  --role-arn "arn:aws:iam::${AWS_ACCOUNT_ID}:role/BedrockKBRole" \
  --knowledge-base-configuration '{
    "type": "VECTOR",
    "vectorKnowledgeBaseConfiguration": {
      "embeddingModelArn": "arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v2:0"
    }
  }' \
  --storage-configuration '{
    "type": "PINECONE",
    "pineconeConfiguration": {
      "connectionString": "https://your-index.pinecone.io",
      "credentialsSecretArn": "arn:aws:secretsmanager:us-east-1:${AWS_ACCOUNT_ID}:secret:pinecone-api-key"
    }
  }'
```

---

## Interaction Rules

### Supervisor Agent Instructions

```
You are the Troubleshooting Orchestrator, a supervisor agent coordinating headset 
audio troubleshooting. You have access to three specialized sub-agents:

AVAILABLE SUB-AGENTS:
1. DiagnosticAgent: Hardware diagnostics, physical connections, basic functionality
2. PlatformAgent: OS settings, driver issues, application configuration
3. EscalationAgent: Human agent transfers, ticket creation

ROUTING RULES:
- Physical/hardware issues → DiagnosticAgent
- Settings/configuration issues → PlatformAgent  
- User requests human or frustration detected → EscalationAgent
- Complex issues → Consult DiagnosticAgent first, then PlatformAgent

CONVERSATION RULES:
1. Greet warmly and ask what issue they're experiencing
2. Route to appropriate sub-agent based on issue description
3. If sub-agent needs more info, relay the question naturally
4. Always maintain conversational, supportive tone
5. Never mention "sub-agents" or technical architecture to user
6. Keep responses under 3 sentences for voice clarity
7. Confirm understanding before proceeding to next step

ESCALATION TRIGGERS:
- User says: "agent", "human", "representative", "speak to someone"
- User expresses frustration more than twice
- Same issue persists after 4 troubleshooting steps
- Technical issue requires physical inspection

When escalation is triggered, immediately invoke EscalationAgent.
```

### Response Formatting for Voice

```
VOICE-OPTIMIZED RESPONSE RULES:
1. Maximum 3 sentences per response
2. Avoid technical jargon unless user demonstrates expertise
3. Spell out acronyms on first use
4. Use conversational connectors: "Great!", "Okay,", "I see,"
5. End with clear question or instruction
6. Avoid lists - speak in natural sentences
7. Use pauses: indicate with "..." for dramatic effect

GOOD: "I understand your headset isn't producing any sound. Let's start with a 
quick check. Is the headset currently plugged into your computer?"

BAD: "Audio output issues can stem from multiple sources including: 1) Hardware 
connections, 2) Driver problems, 3) OS configuration. Let's systematically 
troubleshoot each potential cause."
```

---

## Escape Hatch Implementation

### Escape Hatch Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    USER INPUT                                │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│              ESCAPE DETECTION LAYER                          │
│                                                              │
│  Keywords: "agent", "human", "representative", "transfer",  │
│            "speak to someone", "real person", "manager"      │
│                                                              │
│  Signals: frustration_count > 2, failed_steps > 4           │
└──────────────────────────┬──────────────────────────────────┘
                           │
              ┌────────────┴────────────┐
              │                         │
              ▼                         ▼
       ┌──────────────┐         ┌──────────────┐
       │   CONTINUE   │         │   ESCALATE   │
       │   NORMAL     │         │   TO HUMAN   │
       │   FLOW       │         │              │
       └──────────────┘         └──────┬───────┘
                                       │
                                       ▼
                        ┌──────────────────────────────┐
                        │   EscalationAgent Actions:   │
                        │   1. Acknowledge request     │
                        │   2. Generate summary        │
                        │   3. Transfer/Create ticket  │
                        └──────────────────────────────┘
```

### Lambda Escape Hatch Handler (Go)

```go
package handlers

import (
    "strings"
)

// EscapeKeywords that trigger human escalation
var EscapeKeywords = []string{
    "agent",
    "human", 
    "representative",
    "speak to someone",
    "real person",
    "transfer me",
    "manager",
    "supervisor",
    "talk to a person",
}

// FrustrationIndicators suggest user is frustrated
var FrustrationIndicators = []string{
    "this is ridiculous",
    "doesn't work",
    "still not working",
    "frustrated",
    "waste of time",
    "useless",
    "terrible",
}

type EscalationDecision struct {
    ShouldEscalate bool
    Reason         string
    Priority       string
}

func DetectEscalation(
    transcript string,
    frustrationCount int,
    failedSteps int,
) EscalationDecision {
    
    lowerTranscript := strings.ToLower(transcript)
    
    // Check for explicit escape keywords
    for _, keyword := range EscapeKeywords {
        if strings.Contains(lowerTranscript, keyword) {
            return EscalationDecision{
                ShouldEscalate: true,
                Reason:         "user_requested",
                Priority:       "high",
            }
        }
    }
    
    // Check frustration indicators
    for _, indicator := range FrustrationIndicators {
        if strings.Contains(lowerTranscript, indicator) {
            frustrationCount++
        }
    }
    
    // Escalate if frustration threshold exceeded
    if frustrationCount > 2 {
        return EscalationDecision{
            ShouldEscalate: true,
            Reason:         "user_frustrated",
            Priority:       "medium",
        }
    }
    
    // Escalate if too many failed troubleshooting steps
    if failedSteps > 4 {
        return EscalationDecision{
            ShouldEscalate: true,
            Reason:         "troubleshooting_exhausted",
            Priority:       "medium",
        }
    }
    
    return EscalationDecision{
        ShouldEscalate: false,
    }
}
```

### Escalation Response Template

```go
func BuildEscalationResponse(decision EscalationDecision, summary string) LexResponse {
    var message string
    
    switch decision.Reason {
    case "user_requested":
        message = "Absolutely, I'll connect you with a support specialist right away. " +
            "I'm transferring you now. Please hold for just a moment."
    case "user_frustrated":
        message = "I can hear this has been frustrating. Let me get you to a " +
            "specialist who can help resolve this directly. Transferring you now."
    case "troubleshooting_exhausted":
        message = "We've tried several steps without success. I think a specialist " +
            "can better assist you. Let me transfer you to someone who can help."
    }
    
    return LexResponse{
        SessionState: SessionState{
            DialogAction: DialogAction{
                Type: "Close",
            },
            Intent: Intent{
                State: "Fulfilled",
            },
            SessionAttributes: map[string]string{
                "escalation_reason":  decision.Reason,
                "escalation_priority": decision.Priority,
                "conversation_summary": summary,
            },
        },
        Messages: []Message{
            {ContentType: "PlainText", Content: message},
        },
    }
}
```

---

## Implementation Guide

### Phase 1: Foundation (Week 1-2)

#### 1.1 Set Up Amazon Connect

```bash
# Create Connect instance (console recommended for initial setup)
# Then configure via CLI:

# Claim phone number
aws connect claim-phone-number \
  --target-arn "arn:aws:connect:us-east-1:${ACCOUNT_ID}:instance/${INSTANCE_ID}" \
  --phone-number-country-code "US" \
  --phone-number-type "DID"
```

#### 1.2 Create Lex Bot

```bash
# Create bot
aws lexv2-models create-bot \
  --bot-name "HeadsetTroubleshooterBot" \
  --role-arn "arn:aws:iam::${ACCOUNT_ID}:role/LexServiceRole" \
  --data-privacy '{"childDirected": false}' \
  --idle-session-ttl-in-seconds 300
```

#### 1.3 Deploy Lambda Function (Go)

**Project Structure:**
```
headset-agent/
├── cmd/
│   └── lambda/
│       └── main.go
├── internal/
│   ├── agents/
│   │   ├── supervisor.go
│   │   └── bedrock.go
│   ├── handlers/
│   │   ├── lex.go
│   │   └── escalation.go
│   └── models/
│       └── types.go
├── go.mod
├── go.sum
├── Makefile
└── template.yaml
```

**SAM Template (template.yaml):**
```yaml
AWSTemplateFormatVersion: '2010-09-09'
Transform: AWS::Serverless-2016-10-31

Globals:
  Function:
    Timeout: 30
    MemorySize: 256
    Runtime: provided.al2023
    Architectures:
      - arm64
    Environment:
      Variables:
        SUPERVISOR_AGENT_ID: !Ref SupervisorAgentId
        SUPERVISOR_AGENT_ALIAS: !Ref SupervisorAgentAlias

Parameters:
  SupervisorAgentId:
    Type: String
    Description: Bedrock Supervisor Agent ID
  SupervisorAgentAlias:
    Type: String
    Description: Bedrock Supervisor Agent Alias ID

Resources:
  TroubleshootingFunction:
    Type: AWS::Serverless::Function
    Metadata:
      BuildMethod: go1.x
    Properties:
      Handler: bootstrap
      CodeUri: cmd/lambda/
      Policies:
        - Version: '2012-10-17'
          Statement:
            - Effect: Allow
              Action:
                - bedrock:InvokeAgent
                - bedrock:InvokeModel
              Resource: '*'

Outputs:
  FunctionArn:
    Description: Lambda function ARN
    Value: !GetAtt TroubleshootingFunction.Arn
```

### Phase 2: Create Multi-Agent System (Week 3-4)

#### 2.1 Create Sub-Agents

```python
# Python script for agent creation (use with boto3)
import boto3

bedrock_agent = boto3.client('bedrock-agent')

# Create Diagnostic Sub-Agent
diagnostic_agent = bedrock_agent.create_agent(
    agentName='DiagnosticAgent',
    agentResourceRoleArn=f'arn:aws:iam::{account_id}:role/BedrockAgentRole',
    foundationModel='anthropic.claude-3-5-haiku-20241022-v1:0',
    instruction='''You are a hardware diagnostic specialist for headsets...''',
    idleSessionTTLInSeconds=600
)

# Create Platform Sub-Agent
platform_agent = bedrock_agent.create_agent(
    agentName='PlatformAgent',
    agentResourceRoleArn=f'arn:aws:iam::{account_id}:role/BedrockAgentRole',
    foundationModel='anthropic.claude-3-5-haiku-20241022-v1:0',
    instruction='''You are a platform configuration specialist...''',
    idleSessionTTLInSeconds=600
)

# Create Escalation Sub-Agent
escalation_agent = bedrock_agent.create_agent(
    agentName='EscalationAgent',
    agentResourceRoleArn=f'arn:aws:iam::{account_id}:role/BedrockAgentRole',
    foundationModel='anthropic.claude-3-5-haiku-20241022-v1:0',
    instruction='''You are an escalation specialist...''',
    idleSessionTTLInSeconds=600
)
```

#### 2.2 Create Supervisor Agent with Collaboration

```python
# Enable multi-agent collaboration on sub-agents
for agent_id in [diagnostic_agent['agent']['agentId'], 
                  platform_agent['agent']['agentId'],
                  escalation_agent['agent']['agentId']]:
    bedrock_agent.update_agent(
        agentId=agent_id,
        agentCollaboration='SUPERVISOR'  # Enable as collaborator
    )

# Create Supervisor Agent
supervisor_agent = bedrock_agent.create_agent(
    agentName='TroubleshootingOrchestrator',
    agentResourceRoleArn=f'arn:aws:iam::{account_id}:role/BedrockAgentRole',
    foundationModel='anthropic.claude-3-5-sonnet-20241022-v2:0',
    instruction='''You are the Troubleshooting Orchestrator...''',
    idleSessionTTLInSeconds=600
)

# Associate sub-agents with supervisor
bedrock_agent.associate_agent_collaborator(
    agentId=supervisor_agent['agent']['agentId'],
    agentVersion='DRAFT',
    agentDescriptor={
        'aliasArn': f"arn:aws:bedrock:us-east-1:{account_id}:agent-alias/{diagnostic_agent['agent']['agentId']}/TSTALIASID"
    },
    collaboratorName='DiagnosticAgent',
    collaborationInstruction='Use for hardware diagnostics and connection issues',
    relayConversationHistory='TO_COLLABORATOR'
)

# Repeat for other sub-agents...
```

### Phase 3: Knowledge Base Integration (Week 5)

```bash
# Create data source for knowledge base
aws bedrock-agent create-data-source \
  --knowledge-base-id "KB_ID" \
  --name "HardwareDocs" \
  --data-source-configuration '{
    "type": "S3",
    "s3Configuration": {
      "bucketArn": "arn:aws:s3:::headset-kb-bucket",
      "inclusionPrefixes": ["hardware/"]
    }
  }' \
  --vectorIngestionConfiguration '{
    "chunkingConfiguration": {
      "chunkingStrategy": "SEMANTIC",
      "semanticChunkingConfiguration": {
        "maxTokens": 300,
        "bufferSize": 0,
        "breakpointPercentileThreshold": 95
      }
    }
  }'

# Start ingestion
aws bedrock-agent start-ingestion-job \
  --knowledge-base-id "KB_ID" \
  --data-source-id "DS_ID"
```

### Phase 4: Connect Flow Integration (Week 6)

```json
// Contact Flow (exported format)
{
  "Version": "2019-10-30",
  "StartAction": "SetLogging",
  "Actions": [
    {
      "Identifier": "SetLogging",
      "Type": "UpdateContactRecordingBehavior",
      "Parameters": {
        "RecordingBehavior": {"RecordedParticipants": ["Customer"]}
      },
      "Transitions": {"NextAction": "SetVoice"}
    },
    {
      "Identifier": "SetVoice",
      "Type": "UpdateContactTextToSpeechVoice", 
      "Parameters": {
        "TextToSpeechVoice": "Joanna",
        "TextToSpeechEngine": "Neural"
      },
      "Transitions": {"NextAction": "GetInput"}
    },
    {
      "Identifier": "GetInput",
      "Type": "ConnectParticipantWithLexBot",
      "Parameters": {
        "LexBot": {
          "Name": "HeadsetTroubleshooterBot",
          "Alias": "$LATEST"
        }
      },
      "Transitions": {
        "NextAction": "CheckEscalation",
        "Errors": [{"ErrorType": "Error", "NextAction": "Disconnect"}]
      }
    },
    {
      "Identifier": "CheckEscalation",
      "Type": "CheckContactAttributes",
      "Parameters": {
        "Attribute": {"Key": "escalation_requested", "Value": "true"}
      },
      "Transitions": {
        "Conditions": [
          {"Condition": "Equals", "NextAction": "TransferToQueue"}
        ],
        "Default": {"NextAction": "GetInput"}
      }
    },
    {
      "Identifier": "TransferToQueue",
      "Type": "TransferToQueue",
      "Parameters": {
        "QueueId": "SUPPORT_QUEUE_ARN"
      },
      "Transitions": {}
    }
  ]
}
```

---

## Deployment Options

### Option 1: Lambda-Based (Recommended for POC)

Traditional serverless deployment using Lambda for the orchestration layer.

**Pros:**
- Simple, familiar deployment model
- Pay-per-invocation
- Easy integration with Lex

**Cons:**
- Cold starts for voice latency-sensitive applications
- 15-minute execution limit

### Option 2: AgentCore Runtime (Recommended for Production)

Use Amazon Bedrock AgentCore Runtime for production deployment.

```python
# Using Strands Agents SDK with AgentCore
from bedrock_agentcore.runtime import BedrockAgentCoreApp
from strands import Agent, tool

app = BedrockAgentCoreApp()

# Define supervisor agent
supervisor = Agent(
    model="anthropic.claude-3-5-sonnet-20241022-v2:0",
    system_prompt=SUPERVISOR_PROMPT,
    tools=[diagnostic_tool, platform_tool, escalation_tool]
)

@app.entrypoint
async def handle_request(payload, context):
    prompt = payload.get("prompt")
    session_id = context.get("session_id")
    
    result = supervisor.stream_async(prompt)
    async for chunk in result:
        if 'data' in chunk:
            yield chunk['data']

if __name__ == "__main__":
    app.run()
```

**Pros:**
- Purpose-built for agents
- Session isolation
- Built-in memory management
- 8-hour session support
- Native MCP/A2A support

**Cons:**
- Preview pricing (free until Sept 2025)
- Learning curve for new patterns

---

## Cost Analysis

### Static Monthly Costs

| Component | POC (Pinecone) | Production (OpenSearch) |
|-----------|----------------|------------------------|
| Amazon Connect (DID) | $0.90 | $0.90 |
| Vector Database | $0 (free tier) | $360 (2 OCU) |
| S3 Storage | $0.02 | $0.50 |
| CloudWatch | $5.00 | $15.00 |
| **Total Static** | **~$6/month** | **~$376/month** |

### Usage Costs per 1,000 Minutes

| Component | Cost |
|-----------|------|
| **Amazon Connect Voice** | $38.00 |
| **Telephony (DID inbound)** | $2.20 |
| **Amazon Lex (voice @ $0.004/req)** | $8.00 |
| **Bedrock - Supervisor (Sonnet)** | |
| - Input (200K tokens) | $0.60 |
| - Output (50K tokens) | $0.75 |
| **Bedrock - Sub-Agents (Haiku)** | |
| - Input (300K tokens) | $0.075 |
| - Output (75K tokens) | $0.094 |
| **Knowledge Base Queries** | $0.04 |
| **Lambda** | $0.01 |
| **Total per 1,000 min** | **~$49.77** |

### Cost Comparison: Single vs Multi-Agent

| Metric | Single Agent | Multi-Agent |
|--------|--------------|-------------|
| Token Usage | Higher (large context) | Lower (specialized contexts) |
| Model Cost | Sonnet for all | Sonnet (supervisor) + Haiku (sub-agents) |
| Accuracy | Good | Better (specialized) |
| Latency | Single hop | Multiple hops (slightly higher) |
| **Estimated Monthly (10K min)** | ~$520 | ~$498 |

### Cost Optimization Strategies

1. **Use Supervisor with Routing Mode**: Route simple queries directly to sub-agents
2. **Prompt Caching**: Enable Bedrock prompt caching (90% discount on cached tokens)
3. **Haiku for Sub-Agents**: Use Claude 3.5 Haiku (5-10x cheaper than Sonnet)
4. **Batch Processing**: Use batch API for analytics and summaries
5. **Pinecone Free Tier**: Use for POC to eliminate ~$360/month OpenSearch cost

---

## Summary

This implementation guide provides a comprehensive blueprint for building a voice-based headset troubleshooting agent using the latest AWS multi-agent technologies:

1. **Multi-Agent Architecture**: Supervisor + 3 specialized sub-agents
2. **Latest Technologies**: Bedrock Multi-Agent (GA), Strands Agents SDK v1.0, AgentCore Runtime
3. **Cost-Optimized**: Haiku for sub-agents, Pinecone for POC, prompt caching
4. **Production-Ready Patterns**: Escape hatch, escalation, session management
5. **100% AWS Stack**: Connect, Lex, Lambda (Go), Bedrock, API Gateway

### Next Steps

1. Clone the repository template
2. Deploy foundation infrastructure (Connect, Lex, Lambda)
3. Create sub-agents and supervisor
4. Upload knowledge base documents
5. Test with sample calls
6. Iterate on prompts based on conversation quality
7. Deploy to production with AgentCore Runtime

---

## References

- [Amazon Bedrock Multi-Agent Collaboration](https://docs.aws.amazon.com/bedrock/latest/userguide/agents-multi-agent-collaboration.html)
- [Strands Agents SDK](https://github.com/strands-agents/sdk-python)
- [Amazon Bedrock AgentCore](https://docs.aws.amazon.com/bedrock-agentcore/latest/devguide/what-is-bedrock-agentcore.html)
- [Amazon Connect + Lex Integration](https://docs.aws.amazon.com/connect/latest/adminguide/amazon-lex.html)
- [MCP Protocol](https://modelcontextprotocol.io/)
