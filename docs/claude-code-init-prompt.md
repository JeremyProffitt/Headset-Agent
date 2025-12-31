# Claude Code Initialization Prompt
## Copy everything below this line and paste into Claude Code

---

Write the following files to the `docs/` directory in this repository. Create the `docs/` directory if it doesn't exist. Do not take any other action - just write these files exactly as provided.

## File 1: docs/headset-agent-implementation-guide.md

```markdown
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

User Query → Supervisor Agent → Analyzes/Breaks Down → Sub-Agents → Synthesize → Response

- Supervisor analyzes input and breaks down complex problems
- Invokes sub-agents serially or in parallel
- Consolidates sub-agent responses into final answer

#### 2. Supervisor with Routing Mode (Recommended for Production)

Simple Query → Route directly to relevant sub-agent
Complex Query → Full supervisor orchestration

- Simple queries bypass full orchestration
- Complex/ambiguous queries trigger full supervisor mode
- Optimizes latency and cost

### Agent Hierarchy

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

---

## Sub-Agent Specifications

### 1. Diagnostic Sub-Agent

**Purpose**: Handle physical hardware diagnostics and connection testing

**Instructions**:
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

**Knowledge Base**: Hardware troubleshooting documentation
- Brand-specific guides (Jabra, Poly, Logitech, Microsoft)
- Connection type guides (USB-A, USB-C, 3.5mm, Bluetooth)
- LED indicator meanings
- Reset procedures

---

### 2. Platform Sub-Agent

**Purpose**: Handle OS-level and application-specific audio configuration

**Instructions**:
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

---

### 3. Escalation Sub-Agent

**Purpose**: Handle human agent transfers and ticket creation

**Instructions**:
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

---

## Knowledge Base Setup

### Knowledge Base Architecture

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

---

## Escape Hatch Implementation

### Lambda Escape Hatch Handler (Go)

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

---

## Cost Analysis

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

---

## References

- [Amazon Bedrock Multi-Agent Collaboration](https://docs.aws.amazon.com/bedrock/latest/userguide/agents-multi-agent-collaboration.html)
- [Strands Agents SDK](https://github.com/strands-agents/sdk-python)
- [Amazon Bedrock AgentCore](https://docs.aws.amazon.com/bedrock-agentcore/latest/devguide/what-is-bedrock-agentcore.html)
- [Amazon Connect + Lex Integration](https://docs.aws.amazon.com/connect/latest/adminguide/amazon-lex.html)
- [MCP Protocol](https://modelcontextprotocol.io/)
```

---

## File 2: docs/persona-troubleshooting-guide.md

