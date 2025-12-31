# Regional Deployment Analysis
## Headset Troubleshooting Agent POC

**Last Updated:** December 31, 2025  
**Assessment Version:** 1.0

---

## Executive Summary

This document analyzes the feasibility of deploying the Headset Troubleshooting Agent in **US East 2 (Ohio / us-east-2)** and identifies potential blockers, limitations, and workarounds.

### Verdict: ⚠️ PARTIAL DEPLOYMENT POSSIBLE WITH CRITICAL LIMITATIONS

**Primary Blocker:** Amazon Connect is **NOT AVAILABLE** in us-east-2

---

## Service Availability Matrix

### Core Services Required

| Service | us-east-1 (N. Virginia) | us-east-2 (Ohio) | Blocker? |
|---------|-------------------------|------------------|----------|
| **Amazon Connect** | ✅ Available | ❌ **NOT AVAILABLE** | 🚫 **CRITICAL BLOCKER** |
| **Amazon Lex V2** | ✅ Available | ✅ Available | ✅ No |
| **Amazon Bedrock** | ✅ Available | ✅ Available | ✅ No |
| **Bedrock Agents** | ✅ Available | ✅ Available | ✅ No |
| **Bedrock Multi-Agent** | ✅ Available | ✅ Available | ✅ No |
| **Bedrock AgentCore Runtime** | ✅ Available | ✅ Available | ✅ No |
| **Bedrock Knowledge Bases** | ✅ Available | ✅ Available | ✅ No |
| **AWS Lambda** | ✅ Available | ✅ Available | ✅ No |
| **API Gateway** | ✅ Available | ✅ Available | ✅ No |
| **DynamoDB** | ✅ Available | ✅ Available | ✅ No |
| **Amazon S3** | ✅ Available | ✅ Available | ✅ No |
| **Amazon Polly** | ✅ Available | ✅ Available | ✅ No |
| **CloudWatch** | ✅ Available | ✅ Available | ✅ No |
| **Secrets Manager** | ✅ Available | ✅ Available | ✅ No |

### Bedrock Model Availability

| Model | us-east-1 | us-east-2 | Notes |
|-------|-----------|-----------|-------|
| Claude 3.5 Sonnet v2 | ✅ | ✅ | Full support |
| Claude 3.5 Haiku | ✅ | ✅ | Full support |
| Amazon Titan Embeddings | ✅ | ✅ | For knowledge bases |
| Cross-Region Inference | ✅ Source | ✅ Source | Both can route to other regions |

---

## Critical Blocker: Amazon Connect

### The Problem

**Amazon Connect is NOT available in us-east-2 (Ohio).**

Amazon Connect is available in the following US regions only:
- ✅ **us-east-1** (N. Virginia)
- ✅ **us-west-2** (Oregon)
- ❌ us-east-2 (Ohio) - **NOT SUPPORTED**
- ❌ us-west-1 (N. California) - NOT SUPPORTED

### Impact on Architecture

```
DESIRED ARCHITECTURE (us-east-2):
┌─────────────────────────────────────────────────────────────┐
│                     ❌ BLOCKED                               │
│                                                              │
│  Phone Call → Amazon Connect → Lex → Bedrock → Response     │
│                    ↑                                         │
│                    │                                         │
│             NOT AVAILABLE                                    │
│              IN us-east-2                                    │
└─────────────────────────────────────────────────────────────┘
```

### Why This Matters

Amazon Connect is the **ONLY** fully-managed AWS service for:
- Inbound phone call handling
- IVR (Interactive Voice Response)
- Native integration with Amazon Lex for speech recognition
- Real-time voice streaming to AI/ML services
- Call recording and analytics
- Agent routing and queue management

Without Connect, we cannot receive phone calls in the AWS ecosystem.

---

## Deployment Options

### Option 1: Deploy Everything in us-east-1 (RECOMMENDED)

**Recommendation:** Deploy the entire solution in us-east-1 (N. Virginia)

```
┌─────────────────────────────────────────────────────────────┐
│                    us-east-1 (N. Virginia)                   │
│                                                              │
│  Phone → Connect → Lex → Lambda → Bedrock Agents → Polly    │
│            │                          │                      │
│            └─── DynamoDB ←───────────┘                      │
│                    │                                         │
│                    └──── S3 (Knowledge Base)                │
│                                                              │
│  ✅ All services available                                  │
│  ✅ Lowest latency (single region)                          │
│  ✅ Simplest architecture                                   │
└─────────────────────────────────────────────────────────────┘
```

