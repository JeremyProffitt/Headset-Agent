# Headset Agent Dual-Path AWS Deployment Plan

## Overview

Deploy a voice-based headset troubleshooting agent with **two parallel architectures**:
1. **Path A: Amazon Lex + Bedrock Multi-Agent** - Traditional conversational AI with Bedrock orchestration
2. **Path B: Amazon Bedrock + Nova Sonic** - Direct Bedrock streaming with Nova Sonic bidirectional voice

Each path gets a dedicated phone number for A/B testing and architecture comparison.

---

## Architecture Diagram

```
                    ┌─────────────────────────────────────────────────────────────┐
                    │                    AMAZON CONNECT                            │
                    │  ┌─────────────────┐          ┌─────────────────┐           │
                    │  │ Phone Number A  │          │ Phone Number B  │           │
                    │  │ (Lex Path)      │          │ (Nova Sonic)    │           │
                    │  └────────┬────────┘          └────────┬────────┘           │
                    │           │                            │                     │
                    │  ┌────────▼────────┐          ┌────────▼────────┐           │
                    │  │ Contact Flow A  │          │ Contact Flow B  │           │
                    │  │ (Lex Bot)       │          │ (Bedrock Direct)│           │
                    │  └────────┬────────┘          └────────┬────────┘           │
                    └───────────┼────────────────────────────┼────────────────────┘
                                │                            │
           ┌────────────────────▼────────────────┐   ┌───────▼───────────────────┐
           │          PATH A: LEX                │   │    PATH B: NOVA SONIC     │
           │  ┌──────────────────────────────┐   │   │  ┌─────────────────────┐  │
           │  │      Amazon Lex V2 Bot       │   │   │  │  Bedrock Streaming  │  │
           │  │  - Intent Recognition        │   │   │  │  - Nova Sonic Voice │  │
           │  │  - Slot Filling              │   │   │  │  - Bidirectional    │  │
           │  │  - Dialog Management         │   │   │  │  - Real-time Audio  │  │
           │  └──────────────┬───────────────┘   │   │  └──────────┬──────────┘  │
           │                 │                   │   │             │             │
           │  ┌──────────────▼───────────────┐   │   │  ┌──────────▼──────────┐  │
           │  │    Lambda Orchestrator       │   │   │  │  Lambda Streaming   │  │
           │  │  - Session Management        │   │   │  │  - WebSocket/HTTP2  │  │
           │  │  - Persona Loading           │   │   │  │  - Audio Processing │  │
           │  │  - Bedrock Invocation        │   │   │  │  - Persona Context  │  │
           │  └──────────────┬───────────────┘   │   │  └──────────┬──────────┘  │
           └─────────────────┼───────────────────┘   └─────────────┼─────────────┘
                             │                                     │
                             └──────────────┬──────────────────────┘
                                            │
                    ┌───────────────────────▼───────────────────────┐
                    │         BEDROCK MULTI-AGENT SYSTEM            │
                    │  ┌─────────────────────────────────────────┐  │
                    │  │     Supervisor Agent (Claude Sonnet)    │  │
                    │  │  - Persona-aware orchestration          │  │
                    │  │  - Sub-agent routing                    │  │
                    │  │  - Conversation management              │  │
                    │  └───────────────────┬─────────────────────┘  │
                    │           ┌──────────┼──────────┐             │
                    │  ┌────────▼───┐ ┌────▼────┐ ┌───▼────────┐   │
                    │  │ Diagnostic │ │Platform │ │ Escalation │   │
                    │  │   Agent    │ │  Agent  │ │   Agent    │   │
                    │  │  (Haiku)   │ │ (Haiku) │ │  (Haiku)   │   │
                    │  └────────────┘ └─────────┘ └────────────┘   │
                    └──────────────────────┬────────────────────────┘
                                           │
                    ┌──────────────────────▼────────────────────────┐
                    │              SHARED RESOURCES                  │
                    │  ┌──────────┐ ┌──────────┐ ┌───────────────┐  │
                    │  │ DynamoDB │ │    S3    │ │ Knowledge     │  │
                    │  │ Personas │ │    KB    │ │ Bases (3)     │  │
                    │  │ Sessions │ │  Docs    │ │ HW/SW/Genesys │  │
                    │  └──────────┘ └──────────┘ └───────────────┘  │
                    └───────────────────────────────────────────────┘
```

---

## Phase 1: Foundation Infrastructure
**Goal**: Establish shared AWS resources used by both paths