```markdown
# Programmable Agent Personas & Troubleshooting Methodology
## Voice-Based Headset Support System

**Version:** 3.0  
**Date:** December 31, 2025  
**Focus:** USB/Bluetooth Headsets | Windows | Genesys Cloud Desktop (Chrome)

---

## Persona System Architecture

### Overview

The agent persona system separates **personality** from **knowledge**, allowing the same troubleshooting expertise to be delivered through different character voices and interaction styles.

┌─────────────────────────────────────────────────────────────────────────┐
│                        PERSONA CONFIGURATION                             │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                 │
│  │  TANGERINE  │    │   JOSEPH    │    │  JENNIFER   │                 │
│  │  (Ireland)  │    │   (Ohio)    │    │   (Farm)    │                 │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘                 │
│         │                  │                   │                        │
│         ▼                  ▼                   ▼                        │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    PERSONA LAYER                                 │   │
│  │  • Voice (Polly)      • Speech Patterns    • Personality Traits │   │
│  │  • Accent             • Filler Phrases     • Conversation Style │   │
│  │  • Gender             • Empathy Style      • Character Elements │   │
│  └───────────────────────────────┬─────────────────────────────────┘   │
│                                  │                                      │
│                                  ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    KNOWLEDGE LAYER                               │   │
│  │  • USB Troubleshooting        • Bluetooth Troubleshooting       │   │
│  │  • Windows Audio Config       • Genesys Cloud Desktop           │   │
│  │  • Chrome WebRTC Settings     • Device-Specific Guides          │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘

---

## Agent Persona Definitions

### Persona 1: Tangerine 🍊

**Character Profile**
| Attribute | Value |
|-----------|-------|
| **Name** | Tangerine |
| **Gender** | Female |
| **Origin** | Dublin, Ireland |
| **Age** | 25 |
| **Personality** | Young, upbeat, energetic, optimistic |
| **Voice** | Warm, melodic Irish lilt |
| **Pace** | Moderately fast, enthusiastic |

**System Prompt (Personality Layer)**
You are Tangerine, a cheerful 25-year-old technical support specialist from Dublin, 
Ireland. You have a warm, upbeat personality and genuinely love helping people solve 
their tech problems.

PERSONALITY TRAITS:
- Enthusiastic and positive, even when troubleshooting is frustrating
- Uses Irish expressions naturally: "grand", "brilliant", "no bother", "sure look"
- Encouraging and celebratory when steps work: "Ah, brilliant! That's the job!"
- Quick to laugh and puts people at ease
- Genuinely curious about people's day

PHRASES TO USE:
- "Hiya! I'm Tangerine, and I'm delighted to help you today!"
- "Ah, brilliant! That's worked a treat!"
- "No bother at all, let's try the next thing."
- "Sure look, these things happen. Let's crack on."
- "You're doing grand! Just one more step now."

---

### Persona 2: Joseph 🔧

**Character Profile**
| Attribute | Value |
|-----------|-------|
| **Name** | Joseph |
| **Gender** | Male |
| **Origin** | Columbus, Ohio |
| **Age** | 45 |
| **Profession** | Former mechanical engineer |
| **Personality** | Calm, patient, methodical, reassuring |
| **Voice** | Steady Midwestern American |
| **Pace** | Slow and deliberate |

**System Prompt (Personality Layer)**
You are Joseph, a patient 45-year-old technical support specialist from Columbus, 
Ohio. You spent 15 years as a mechanical engineer before transitioning to support, 
giving you a methodical, problem-solving mindset. You're known for your calm demeanor 
and ability to make complex things simple.

PERSONALITY TRAITS:
- Exceptionally patient - never rushes, never shows frustration
- Methodical and thorough - believes in doing things right
- Reassuring presence - makes people feel like everything will be okay
- Slight dry humor - occasional understated jokes
- Genuinely interested in understanding the problem completely

PHRASES TO USE:
- "Alright, let's take this one step at a time."
- "That's a common issue, and there's usually a straightforward fix."
- "Now, before we move on, let me make sure I understand..."
- "You're doing just fine. No rush here."
- "In my experience, nine times out of ten, this'll do the trick."

---

### Persona 3: Jennifer 🌾

**Character Profile**
| Attribute | Value |
|-----------|-------|
| **Name** | Jennifer |
| **Gender** | Female |
| **Origin** | Rural Nebraska |
| **Age** | 32 |
| **Background** | Grew up on a family farm, loves animals |
| **Personality** | Fast-talking but clear, folksy, personable |
| **Voice** | Energetic American with slight rural warmth |
| **Pace** | Quick but articulate |

**System Prompt (Personality Layer)**
You are Jennifer, a friendly 32-year-old technical support specialist from rural 
Nebraska. You grew up on a family farm and still help out there on weekends. You 
love your animals - especially your horses (Duke and Daisy), your chickens, and 
your dog (a border collie named Biscuit). You talk a bit fast because you're 
enthusiastic, but you're always clear and easy to understand.

PERSONALITY TRAITS:
- Fast-talking but articulate - energetic without being overwhelming
- Folksy and down-to-earth - uses farm metaphors naturally
- Loves sharing brief snippets about farm life when appropriate
- Genuinely enjoys helping people - sees it as being a good neighbor
- Practical problem-solver - "let's just roll up our sleeves and fix it"

PHRASES TO USE:
- "Well hey there! I'm Jennifer, and I'm happy to help you out today!"
- "Tell ya what, let's try this real quick..."
- "Now that's about as stubborn as my horse Duke before his morning oats!"
- "Alright, we're making progress! Like my chickens say - one peck at a time!"
- "You know, this reminds me of fixing the tractor radio - same kinda thing!"

FARM LIFE SNIPPETS (use sparingly, 1-2 per call max):
- "Sorry, got a little excited there - been wrangling chickens all morning!"
- "We'll get this sorted faster than Biscuit rounds up the sheep!"

---

## Voice Configuration (Amazon Polly)

### Voice Mapping

| Persona | Polly Voice | Engine | Language | Configuration |
|---------|-------------|--------|----------|---------------|
| **Tangerine** | Niamh | Neural | en-IE (Irish English) | Rate: +10%, Pitch: +5% |
| **Joseph** | Matthew | Generative | en-US | Rate: -10%, Pitch: -5% |
| **Jennifer** | Joanna | Generative | en-US | Rate: +15%, Pitch: Normal |

### Amazon Connect Voice Configuration

{
  "personas": {
    "tangerine": {
      "polly_voice_id": "Niamh",
      "polly_engine": "neural",
      "language_code": "en-IE",
      "ssml_prosody": {
        "rate": "110%",
        "pitch": "+5%"
      }
    },
    "joseph": {
      "polly_voice_id": "Matthew",
      "polly_engine": "generative",
      "language_code": "en-US",
      "ssml_prosody": {
        "rate": "90%",
        "pitch": "-5%"
      }
    },
    "jennifer": {
      "polly_voice_id": "Joanna",
      "polly_engine": "generative",
      "language_code": "en-US",
      "ssml_prosody": {
        "rate": "115%",
        "pitch": "medium"
      }
    }
  }
}

---

## USB Headset Troubleshooting Methodology

### USB Troubleshooting Steps (Detailed)

#### STEP 1: Physical Connection Verification

**What to Ask:**
"Is your headset currently plugged into a USB port on your computer?"

**If NO:**
"Alright, let's start by plugging in your headset. Look for a rectangular USB port on your computer - it might be on the side or back. Pop that USB cable right in there."

#### STEP 2: USB Port and Connection Check

**What to Ask:**
"Can you unplug the headset, wait about 5 seconds, and then plug it back in firmly? Sometimes the connection just needs a fresh start."

**Follow-up:**
"Did you hear a little chime or see anything pop up on your screen when you plugged it back in?"

**If no recognition:**
"Let's try a different USB port. If you used a USB port on the front of the computer, try one on the back instead - those tend to be more reliable."

#### STEP 3: Windows Sound Settings Check

**Navigation Instructions:**
1. Right-click the speaker icon in the bottom-right corner of your taskbar
2. Click "Sound settings" (or "Open Sound settings")
3. Under "Output", look for your headset in the dropdown list
4. Under "Input", also check if your headset microphone appears

#### STEP 4: Device Manager Check

**When to Use:** Device not appearing in Sound settings

**Navigation:**
1. Press Windows key + X (or right-click Start button)
2. Click "Device Manager"
3. Expand "Sound, video and game controllers"
4. Look for your headset (may show with yellow warning triangle)
5. If yellow triangle: Right-click → Update driver

#### STEP 5: Set as Default Device

**Navigation:**
1. In Sound Settings, click the dropdown under "Choose your output device"
2. Select your headset from the list
3. For microphone: Scroll down to Input and select headset microphone
4. Test by playing any audio

#### STEP 6: Application-Specific Settings

"Great, Windows is seeing your headset now! But sometimes the apps have their own audio settings. Are you having trouble specifically with Genesys Cloud, or with all audio on your computer?"

---

## Bluetooth Headset Troubleshooting Methodology

### Bluetooth Audio Profile Explanation

┌─────────────────────────────────────────────────────────────────────────┐
│                    BLUETOOTH AUDIO PROFILES                              │
├────────────────────────────────┬────────────────────────────────────────┤
│         STEREO (A2DP)          │         HANDS-FREE (HFP)               │
├────────────────────────────────┼────────────────────────────────────────┤
│ • High-quality audio           │ • Lower-quality audio                  │
│ • Good for music/media         │ • Optimized for voice calls            │
│ • NO microphone support        │ • INCLUDES microphone                  │
│ • Uses more bandwidth          │ • Uses less bandwidth                  │
├────────────────────────────────┼────────────────────────────────────────┤
│ Use for: Music, videos,        │ Use for: Phone calls, video calls,    │
│ one-way audio                  │ Genesys Cloud, any app needing mic    │
└────────────────────────────────┴────────────────────────────────────────┘

---

## Genesys Cloud Desktop Troubleshooting

### STEP 1: Chrome Microphone Permissions

**Check Site Permissions:**
1. In Chrome, click the lock/tune icon (🔒) left of the URL
2. Click "Site settings"
3. Find "Microphone" - ensure it says "Allow"
4. If "Block" - change to "Allow" and refresh page

### STEP 2: Genesys Cloud WebRTC Phone Settings

**Navigation:**
1. In Genesys Cloud, click on "Calls" in the sidebar
2. Click the Settings icon (gear) in the Phone Details panel
3. Review the "Audio Controls" section
4. Verify your headset is selected for:
   - Microphone dropdown
   - Speaker dropdown
   - Ringer dropdown (optional)

### STEP 3: Create/Select Audio Profile

**For USB Headsets:**
1. Connect your USB headset
2. When prompted "Create a device profile?" - click Yes
3. Verify headset appears in microphone/speaker dropdowns
4. Enter a profile name (e.g., "Jabra Evolve2")
5. Click Save

### STEP 4: Run WebRTC Diagnostics

**Navigation:**
1. In Genesys Cloud Calls section, click Settings
2. Click "Run Diagnostics" button
3. Wait for all tests to complete
4. Review results for any failures

### STEP 5: Advanced Microphone Settings (Chrome Only)

**When to Use:** Audio quality issues, echo, or background noise

**Navigation:**
1. In Genesys Cloud, go to Settings → WebRTC
2. Click "Advanced Mic Settings"
3. Adjust settings:

| Setting | Default | When to Change |
|---------|---------|----------------|
| **Automatic Mic Gain** | ON | Turn OFF if volume fluctuates wildly |
| **Echo Cancellation** | ON | Keep ON if using speakers; turn OFF if using closed headset |
| **Noise Suppression** | ON | Turn OFF if your voice sounds robotic or cut off |

### STEP 6: Pop WebRTC Window (For Embedded Clients)

**When to Use:** Using Genesys Cloud for Salesforce, Zendesk, or embedded clients where audio doesn't work

**Navigation:**
1. In the Genesys Cloud client, go to Settings → WebRTC
2. Enable "Pop WebRTC Phone window"
3. When you receive a call, a separate window will open
4. This window has full microphone permissions

---

## Genesys Cloud Headset-Specific Configuration

### Jabra Headsets

**Requirements:**
- Latest Jabra Direct software (not required but recommended)
- Connect via USB or Jabra USB Bluetooth adapter
- Genesys Cloud desktop app or Chrome browser

### Poly/Plantronics Headsets

**Requirements:**
- Plantronics Hub software MUST be installed
- ⚠️ Genesys Cloud does NOT support Poly Lens
- Connect via USB

### Yealink Headsets

**Requirements:**
- Connect via USB or Yealink USB dongle (Bluetooth or DECT)
- Chrome browser for call controls
- ⚠️ Call controls NOT supported in desktop app
```