**Pros:**
- All services available
- Lowest latency
- Simplest deployment
- Single point of management

**Cons:**
- May not meet data residency requirements if Ohio specifically required
- Single region (no geographic redundancy)

---

### Option 2: Hybrid Architecture (Connect in us-east-1, Backend in us-east-2)

If there's a specific requirement for compute/data to reside in us-east-2:

```
┌─────────────────────────────────────────────────────────────┐
│                    us-east-1 (N. Virginia)                   │
│                                                              │
│  Phone → Amazon Connect ──┐                                 │
│                           │                                  │
└───────────────────────────│──────────────────────────────────┘
                            │ (~12ms latency)
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    us-east-2 (Ohio)                          │
│                                                              │
│  ┌──────────┐    ┌────────────┐    ┌──────────────────┐    │
│  │ Lex Bot  │ →  │  Lambda    │ →  │  Bedrock Agents  │    │
│  └──────────┘    └────────────┘    └──────────────────┘    │
│        │                                    │               │
│        └─────── DynamoDB ←─────────────────┘               │
│                     │                                       │
│                     └──── S3 (Knowledge Base)               │
└─────────────────────────────────────────────────────────────┘
```

**Technical Implementation:**

1. **Amazon Connect** stays in us-east-1
2. Configure Lex bot integration to point to us-east-2 Lambda
3. All backend services (Bedrock, DynamoDB, S3) in us-east-2

**Cross-Region Lex Integration:**
```yaml
# Connect contact flow - invoke Lex in different region
ContactFlow:
  GetCustomerInput:
    Type: GetCustomerInput
    Parameters:
      LexBot:
        Name: HeadsetTroubleshooterBot
        Region: us-east-2  # Cross-region Lex
        Alias: prod
```

**Pros:**
- Data processing in us-east-2 if required
- Connect available for telephony

**Cons:**
- Added latency (~12ms between regions)
- More complex architecture
- Cross-region data transfer costs
- More failure points

---

### Option 3: Alternative Telephony Provider (NOT RECOMMENDED for POC)

Use a third-party telephony provider that can operate in us-east-2:

- **Twilio** + Amazon Lex
- **Vonage** + Amazon Lex
- **Bandwidth** + Amazon Lex