### 1.1 Core AWS Resources
- [ ] **[SUBAGENT: Explore]** Analyze existing CloudFormation patterns in git history
- [ ] **[SUBAGENT: Plan]** Design SAM template structure for dual-path deployment
- [ ] Create S3 bucket for knowledge base documents
- [ ] Create S3 bucket for SAM deployment artifacts
- [ ] Create DynamoDB table for persona configurations
- [ ] Create DynamoDB table for session state (TTL-enabled)
- [ ] Create CloudWatch log groups for both paths
- [ ] Configure SSM Parameter Store hierarchy:
  ```
  /headset-agent/{env}/
    ├── lex/
    │   ├── bot-id
    │   ├── bot-alias-arn
    │   └── lambda-arn
    ├── nova-sonic/
    │   ├── streaming-lambda-arn
    │   └── websocket-api-id
    ├── bedrock/
    │   ├── supervisor-agent-id
    │   ├── supervisor-alias-id
    │   ├── diagnostic-agent-id
    │   ├── platform-agent-id
    │   └── escalation-agent-id
    └── connect/
        ├── instance-id
        ├── phone-number-lex
        └── phone-number-nova-sonic
  ```

### 1.2 IAM Roles and Policies
- [ ] **[SUBAGENT: Explore]** Review IAM best practices for Bedrock + Connect
- [ ] Create BedrockAgentExecutionRole with:
  - bedrock:InvokeModel permissions
  - bedrock:InvokeAgent permissions
  - bedrock:Retrieve permissions (KB)
  - s3:GetObject for KB bucket
- [ ] Create LambdaExecutionRole (Path A - Lex) with:
  - logs:CreateLogStream, PutLogEvents
  - dynamodb:GetItem, PutItem, Query
  - ssm:GetParameter
  - bedrock-agent-runtime:InvokeAgent
  - lex:RecognizeText, RecognizeUtterance
- [ ] Create LambdaExecutionRole (Path B - Nova Sonic) with:
  - Above permissions plus:
  - bedrock:InvokeModelWithResponseStream
  - bedrock:Converse, ConverseStream
- [ ] Create LexServiceRole with Polly permissions
- [ ] Create ConnectServiceRole with Lex and Lambda permissions

### 1.3 Knowledge Base Setup
- [ ] **[SUBAGENT: Explore]** Review Bedrock KB documentation for best practices
- [ ] Upload knowledge-base/ documents to S3:
  - bluetooth/bt-pairing-failed.md
  - usb/usb-no-audio.md
  - windows/windows-sound-settings.md
  - genesys-cloud/gc-chrome-setup.md
  - common/faq.md
- [ ] Create Bedrock Knowledge Base: `HeadsetHardwareKB`
  - Data source: S3 bucket
  - Embedding model: Amazon Titan Embeddings G1
  - Vector store: OpenSearch Serverless (prod) or Pinecone (POC)
  - Chunking: Semantic chunking (512 tokens)
- [ ] Create Bedrock Knowledge Base: `HeadsetPlatformKB`
  - Windows-specific and Genesys Cloud documents
- [ ] Create Bedrock Knowledge Base: `HeadsetGenesysKB`
  - Genesys Cloud Desktop configuration
- [ ] Test KB queries with sample troubleshooting questions

### 1.4 Persona Deployment
- [ ] Deploy persona configurations to DynamoDB:
  - tangerine (Irish, upbeat - Niamh voice)
  - joseph (Ohio engineer, calm - Matthew voice)
  - jennifer (Nebraska, fast-talking - Joanna voice)
- [ ] Validate persona voice configurations for both Polly and Nova Sonic
- [ ] Create persona selection logic for IVR routing

---

## Phase 2: Bedrock Multi-Agent System
**Goal**: Create the shared AI backbone for both paths

### 2.1 Supervisor Agent
- [ ] **[SUBAGENT: Plan]** Design supervisor agent system prompt with persona integration
- [ ] Create Bedrock Agent: `TroubleshootingOrchestrator-{env}`
  - Model: `anthropic.claude-3-5-sonnet-20241022-v2:0`
  - Instruction: Persona-aware troubleshooting orchestration
  - Enable multi-agent collaboration
- [ ] Create agent alias for routing
- [ ] Configure agent action groups:
  - LoadPersona: Retrieve persona from DynamoDB
  - GetTroubleshootingContext: Query knowledge bases
  - EscalateToHuman: Trigger Connect transfer