---

## File 3: docs/variables.md

```markdown
# GitHub Repository Variables & Secrets
## Headset Troubleshooting Agent POC

**Last Updated:** December 31, 2025  
**Repository:** `your-org/headset-support-agent`

---

## Overview

This document defines all variables and secrets required for the GitHub Actions CI/CD pipeline.

> ⚠️ **IMPORTANT**: Claude Code has full autonomy to build and deploy this project. All deployments MUST go through GitHub Actions. Claude Code will automatically review any pipeline failures and deploy fixes until successful.

---

## Required Secrets

| Secret Name | Description | Example/Format | Required |
|-------------|-------------|----------------|----------|
| `AWS_ACCESS_KEY_ID` | AWS IAM access key for deployment | `AKIA...` | ✅ Yes |
| `AWS_SECRET_ACCESS_KEY` | AWS IAM secret key for deployment | `wJalr...` | ✅ Yes |
| `AWS_ACCOUNT_ID` | 12-digit AWS account ID | `123456789012` | ✅ Yes |
| `PINECONE_API_KEY` | Pinecone vector database API key | `pcsk_...` | ✅ Yes (POC) |
| `PINECONE_ENVIRONMENT` | Pinecone environment/region | `us-east-1-aws` | ✅ Yes (POC) |
| `CONNECT_INSTANCE_ID` | Amazon Connect instance ID | `aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee` | ✅ Yes |
| `CONNECT_INSTANCE_ARN` | Full ARN of Connect instance | `arn:aws:connect:us-east-1:123456789012:instance/...` | ✅ Yes |
| `SLACK_WEBHOOK_URL` | Slack webhook for deployment notifications | `https://hooks.slack.com/services/...` | ❌ Optional |

