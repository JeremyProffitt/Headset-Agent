# Headset Support Agent Project

## Workflow Requirements
1. Always commit and push changes after making fixes
2. Review the GitHub Actions pipeline for any issues
3. Remediate failures autonomously and re-push until pipeline succeeds

## Project Overview
Voice-based headset troubleshooting agent using AWS multi-agent architecture with programmable personas.

## Current Status (January 2, 2026)

### Working Components
- Infrastructure deployed (CloudFormation stack)
- Lambda function responding
- Amazon Lex bot configured with DialogCodeHook
- Amazon Connect with phone number (+16084663796)
- Nova Sonic enabled for speech-to-speech
- Bedrock agents created (supervisor + 3 sub-agents)
- SSM parameters updated (no longer PLACEHOLDER)

### Not Working
- **Bedrock Agent Invocation**: Returns "I'm having a bit of trouble connecting"
- **Multi-Agent Collaboration**: Sub-agents not linked to supervisor
- **Knowledge Bases**: Not created or associated with agents

## Priority Fixes Needed

### 1. Fix Bedrock Agent Invocation
The Lambda calls `bedrockagentruntime:InvokeAgent` but receives errors.

**Check CloudWatch logs for**:
- `/aws/lambda/headset-agent-orchestrator-dev`
- Filter: "Error invoking Bedrock agent"

**Possible causes**:
- Agent not in PREPARED state
- Agent alias not pointing to correct version
- Missing IAM permissions
- Agent instructions causing errors

### 2. Add Agent Collaborator Associations
Update `scripts/create-agents.py` to call `associate_agent_collaborator()`:

```python
bedrock_agent.associate_agent_collaborator(
    agentId=supervisor_agent_id,
    agentVersion='DRAFT',
    agentDescriptor={
        'aliasArn': f'arn:aws:bedrock:{region}:{account_id}:agent-alias/{sub_agent_id}/{alias_id}'
    },
    collaboratorName='DiagnosticAgent',
    collaborationInstruction='Route hardware diagnostics to this agent',
    relayConversationHistory='TO_COLLABORATOR'
)
```

### 3. Create and Associate Knowledge Bases
Add to `scripts/create-agents.py`:
1. Create knowledge base with S3 data source
2. Start ingestion job
3. Associate KB with agents

## Key Documentation
- docs/headset-agent-implementation-guide.md - Architecture and implementation
- docs/persona-troubleshooting-guide.md - Personas and troubleshooting flows
- docs/deployment-guide.md - GitHub Actions CI/CD with Claude Code autonomy
- docs/variables.md - GitHub secrets and variables
- docs/regions.md - AWS region requirements

## Critical Requirements
1. Deploy via GitHub Actions ONLY - No local deployments
2. Claude Code has full autonomy for builds and fixes
3. Primary region: us-east-1 (required for Amazon Connect)
4. Three personas: Tangerine (Irish), Joseph (Ohio), Jennifer (Farm)

## Key Files

| File | Purpose |
|------|---------|
| `cmd/lambda/main.go` | Lambda handler (Lex + API) |
| `internal/agents/bedrock.go` | Bedrock agent client |
| `internal/handlers/lex.go` | Lex response builders |
| `scripts/create-agents.py` | Bedrock agent creation |
| `scripts/configure-nova-sonic.py` | Nova Sonic setup |
| `infrastructure/template.yaml` | SAM/CloudFormation |

## SSM Parameters

| Parameter | Current Value |
|-----------|---------------|
| `/headset-agent/dev/supervisor-agent-id` | UOAPORBDLT |
| `/headset-agent/dev/supervisor-agent-alias` | XKSDIPKVWN |
| `/headset-agent/dev/nova-sonic-status` | enabled |

## Recent Commits
- `f270c48` - Nova Sonic enabled via API
- `d24b1cd` - Nova Sonic IAM permissions
- `fffaccc` - Empty response validation + timeout handling
- `79ac61d` - Validation fails on PLACEHOLDER
- `d4d6a4f` - DialogCodeHook enabled