### 2.2 Diagnostic Sub-Agent
- [ ] **[SUBAGENT: Explore]** Analyze troubleshooting decision trees from docs
- [ ] Create Bedrock Agent: `DiagnosticAgent-{env}`
  - Model: `anthropic.claude-3-5-haiku-20241022-v1:0`
  - Instruction: USB/Bluetooth hardware diagnosis
  - Knowledge base: HeadsetHardwareKB
- [ ] Define diagnostic action groups:
  - IdentifyConnectionType (USB/Bluetooth/DECT)
  - RunDiagnosticTree (step-by-step troubleshooting)
  - CollectSymptoms (gather user input)

### 2.3 Platform Sub-Agent
- [ ] Create Bedrock Agent: `PlatformAgent-{env}`
  - Model: `anthropic.claude-3-5-haiku-20241022-v1:0`
  - Instruction: Windows and application configuration
  - Knowledge base: HeadsetPlatformKB
- [ ] Define platform action groups:
  - CheckWindowsSettings (sound settings, drivers)
  - ConfigureGenesysCloud (WebRTC, permissions)
  - VerifyDeviceManager (driver status)

### 2.4 Escalation Sub-Agent
- [ ] Create Bedrock Agent: `EscalationAgent-{env}`
  - Model: `anthropic.claude-3-5-haiku-20241022-v1:0`
  - Instruction: Human handoff management
  - Knowledge base: HeadsetGenesysKB
- [ ] Define escalation triggers:
  - User requests human (keywords)
  - Frustration detection (sentiment)
  - Troubleshooting exhausted (step count)
- [ ] Configure Connect transfer metadata

### 2.5 Multi-Agent Collaboration
- [ ] **[SUBAGENT: Explore]** Review Bedrock multi-agent patterns
- [ ] Configure supervisor → sub-agent routing:
  - Hardware issues → DiagnosticAgent
  - Software/config issues → PlatformAgent
  - Escalation requests → EscalationAgent
- [ ] Enable conversation relay between agents
- [ ] Test multi-turn conversations with agent handoffs
- [ ] Store agent IDs in SSM Parameter Store

---

## Phase 3: Path A - Amazon Lex Integration
**Goal**: Traditional Lex-based voice interface with Bedrock backend

### 3.1 Lex V2 Bot Configuration
- [ ] **[SUBAGENT: Plan]** Design Lex bot intent structure
- [ ] Create Lex V2 Bot: `HeadsetTroubleshooterBot-{env}`
  - Language: en-US
  - Voice: Configurable per persona (Niamh/Matthew/Joanna)
  - Child-directed: No
- [ ] Create intents:
  - `TroubleshootIntent` - Main troubleshooting flow
    - Sample utterances: "my headset isn't working", "no audio", "can't hear"
    - Slots: connection_type (USB/Bluetooth), issue_type, headset_brand
  - `SelectPersonaIntent` - Persona selection
    - Slots: persona_name (tangerine/joseph/jennifer)
  - `EscalationIntent` - Human handoff
    - Sample utterances: "speak to agent", "human please", "transfer me"
  - `FallbackIntent` - Unrecognized input handling
- [ ] Configure dialog code hooks for Lambda fulfillment
- [ ] Enable sentiment analysis

### 3.2 Lambda Orchestrator (Path A)
- [ ] **[SUBAGENT: Explore]** Recover Lambda code from git history (commit 1dc249f)
- [ ] Create Go Lambda function: `headset-lex-orchestrator-{env}`
- [ ] Implement handlers:
  - `handleLexEvent()` - Parse Lex V2 events
  - `loadPersona()` - Fetch from DynamoDB
  - `invokeBedrockAgent()` - Call supervisor agent
  - `buildLexResponse()` - Format response with SSML
  - `detectEscalation()` - Check escape hatch triggers
- [ ] Configure environment variables:
  - PERSONA_TABLE_NAME
  - SESSION_TABLE_NAME
  - SUPERVISOR_AGENT_ID
  - SUPERVISOR_ALIAS_ID
  - DEFAULT_PERSONA
- [ ] Build ARM64 binary for Lambda runtime
- [ ] Configure Lambda resource policy for Lex invocation

### 3.3 SSML and Voice Configuration
- [ ] **[SUBAGENT: Explore]** Review Amazon Polly SSML capabilities
- [ ] Implement SSML builder with persona prosody:
  ```xml
  <speak>
    <prosody rate="{persona.rate}" pitch="{persona.pitch}">
      <amazon:domain name="conversational">
        {response_text}
      </amazon:domain>
    </prosody>
  </speak>
  ```