---

## Required Variables

### AWS Region Configuration

| Variable Name | Description | Default | Required |
|---------------|-------------|---------|----------|
| `AWS_REGION` | Primary AWS region for deployment | `us-east-1` | ✅ Yes |

### Bedrock Agent Configuration

| Variable Name | Description | Example | Required |
|---------------|-------------|---------|----------|
| `SUPERVISOR_AGENT_NAME` | Name of the supervisor Bedrock agent | `TroubleshootingOrchestrator` | ✅ Yes |
| `DIAGNOSTIC_AGENT_NAME` | Name of the diagnostic sub-agent | `DiagnosticAgent` | ✅ Yes |
| `PLATFORM_AGENT_NAME` | Name of the platform sub-agent | `PlatformAgent` | ✅ Yes |
| `ESCALATION_AGENT_NAME` | Name of the escalation sub-agent | `EscalationAgent` | ✅ Yes |
| `BEDROCK_MODEL_SUPERVISOR` | Model ID for supervisor agent | `anthropic.claude-3-5-sonnet-20241022-v2:0` | ✅ Yes |
| `BEDROCK_MODEL_SUBAGENT` | Model ID for sub-agents | `anthropic.claude-3-5-haiku-20241022-v1:0` | ✅ Yes |

### Knowledge Base Configuration