```
┌─────────────────────────────────────────────────────────────┐
│                    us-east-2 (Ohio)                          │
│                                                              │
│  Phone → Twilio → API Gateway → Lambda → Lex → Bedrock     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**Cons:**
- Additional vendor relationship
- Not fully managed by AWS
- Additional integration complexity
- Additional costs
- Loss of native Connect features

**NOT RECOMMENDED for POC** - adds unnecessary complexity.

---

## Latency Analysis

### Cross-Region Latency (us-east-1 ↔ us-east-2)

| Metric | Value |
|--------|-------|
| Round-trip latency | ~12ms |
| Data transfer cost | $0.01/GB |

### Impact on Voice Quality

For real-time voice applications:
- **Acceptable latency:** < 150ms total
- **Noticeable latency:** 150-300ms
- **Poor quality:** > 300ms

**Assessment:** 12ms cross-region latency is **negligible** and will not impact voice quality.

---

## Cost Implications

### Option 1: Single Region (us-east-1)

| Component | Monthly Cost |
|-----------|--------------|
| No cross-region transfer | $0 |
| Single deployment | Baseline |
| **Total Additional** | **$0** |

### Option 2: Hybrid (Connect us-east-1, Backend us-east-2)

| Component | Monthly Cost (10K minutes) |
|-----------|---------------------------|
| Cross-region data transfer | ~$5-10 |
| Additional CloudWatch logs | ~$2-5 |
| Multi-region management | Operational overhead |
| **Total Additional** | **~$7-15/month** |

---

## Compliance & Data Residency

### If Ohio Data Residency is Required

Some organizations may require data to physically reside in Ohio for compliance reasons. In this case:

**What CAN be in us-east-2:**
- ✅ Customer data storage (DynamoDB, S3)
- ✅ Knowledge bases
- ✅ Bedrock agent processing
- ✅ Conversation transcripts
- ✅ Lambda functions
- ✅ Persona configurations

**What CANNOT be in us-east-2:**
- ❌ Phone call reception (Connect)
- ❌ Initial voice processing (Connect)

**Mitigation:** 
- Configure Connect to NOT store call recordings in us-east-1
- Stream transcripts to us-east-2 immediately
- Store all persistent data in us-east-2

---

## Recommendations

### For POC Phase

| Scenario | Recommendation |
|----------|----------------|
| No specific region requirement | Deploy entirely in **us-east-1** |
| Preference for Ohio | Deploy entirely in **us-east-1**, migrate backend later |
| Hard requirement for Ohio data | Use **Hybrid architecture** (Option 2) |

### For Production Phase

| Scenario | Recommendation |
|----------|----------------|
| Standard deployment | **us-east-1** single region |
| High availability required | **us-east-1** primary + **us-west-2** DR |
| Ohio data residency required | Hybrid with Connect in us-east-1, backend in us-east-2 |

---

## Migration Path: us-east-1 → Hybrid

If you start in us-east-1 and later need Ohio data residency:

### Phase 1: Deploy in us-east-1 (POC)
All services in us-east-1 for simplicity.

### Phase 2: Migrate Backend to us-east-2 (If Required)
1. Create new DynamoDB table in us-east-2
2. Replicate S3 bucket to us-east-2
3. Deploy Lambda functions in us-east-2
4. Create Bedrock agents in us-east-2
5. Update Connect/Lex to point to us-east-2 resources
6. Migrate Lex bot to us-east-2
7. Test thoroughly
8. Cutover

### Estimated Migration Effort
- **Duration:** 2-4 weeks
- **Risk:** Medium
- **Downtime:** Minimal (with blue-green deployment)

---

## Service Availability Summary

### Available in us-east-2

✅ Amazon Bedrock (all models)  
✅ Amazon Bedrock Agents  
✅ Amazon Bedrock Multi-Agent Collaboration  
✅ Amazon Bedrock AgentCore Runtime  
✅ Amazon Bedrock Knowledge Bases  
✅ Amazon Lex V2  
✅ AWS Lambda  
✅ Amazon API Gateway  
✅ Amazon DynamoDB  
✅ Amazon S3  
✅ Amazon Polly  
✅ Amazon CloudWatch  
✅ AWS Secrets Manager  
✅ AWS CloudFormation  
✅ AWS SAM  

### NOT Available in us-east-2

❌ **Amazon Connect** - CRITICAL BLOCKER  
❌ Amazon Connect Voice ID  
❌ Amazon Connect Contact Lens  

---

## Alternative Regions Analysis

| Region | Connect | Bedrock | AgentCore | Recommendation |
|--------|---------|---------|-----------|----------------|
| **us-east-1** | ✅ | ✅ | ✅ | **RECOMMENDED** |
| us-east-2 | ❌ | ✅ | ✅ | Backend only |
| **us-west-2** | ✅ | ✅ | ✅ | **Good alternative** |
| us-west-1 | ❌ | ✅ | ❌ | Not recommended |
| ca-central-1 | ✅ | ✅ | ❌ | Canada only |
| eu-central-1 | ✅ | ✅ | ✅ | EU deployment |
| eu-west-2 | ✅ | ✅ | ❌ | UK deployment |
| ap-southeast-2 | ✅ | ✅ | ✅ | APAC deployment |

---

## Final Recommendation

### For This POC

```
┌─────────────────────────────────────────────────────────────┐
│                                                              │
│   DEPLOY EVERYTHING IN us-east-1 (N. Virginia)              │
│                                                              │
│   • Simplest architecture                                   │
│   • All services available                                  │
│   • Lowest latency                                          │
│   • Lowest cost                                             │
│   • Fastest time to deployment                              │
│                                                              │
│   Migration to hybrid (if needed) can happen post-POC       │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Set AWS_REGION Variable

```yaml
# In GitHub repository variables
AWS_REGION: us-east-1
```

---

## Appendix: Amazon Connect Regional Availability

As of December 2025, Amazon Connect is available in:

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

## Document History

| Date | Version | Change | Author |
|------|---------|--------|--------|
| 2025-12-31 | 1.0 | Initial analysis | Claude Code |