- [ ] Escape special characters (& < > ' ")
- [ ] Configure voice per persona:
  - Tangerine: Niamh (en-IE), Neural, 110% rate, +5% pitch
  - Joseph: Matthew (en-US), Generative, 90% rate, -5% pitch
  - Jennifer: Joanna (en-US), Generative, 115% rate, normal pitch
- [ ] Test voice output quality

### 3.4 Lex-Connect Integration
- [ ] Create Lex bot alias with Lambda fulfillment
- [ ] Associate Lex bot with Connect instance
- [ ] Configure Lex session timeout (5 minutes)
- [ ] Test Lex bot via AWS Console

---

## Phase 4: Path B - Nova Sonic Direct Integration
**Goal**: Bedrock streaming with Nova Sonic bidirectional voice

### 4.1 Nova Sonic Configuration
- [ ] **[SUBAGENT: Explore]** Research Nova Sonic API and streaming patterns
- [ ] **[SUBAGENT: Plan]** Design bidirectional audio streaming architecture
- [ ] Enable Nova Sonic in Bedrock console
- [ ] Configure Nova Sonic voice settings:
  - Voice ID selection for each persona
  - Speech rate and pitch adjustments
  - Conversational style parameters

### 4.2 Streaming Lambda (Path B)
- [ ] Create Go Lambda function: `headset-nova-sonic-{env}`
- [ ] Implement bidirectional streaming:
  - Audio input stream processing
  - Bedrock ConverseStream API integration
  - Real-time audio output generation
- [ ] Implement handlers:
  - `handleStreamingEvent()` - Process audio chunks
  - `invokeBedrockStreaming()` - ConverseStream with Nova Sonic
  - `processAudioResponse()` - Stream audio back to Connect
- [ ] Configure for response streaming:
  - Function URL with streaming enabled
  - Or API Gateway WebSocket
- [ ] Set appropriate timeout (streaming can be long)

### 4.3 Connect Integration (Path B)
- [ ] **[SUBAGENT: Explore]** Research Connect + Bedrock streaming patterns
- [ ] Configure Connect media streaming
- [ ] Set up Kinesis Video Streams for audio capture
- [ ] Create Lambda for audio stream processing
- [ ] Configure bidirectional audio routing

### 4.4 Nova Sonic Voice Mapping
- [ ] Map personas to Nova Sonic voices:
  - Tangerine: Amy (British, closest to Irish)
  - Joseph: Matthew (US Male)
  - Jennifer: Joanna (US Female)
- [ ] Configure voice parameters per persona
- [ ] Test voice quality and latency

---

## Phase 5: Amazon Connect Setup
**Goal**: Configure telephony with dual phone numbers

### 5.1 Connect Instance Configuration
- [ ] **[SUBAGENT: Explore]** Review Connect best practices for multi-path routing
- [ ] Create/configure Connect instance: `headset-support-{env}`
- [ ] Configure instance settings:
  - Data storage (S3 for recordings)
  - Data streaming (Kinesis)
  - Contact flow logs enabled
- [ ] Set up hours of operation (24/7 for POC)
- [ ] Configure agent queues for escalation

### 5.2 Phone Number Claims
- [ ] Claim Phone Number A (Lex Path):
  - Type: TOLL_FREE or DID
  - Country: US
  - Description: "Headset Support - Lex Path"
- [ ] Claim Phone Number B (Nova Sonic Path):
  - Type: TOLL_FREE or DID
  - Country: US
  - Description: "Headset Support - Nova Sonic Path"
- [ ] Store phone numbers in SSM Parameter Store
- [ ] Document phone numbers for testing

### 5.3 Contact Flow A (Lex Path)
- [ ] Create contact flow: `Headset-Support-Lex`
- [ ] Configure flow blocks:
  1. **Set Voice** - Default to Joanna (Neural)
  2. **Play Prompt** - Welcome message
  3. **Get Customer Input** - "Press 1 for Tangerine, 2 for Joseph, 3 for Jennifer"
  4. **Set Contact Attributes** - Store persona selection
  5. **Get Customer Input (Lex)** - Connect to Lex bot
  6. **Check Attribute** - Escalation flag check
  7. **Transfer to Queue** - If escalation requested
  8. **Disconnect** - End call
- [ ] Associate with Phone Number A
- [ ] Test IVR flow

### 5.4 Contact Flow B (Nova Sonic Path)
- [ ] Create contact flow: `Headset-Support-NovaSonic`
- [ ] Configure flow blocks:
  1. **Set Voice** - Nova Sonic voice
  2. **Play Prompt** - Welcome message
  3. **Get Customer Input** - Persona selection
  4. **Set Contact Attributes** - Store persona
  5. **Invoke AWS Lambda** - Call streaming Lambda
  6. **Start Media Streaming** - Enable bidirectional audio
  7. **Loop** - Continuous conversation
  8. **Check Attribute** - Escalation check
  9. **Transfer to Queue** - Human handoff
  10. **Disconnect** - End call
- [ ] Associate with Phone Number B
- [ ] Test streaming flow

### 5.5 Escalation Queue Setup
- [ ] Create queue: `HeadsetSupport-Escalation`
- [ ] Configure queue settings:
  - Max contacts in queue: 10
  - Queue timeout: 300 seconds
- [ ] Create routing profile for human agents
- [ ] Set up quick connect for transfers

---

## Phase 6: GitHub Actions CI/CD Pipeline
**Goal**: Automated deployment via GitHub Actions (per CLAUDE.md policy)

### 6.1 Workflow Structure
- [ ] **[SUBAGENT: Explore]** Recover workflow files from git history
- [ ] **[SUBAGENT: Plan]** Design dual-path deployment workflow
- [ ] Create `.github/workflows/deploy.yml`:

```yaml
name: Deploy Headset Agent (Dual Path)

on:
  push:
    branches: [main, 'release/*']
  workflow_dispatch:
    inputs:
      environment:
        description: 'Target environment'
        required: true
        default: 'dev'
        type: choice
        options: [dev, staging, prod]
      deploy_lex_path:
        description: 'Deploy Lex Path (A)'
        type: boolean
        default: true
      deploy_nova_sonic_path:
        description: 'Deploy Nova Sonic Path (B)'
        type: boolean
        default: true

jobs:
  # See detailed job definitions below
```

### 6.2 Pipeline Jobs
- [ ] **Job 1: setup** - Environment determination
- [ ] **Job 2: validate** - Validate secrets and variables
- [ ] **Job 3: build** - Build Go Lambda binaries (both paths)
- [ ] **Job 4: test** - Run unit tests with coverage
- [ ] **Job 5: deploy-infrastructure** - SAM deploy shared resources
- [ ] **Job 6: sync-knowledge-base** - Upload KB documents to S3
- [ ] **Job 7: deploy-personas** - Load personas to DynamoDB
- [ ] **Job 8: create-bedrock-agents** - Create/update Bedrock agents
- [ ] **Job 9: deploy-lex-path** - Deploy Lex bot and Lambda (Path A)
- [ ] **Job 10: deploy-nova-sonic-path** - Deploy streaming Lambda (Path B)
- [ ] **Job 11: configure-connect** - Set up Connect flows and phone numbers
- [ ] **Job 12: integration-tests** - End-to-end testing
- [ ] **Job 13: validate-deployment** - Health checks
- [ ] **Job 14: notify** - Success/failure notifications

### 6.3 Supporting Workflows
- [ ] Create `.github/workflows/pr-validation.yml`:
  - Lint (gofmt, golangci-lint)
  - Unit tests
  - SAM template validation
  - Security scan (gosec)
  - Persona JSON validation
- [ ] Create `.github/workflows/destroy.yml`:
  - Manual trigger with confirmation
  - Delete Bedrock agents
  - Release phone numbers
  - Delete CloudFormation stack
  - Clean up S3 and DynamoDB

### 6.4 GitHub Configuration
- [ ] Configure repository secrets:
  - AWS_ACCESS_KEY_ID
  - AWS_SECRET_ACCESS_KEY
  - AWS_ACCOUNT_ID
  - CONNECT_INSTANCE_ID
  - CONNECT_INSTANCE_ARN
- [ ] Configure repository variables:
  - AWS_REGION (us-east-1)
  - BEDROCK_MODEL_SUPERVISOR
  - BEDROCK_MODEL_SUBAGENT
  - All other required variables per docs/variables.md
- [ ] Set up environment protection rules (optional for POC)

---

## Phase 7: Lambda Code Development
**Goal**: Implement Lambda functions for both paths

### 7.1 Shared Modules
- [ ] **[SUBAGENT: Explore]** Recover internal packages from git history
- [ ] Create `internal/models/types.go`:
  - Persona struct
  - VoiceConfig struct
  - Session struct
  - EscalationDecision struct
- [ ] Create `internal/persona/loader.go`:
  - DynamoDB persona loader
  - Caching for performance
  - Default persona fallback
- [ ] Create `internal/agents/bedrock.go`:
  - BedrockClient wrapper
  - InvokeAgent implementation
  - Stream processing
- [ ] Create `internal/handlers/escalation.go`:
  - Keyword detection
  - Frustration analysis
  - Escalation response builder

### 7.2 Path A Lambda (Lex)
- [ ] Create `cmd/lex-lambda/main.go`:
  - Lex V2 event handler
  - Dialog fulfillment
  - SSML response formatting
- [ ] Create `internal/handlers/lex.go`:
  - LexV2Response builder
  - Session state management
  - Intent routing
- [ ] Write unit tests for Lex handlers
- [ ] Build and test locally with SAM

### 7.3 Path B Lambda (Nova Sonic)
- [ ] **[SUBAGENT: Explore]** Research Bedrock streaming SDK patterns
- [ ] Create `cmd/nova-sonic-lambda/main.go`:
  - Streaming event handler
  - Audio chunk processing
  - Response streaming
- [ ] Create `internal/handlers/streaming.go`:
  - ConverseStream integration
  - Audio buffer management
  - Real-time response delivery
- [ ] Write unit tests for streaming handlers
- [ ] Test with mock audio streams

### 7.4 Build Configuration
- [ ] Create `Makefile` with targets:
  - `build-lex`: Build Lex Lambda binary
  - `build-nova-sonic`: Build Nova Sonic Lambda binary
  - `test`: Run all unit tests
  - `lint`: Run linters
  - `clean`: Clean build artifacts
- [ ] Configure `go.mod` with dependencies:
  - github.com/aws/aws-lambda-go
  - github.com/aws/aws-sdk-go-v2
  - github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime
  - github.com/aws/aws-sdk-go-v2/service/bedrockruntime
  - github.com/aws/aws-sdk-go-v2/service/dynamodb
  - github.com/aws/aws-sdk-go-v2/service/ssm

---

## Phase 8: SAM Template Development
**Goal**: Infrastructure as Code for dual-path deployment

### 8.1 Template Structure
- [ ] **[SUBAGENT: Plan]** Design modular SAM template structure
- [ ] Create `infrastructure/template.yaml`:

```yaml
AWSTemplateFormatVersion: '2010-09-09'
Transform: AWS::Serverless-2016-10-31
Description: Headset Agent - Dual Path (Lex + Nova Sonic)

Parameters:
  Environment:
    Type: String
    AllowedValues: [dev, staging, prod]
  DeployLexPath:
    Type: String
    Default: 'true'
  DeployNovaSonicPath:
    Type: String
    Default: 'true'
  # ... additional parameters

Conditions:
  DeployLex: !Equals [!Ref DeployLexPath, 'true']
  DeployNovaSonic: !Equals [!Ref DeployNovaSonicPath, 'true']

Resources:
  # Shared Resources
  # Path A Resources (Condition: DeployLex)
  # Path B Resources (Condition: DeployNovaSonic)

Outputs:
  # Shared Outputs
  # Path A Outputs
  # Path B Outputs
```

### 8.2 Shared Resources
- [ ] Define S3 buckets (KB, Website)
- [ ] Define DynamoDB tables (Personas, Sessions)
- [ ] Define IAM roles (shared permissions)
- [ ] Define CloudWatch log groups
- [ ] Define SSM parameters

### 8.3 Path A Resources
- [ ] Define Lex Lambda function
- [ ] Define Lex bot (via custom resource or manual)
- [ ] Define Lambda resource policy for Lex
- [ ] Define API Gateway for web chat (optional)

### 8.4 Path B Resources
- [ ] Define Nova Sonic Lambda function
- [ ] Define Lambda function URL (streaming)
- [ ] Define Kinesis Video Streams (if needed)
- [ ] Define additional IAM for streaming

### 8.5 Outputs
- [ ] Export all resource ARNs
- [ ] Export phone numbers (after Connect config)
- [ ] Export API endpoints
- [ ] Export testing URLs

---

## Phase 9: Python Deployment Scripts
**Goal**: Automation scripts for Bedrock and Connect configuration

### 9.1 Agent Creation Script
- [ ] **[SUBAGENT: Explore]** Recover create-agents.py from git history
- [ ] Create `scripts/create-agents.py`:
  - Supervisor agent creation
  - Sub-agent creation (3 agents)
  - Agent preparation and aliasing
  - Multi-agent collaboration setup
  - SSM parameter storage
- [ ] Add error handling and retry logic
- [ ] Add --environment parameter
- [ ] Add --dry-run option for testing

### 9.2 Nova Sonic Configuration Script
- [ ] Create `scripts/configure-nova-sonic.py`:
  - Nova Sonic voice configuration
  - Lex V2 voice settings update
  - Bidirectional streaming setup
- [ ] Add persona voice mapping

### 9.3 Connect Configuration Script
- [ ] Create `scripts/configure-connect.py`:
  - Phone number claiming
  - Contact flow deployment
  - Lex bot association
  - Queue and routing setup
- [ ] Add dual-path support (two phone numbers)
- [ ] Add rollback capability

### 9.4 Knowledge Base Script
- [ ] Create `scripts/create-knowledge-bases.py`:
  - S3 data source configuration
  - Bedrock KB creation
  - Embedding model setup
  - Vector store configuration
- [ ] Add sync/update capability

---

## Phase 10: Testing Strategy
**Goal**: Comprehensive testing for both paths

### 10.1 Unit Tests
- [ ] **[SUBAGENT: Explore]** Recover test files from git history
- [ ] Create Go unit tests:
  - `cmd/lex-lambda/main_test.go`
  - `cmd/nova-sonic-lambda/main_test.go`
  - `internal/handlers/*_test.go`
  - `internal/agents/*_test.go`
  - `internal/persona/*_test.go`
- [ ] Create Python unit tests:
  - `tests/unit/test_create_agents.py`
  - `tests/unit/test_configure_connect.py`
- [ ] Achieve >80% code coverage

### 10.2 Integration Tests
- [ ] Create `tests/integration/`:
  - `test_lex_path.py` - End-to-end Lex flow
  - `test_nova_sonic_path.py` - End-to-end streaming flow
  - `test_bedrock_agents.py` - Agent invocation
  - `test_personas.py` - Persona loading
  - `test_knowledge_base.py` - KB queries
- [ ] Add pytest fixtures for AWS resources
- [ ] Configure test environment variables

### 10.3 Voice Testing
- [ ] Test each persona voice (Path A):
  - Tangerine (Niamh) - Irish accent quality
  - Joseph (Matthew) - Calm, measured delivery
  - Jennifer (Joanna) - Fast, clear speech
- [ ] Test Nova Sonic voices (Path B):
  - Quality comparison with Polly
  - Latency measurements
  - Persona personality preservation
- [ ] Document voice quality findings

### 10.4 End-to-End Testing
- [ ] Create test scripts for phone calls:
  - Call Phone Number A → Lex Path
  - Call Phone Number B → Nova Sonic Path
- [ ] Test scenarios:
  - USB headset troubleshooting
  - Bluetooth pairing issues
  - Genesys Cloud configuration
  - Escalation to human
- [ ] Compare path performance:
  - Response latency
  - Voice quality
  - Conversation flow
  - Escalation handling

---

## Phase 11: Deployment Execution
**Goal**: Execute deployment via GitHub Actions

### 11.1 Initial Deployment
- [ ] Commit all code and configuration
- [ ] Push to main branch
- [ ] Monitor GitHub Actions workflow:
  - [ ] setup job
  - [ ] validate job
  - [ ] build job
  - [ ] test job
  - [ ] deploy-infrastructure job
  - [ ] sync-knowledge-base job
  - [ ] deploy-personas job
  - [ ] create-bedrock-agents job
  - [ ] deploy-lex-path job
  - [ ] deploy-nova-sonic-path job
  - [ ] configure-connect job
  - [ ] integration-tests job
  - [ ] validate-deployment job
- [ ] **[SUBAGENT: fix-pipeline]** If any job fails, use fix-pipeline skill

### 11.2 Verification
- [ ] Verify CloudFormation stack status
- [ ] Verify Bedrock agents are PREPARED
- [ ] Verify Lex bot is AVAILABLE
- [ ] Verify Lambda functions are deployed
- [ ] Verify Connect contact flows are published
- [ ] Verify phone numbers are claimed
- [ ] Test both paths with phone calls

### 11.3 Post-Deployment
- [ ] Document deployed resources
- [ ] Record phone numbers:
  - Phone Number A (Lex): _______________
  - Phone Number B (Nova Sonic): _______________
- [ ] Update README with testing instructions
- [ ] Create user guide for testers

---

## Phase 12: Monitoring and Operations
**Goal**: Production-ready observability

### 12.1 CloudWatch Dashboards
- [ ] Create dashboard: `HeadsetAgent-{env}`
- [ ] Add widgets:
  - Lambda invocations (both paths)
  - Lambda errors and duration
  - Bedrock agent invocations
  - Connect call metrics
  - DynamoDB read/write capacity

### 12.2 Alarms
- [ ] Create alarms:
  - Lambda error rate > 5%
  - Lambda duration > 25 seconds
  - Bedrock throttling
  - Connect abandoned calls
- [ ] Configure SNS notifications
- [ ] Set up PagerDuty/Slack integration (optional)

### 12.3 Logging
- [ ] Configure structured logging in Lambda
- [ ] Set log retention:
  - Dev: 14 days
  - Staging: 30 days
  - Prod: 90 days
- [ ] Create CloudWatch Insights queries:
  - Error analysis
  - Persona usage statistics
  - Escalation tracking

### 12.4 Cost Tracking
- [ ] Enable cost allocation tags
- [ ] Create Cost Explorer report
- [ ] Set up budget alerts
- [ ] Document expected costs:
  - Bedrock (Sonnet + Haiku)
  - Connect (voice minutes)
  - Lambda (invocations)
  - DynamoDB (read/write)

---

## Subagent Usage Summary

Throughout this plan, use subagents for:

| Task Type | Subagent | Purpose |
|-----------|----------|---------|
| Codebase exploration | `Explore` | Recover code from git history, find patterns |
| Architecture design | `Plan` | Design system components, review trade-offs |
| Pipeline failures | `fix-pipeline` | Automated GitHub Actions remediation |
| Lambda errors | `fix-lambda` | AWS Lambda error detection and fixes |
| Documentation lookup | `claude-code-guide` | Claude Code and Agent SDK guidance |
| Parallel research | `general-purpose` | Multi-step research tasks |

**Aggressive Subagent Strategy**:
1. **Before any major phase**: Launch Explore agent to gather context
2. **For complex decisions**: Launch Plan agent to evaluate options
3. **On any failure**: Immediately launch fix-pipeline or fix-lambda
4. **For parallel work**: Launch multiple agents simultaneously

---

## Success Criteria

### Functional Requirements
- [ ] Both phone numbers answer calls
- [ ] All three personas work correctly on both paths
- [ ] USB troubleshooting flow completes successfully
- [ ] Bluetooth troubleshooting flow completes successfully
- [ ] Genesys Cloud setup guidance works
- [ ] Escalation to human agent functions
- [ ] Knowledge base queries return relevant information

### Performance Requirements
- [ ] Path A (Lex) response time < 3 seconds
- [ ] Path B (Nova Sonic) response time < 2 seconds
- [ ] Voice quality rated acceptable by testers
- [ ] No dropped calls during troubleshooting
- [ ] Session state maintained across turns

### Operational Requirements
- [ ] GitHub Actions deployment succeeds
- [ ] All resources tagged correctly
- [ ] Monitoring dashboards operational
- [ ] Alarms configured and tested
- [ ] Documentation complete

---

## Timeline Estimate

| Phase | Description | Complexity |
|-------|-------------|------------|
| 1 | Foundation Infrastructure | Medium |
| 2 | Bedrock Multi-Agent | High |
| 3 | Lex Path (A) | Medium |
| 4 | Nova Sonic Path (B) | High |
| 5 | Connect Setup | Medium |
| 6 | CI/CD Pipeline | Medium |
| 7 | Lambda Development | High |
| 8 | SAM Template | Medium |
| 9 | Python Scripts | Medium |
| 10 | Testing | Medium |
| 11 | Deployment | Low |
| 12 | Monitoring | Low |

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Bedrock model access denied | Apply for Claude access via AWS console; fallback to Llama models |
| Nova Sonic not available | Deploy Path A only; add Path B when available |
| Connect phone number unavailable | Use different region or DID instead of toll-free |
| Pipeline failures | Use fix-pipeline skill aggressively; iterate until success |
| Voice quality issues | Test early; adjust SSML prosody settings |
| Cost overruns | Set budget alerts; use Haiku for sub-agents |

---

## Quick Reference: Key Resources

**Phone Numbers** (after deployment):
- Path A (Lex): `/headset-agent/{env}/connect/phone-number-lex`
- Path B (Nova Sonic): `/headset-agent/{env}/connect/phone-number-nova-sonic`

**Bedrock Agents**:
- Supervisor: `TroubleshootingOrchestrator-{env}`
- Diagnostic: `DiagnosticAgent-{env}`
- Platform: `PlatformAgent-{env}`
- Escalation: `EscalationAgent-{env}`

**Lambda Functions**:
- Path A: `headset-lex-orchestrator-{env}`
- Path B: `headset-nova-sonic-{env}`

**Documentation**:
- Deployment: `docs/deployment-guide.md`
- Implementation: `docs/headset-agent-implementation-guide.md`
- Personas: `docs/persona-troubleshooting-guide.md`
- Variables: `docs/variables.md`