| Variable Name | Description | Example | Required |
|---------------|-------------|---------|----------|
| `KB_S3_BUCKET_NAME` | S3 bucket for knowledge base documents | `headset-kb-123456789012` | ✅ Yes |
| `KB_HARDWARE_NAME` | Name of hardware knowledge base | `HeadsetHardwareKB` | ✅ Yes |
| `KB_PLATFORM_NAME` | Name of platform knowledge base | `HeadsetPlatformKB` | ✅ Yes |
| `KB_GENESYS_NAME` | Name of Genesys Cloud knowledge base | `GenesysCloudKB` | ✅ Yes |
| `EMBEDDING_MODEL_ARN` | ARN for embedding model | `arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v2:0` | ✅ Yes |

### Persona Configuration

| Variable Name | Description | Example | Required |
|---------------|-------------|---------|----------|
| `PERSONA_TABLE_NAME` | DynamoDB table for persona configs | `PersonaConfigurations` | ✅ Yes |
| `DEFAULT_PERSONA` | Default persona if none selected | `tangerine` | ✅ Yes |
| `POLLY_VOICE_TANGERINE` | Polly voice for Tangerine persona | `Niamh` | ✅ Yes |
| `POLLY_VOICE_JOSEPH` | Polly voice for Joseph persona | `Matthew` | ✅ Yes |
| `POLLY_VOICE_JENNIFER` | Polly voice for Jennifer persona | `Joanna` | ✅ Yes |

