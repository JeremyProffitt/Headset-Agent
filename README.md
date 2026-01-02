# Headset Support Agent

Voice-based headset troubleshooting agent using AWS multi-agent architecture with programmable personas.

## Current Status (January 2, 2026)

| Component | Status | Notes |
|-----------|--------|-------|
| **Infrastructure** | Deployed | CloudFormation stack active |
| **Lambda Function** | Working | Responds to API calls |
| **Amazon Lex Bot** | Configured | DialogCodeHook enabled |
| **Amazon Connect** | Configured | Phone number active |
| **Nova Sonic** | Enabled | Speech-to-speech configured |
| **Bedrock Agents** | Created | Supervisor + 3 sub-agents |
| **Bedrock Invocation** | Not Working | Agent returns error response |
| **Knowledge Bases** | Not Integrated | KB not associated with agents |

### Known Issues

1. **Bedrock Agent Not Responding**: The Lambda function connects to Bedrock but receives errors. The API returns: `"I'm having a bit of trouble connecting. Let me try that again."`

2. **Multi-Agent Collaboration Not Configured**: The `create-agents.py` script creates agents but does not call `associate_agent_collaborator()` to link sub-agents to the supervisor.

3. **Knowledge Bases Not Associated**: Knowledge base documents are synced to S3 but not associated with Bedrock agents.

## Quick Links

- **Phone Number**: +16084663796
- **Web Chat**: http://headset-chat-759775734231-dev.s3-website-us-east-1.amazonaws.com
  - Password: `yesido`
- **API Endpoint**: https://cqmw180j1i.execute-api.us-east-1.amazonaws.com/dev/chat

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Amazon Connect │────▶│   Amazon Lex    │────▶│     Lambda      │
│  (Phone/Voice)  │     │   (Nova Sonic)  │     │  (Go Handler)   │
└─────────────────┘     └─────────────────┘     └────────┬────────┘
                                                         │
                                                         ▼
                                                ┌─────────────────┐
                                                │ Bedrock Agents  │
                                                │  (Supervisor)   │◀── NOT WORKING
                                                └────────┬────────┘
                                                         │
                        ┌────────────────────────────────┼────────────────────────────────┐
                        │                                │                                │
                        ▼                                ▼                                ▼
               ┌─────────────────┐              ┌─────────────────┐              ┌─────────────────┐
               │ DiagnosticAgent │              │  PlatformAgent  │              │ EscalationAgent │
               │   (Hardware)    │              │     (OS)        │              │    (Human)      │
               └─────────────────┘              └─────────────────┘              └─────────────────┘
```

## Personas

| Persona | Voice | Description |
|---------|-------|-------------|
| **Tangerine** | Amy (Nova Sonic) | Irish personality, upbeat and enthusiastic |
| **Joseph** | Matthew (Nova Sonic) | Ohio professional, technical focus |
| **Jennifer** | Tiffany (Nova Sonic) | Farm community, friendly and patient |

## Recent Fixes Applied

| Date | Fix | Commit |
|------|-----|--------|
| 2026-01-02 | Nova Sonic enabled via API | `f270c48` |
| 2026-01-02 | Nova Sonic IAM permissions | `d24b1cd` |
| 2026-01-02 | Empty response validation | `fffaccc` |
| 2026-01-02 | Timeout handling (25s) | `fffaccc` |
| 2026-01-02 | Validation fails on PLACEHOLDER | `79ac61d` |
| 2026-01-01 | DialogCodeHook enabled | `d4d6a4f` |
| 2026-01-01 | Sentiment analysis disabled | `9e11f0b` |

## Project Structure

```
headset-support-agent/
├── .github/workflows/      # CI/CD pipelines
│   └── deploy.yml          # Main deployment workflow
├── cmd/lambda/             # Lambda function (Go)
│   └── main.go
├── internal/
│   ├── agents/             # Bedrock client
│   ├── handlers/           # Lex response handlers
│   ├── persona/            # Persona loader
│   └── models/             # Data types
├── infrastructure/
│   ├── template.yaml       # SAM/CloudFormation
│   └── contact-flow.json   # Connect flow
├── scripts/
│   ├── create-agents.py    # Bedrock agent creation
│   ├── configure-nova-sonic.py  # Nova Sonic setup
│   └── validate-deployment.sh
├── personas/               # Persona configurations
├── knowledge-base/         # Troubleshooting docs
└── docs/                   # Documentation
```

## Deployment

All deployments go through GitHub Actions. See [docs/deployment-guide.md](docs/deployment-guide.md).

```bash
# Trigger deployment
git push origin main

# Monitor pipeline
gh run list
gh run watch <run-id>
```

## Next Steps to Fix Bedrock Agent

1. **Check SSM Parameters**:
   ```bash
   aws ssm get-parameters-by-path --path /headset-agent/dev/ --query 'Parameters[*].[Name,Value]'
   ```

2. **Verify Agent Exists**:
   ```bash
   aws bedrock-agent list-agents --query 'agentSummaries[?contains(agentName, `Orchestrator`)]'
   ```

3. **Check CloudWatch Logs**:
   ```bash
   aws logs filter-log-events --log-group-name /aws/lambda/headset-agent-orchestrator-dev --filter-pattern "ERROR"
   ```

4. **Associate Agent Collaborators** (missing step):
   - Update `scripts/create-agents.py` to call `associate_agent_collaborator()`

5. **Create and Associate Knowledge Bases** (missing step):
   - Create KB via `bedrock-agent create-knowledge-base`
   - Associate with agents via `associate-knowledge-base`

## Documentation

- [Implementation Guide](docs/headset-agent-implementation-guide.md)
- [Deployment Guide](docs/deployment-guide.md)
- [Persona Guide](docs/persona-troubleshooting-guide.md)
- [Variables Reference](docs/variables.md)
- [Regions Reference](docs/regions.md)

## Support

For issues, check:
1. GitHub Actions logs
2. CloudWatch logs: `/aws/lambda/headset-agent-orchestrator-dev`
3. SSM parameters: `/headset-agent/dev/*`