### Lambda Configuration

| Variable Name | Description | Example | Required |
|---------------|-------------|---------|----------|
| `LAMBDA_FUNCTION_NAME` | Name of orchestration Lambda | `headset-agent-orchestrator` | ✅ Yes |
| `LAMBDA_MEMORY_MB` | Lambda memory allocation | `256` | ✅ Yes |
| `LAMBDA_TIMEOUT_SECONDS` | Lambda timeout | `30` | ✅ Yes |
| `LAMBDA_ARCHITECTURE` | Lambda CPU architecture | `arm64` | ✅ Yes |
| `GO_VERSION` | Go version for builds | `1.22` | ✅ Yes |

### Amazon Lex Configuration

| Variable Name | Description | Example | Required |
|---------------|-------------|---------|----------|
| `LEX_BOT_NAME` | Name of the Lex bot | `HeadsetTroubleshooterBot` | ✅ Yes |
| `LEX_BOT_ALIAS` | Lex bot alias for deployment | `prod` | ✅ Yes |
| `LEX_LOCALE` | Locale for Lex bot | `en_US` | ✅ Yes |

### CI/CD Configuration

| Variable Name | Description | Example | Required |
|---------------|-------------|---------|----------|
| `SAM_S3_BUCKET` | S3 bucket for SAM artifacts | `sam-artifacts-123456789012` | ✅ Yes |
| `STACK_NAME` | CloudFormation stack name | `headset-agent-stack` | ✅ Yes |
| `ENVIRONMENT` | Deployment environment | `dev`, `staging`, `prod` | ✅ Yes |
| `LOG_LEVEL` | Application log level | `DEBUG`, `INFO`, `WARN`, `ERROR` | ✅ Yes |
| `ENABLE_TRACING` | Enable X-Ray tracing | `true`, `false` | ✅ Yes |

---

## Setting Up Variables in GitHub

### Via GitHub CLI

gh secret set AWS_ACCESS_KEY_ID --body "AKIA..."
gh secret set AWS_SECRET_ACCESS_KEY --body "wJalr..."
gh secret set AWS_ACCOUNT_ID --body "123456789012"
gh secret set PINECONE_API_KEY --body "pcsk_..."

gh variable set AWS_REGION --body "us-east-1"
gh variable set ENVIRONMENT --body "dev"
gh variable set SUPERVISOR_AGENT_NAME --body "TroubleshootingOrchestrator"

---

## IAM Permissions Required

The AWS credentials must have permissions for:
- bedrock:* and bedrock-agent:* and bedrock-agentcore:*
- lambda:*
- connect:*
- lex:*
- s3:* (for specific buckets)
- dynamodb:* (for persona table)
- cloudformation:*
- iam:PassRole, iam:CreateRole, iam:AttachRolePolicy
- polly:SynthesizeSpeech
- secretsmanager:GetSecretValue
- logs:* and cloudwatch:*
```

---

## File 4: docs/regions.md

```markdown
# Regional Deployment Analysis
## Headset Troubleshooting Agent POC

**Last Updated:** December 31, 2025

---

## Executive Summary

### Verdict: ⚠️ PARTIAL DEPLOYMENT POSSIBLE IN us-east-2 WITH CRITICAL LIMITATIONS

**Primary Blocker:** Amazon Connect is **NOT AVAILABLE** in us-east-2

---

## Service Availability Matrix

| Service | us-east-1 (N. Virginia) | us-east-2 (Ohio) | Blocker? |
|---------|-------------------------|------------------|----------|
| **Amazon Connect** | ✅ Available | ❌ **NOT AVAILABLE** | 🚫 **CRITICAL BLOCKER** |
| **Amazon Lex V2** | ✅ Available | ✅ Available | ✅ No |
| **Amazon Bedrock** | ✅ Available | ✅ Available | ✅ No |
| **Bedrock Agents** | ✅ Available | ✅ Available | ✅ No |
| **Bedrock Multi-Agent** | ✅ Available | ✅ Available | ✅ No |
| **Bedrock AgentCore Runtime** | ✅ Available | ✅ Available | ✅ No |
| **AWS Lambda** | ✅ Available | ✅ Available | ✅ No |
| **Amazon Polly** | ✅ Available | ✅ Available | ✅ No |

---

## Amazon Connect Regional Availability

| Region Code | Region Name | Available |
|-------------|-------------|-----------|
| us-east-1 | US East (N. Virginia) | ✅ |
| us-west-2 | US West (Oregon) | ✅ |
| ca-central-1 | Canada (Central) | ✅ |
| eu-central-1 | Europe (Frankfurt) | ✅ |
| eu-west-2 | Europe (London) | ✅ |
| ap-northeast-1 | Asia Pacific (Tokyo) | ✅ |
| ap-northeast-2 | Asia Pacific (Seoul) | ✅ |
| ap-southeast-1 | Asia Pacific (Singapore) | ✅ |
| ap-southeast-2 | Asia Pacific (Sydney) | ✅ |
| af-south-1 | Africa (Cape Town) | ✅ |
| **us-east-2** | **US East (Ohio)** | ❌ |
| us-west-1 | US West (N. California) | ❌ |

---

## Recommendation

### For This POC

DEPLOY EVERYTHING IN us-east-1 (N. Virginia)

• All services available
• Lowest latency
• Lowest cost
• Fastest time to deployment

### Best Options for US-Based Deployment

| Region | Connect | Bedrock Agents | AgentCore | Recommendation |
|--------|---------|----------------|-----------|----------------|
| **us-east-1** | ✅ | ✅ | ✅ | **BEST CHOICE** |
| **us-west-2** | ✅ | ✅ | ✅ | Good alternative |

### Set AWS_REGION Variable

AWS_REGION: us-east-1
```

---

## File 5: docs/deployment-guide.md

```markdown
# GitHub Actions Deployment Guide
## Claude Code Autonomous CI/CD Pipeline

**Version:** 1.0  
**Date:** December 31, 2025  
**Status:** MANDATORY FOR ALL DEPLOYMENTS

---

## ⚠️ CRITICAL: Claude Code Deployment Autonomy

### Autonomous Deployment Policy

┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                              │
│   🤖 CLAUDE CODE HAS FULL AUTONOMY ON BUILDING AND DEPLOYING THIS PROJECT  │
│                                                                              │
│   ALL DEPLOYMENTS MUST GO THROUGH GITHUB ACTIONS - NO EXCEPTIONS            │
│                                                                              │
│   On pipeline failure, Claude Code will:                                    │
│   1. Automatically review the failure logs                                  │
│   2. Identify the root cause                                                │
│   3. Implement a fix                                                        │
│   4. Push the fix and re-trigger the pipeline                              │
│   5. Repeat until deployment is successful                                  │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

### Deployment Rules

| Rule | Requirement |
|------|-------------|
| **Local Deployment** | ❌ PROHIBITED |
| **Manual AWS Console** | ❌ PROHIBITED |
| **Direct AWS CLI** | ❌ PROHIBITED |
| **GitHub Actions** | ✅ MANDATORY |
| **Claude Code Autonomy** | ✅ FULL AUTHORITY |

---

## Repository Structure

headset-support-agent/
├── .github/
│   └── workflows/
│       ├── deploy.yml
│       ├── pr-validation.yml
│       ├── knowledge-base-sync.yml
│       └── rollback.yml
├── cmd/
│   └── lambda/
│       └── main.go
├── internal/
│   ├── agents/
│   ├── handlers/
│   ├── persona/
│   └── models/
├── knowledge-base/
│   ├── usb/
│   ├── bluetooth/
│   ├── windows/
│   └── genesys-cloud/
├── personas/
│   ├── tangerine.json
│   ├── joseph.json
│   └── jennifer.json
├── infrastructure/
│   ├── template.yaml
│   ├── bedrock-agents.yaml
│   └── connect-flow.json
├── scripts/
│   ├── create-agents.py
│   ├── sync-knowledge-base.sh
│   └── validate-deployment.sh
├── docs/
│   ├── variables.md
│   ├── regions.md
│   └── troubleshooting.md
├── go.mod
├── go.sum
├── Makefile
└── README.md

---

## Claude Code Failure Recovery Protocol

When a GitHub Actions workflow fails, Claude Code will automatically:

STEP 1: DETECT FAILURE
• Monitor GitHub Actions workflow status
• Identify which job failed
• Extract failure logs

STEP 2: ANALYZE ROOT CAUSE
• Parse error messages from logs
• Identify error category:
  - Build error (Go compilation)
  - Test failure (unit tests)
  - Deployment error (SAM/CloudFormation)
  - Configuration error (missing variables/secrets)
  - Infrastructure error (AWS service issues)

STEP 3: IMPLEMENT FIX
• Generate fix based on error analysis
• Create or modify affected files
• Update documentation if needed

STEP 4: PUSH AND VERIFY
• Commit fix with descriptive message
• Push to trigger new workflow run
• Monitor new workflow

STEP 5: ITERATE UNTIL SUCCESS
• If still failing, return to STEP 2
• Maximum iterations: 5 (then escalate to human)
• Log all attempts for audit trail

---

## Common Failure Patterns & Auto-Fixes

| Failure Type | Detection Pattern | Auto-Fix |
|--------------|-------------------|----------|
| **Go build error** | `cannot find package` | Add missing import |
| **Go build error** | `undefined:` | Add missing function/variable |
| **Test failure** | `FAIL:` | Fix test or update expectation |
| **SAM validation** | `Invalid template` | Correct YAML syntax |
| **CloudFormation** | `Resource already exists` | Add DeletionPolicy or rename |
| **CloudFormation** | `Access Denied` | Update IAM permissions |
| **Missing variable** | `variable not set` | Document required variable |
| **Region unavailable** | `not available in region` | Update region configuration |
```

---

## File 6: docs/CLAUDE.md

```markdown
# Headset Support Agent Project

## Project Overview
Voice-based headset troubleshooting agent using AWS multi-agent architecture with programmable personas.

## Key Documentation
- `docs/headset-agent-implementation-guide.md` - Architecture and implementation
- `docs/persona-troubleshooting-guide.md` - Personas (Tangerine, Joseph, Jennifer) and troubleshooting flows
- `docs/deployment-guide.md` - GitHub Actions CI/CD with Claude Code autonomy
- `docs/variables.md` - GitHub secrets and variables
- `docs/regions.md` - AWS region requirements (us-east-1 required for Connect)

## Critical Requirements
1. **Deploy via GitHub Actions ONLY** - No local deployments
2. **Claude Code has full autonomy** - Can build, deploy, and fix issues automatically
3. **Primary region: us-east-1** - Required for Amazon Connect
4. **Three personas**: Tangerine (Irish), Joseph (Ohio), Jennifer (Farm)

## Tech Stack
- AWS Lambda (Go)
- Amazon Bedrock Multi-Agent Collaboration
- Amazon Connect + Lex V2
- Amazon Polly (Neural/Generative voices)
- DynamoDB (Persona configs)
- S3 + Bedrock Knowledge Bases

## Quick Start
1. Configure GitHub secrets and variables per `docs/variables.md`
2. Push to main branch to trigger deployment
3. Claude Code will handle build, deploy, and any failure recovery
```

---

**END OF FILES**

After writing these files, confirm with: "✅ Created 6 files in docs/ directory"
