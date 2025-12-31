# GitHub Repository Variables & Secrets
## Headset Troubleshooting Agent POC

**Last Updated:** December 31, 2025  
**Repository:** `your-org/headset-support-agent`

---

## Overview

This document defines all variables and secrets required for the GitHub Actions CI/CD pipeline. These must be configured in your repository settings under **Settings → Secrets and variables → Actions**.

> ⚠️ **IMPORTANT**: Claude Code has full autonomy to build and deploy this project. All deployments MUST go through GitHub Actions. Claude Code will automatically review any pipeline failures and deploy fixes until successful.

---

## Required Secrets

Secrets are encrypted and cannot be viewed after creation. They are used for sensitive credentials.

### AWS Credentials

| Secret Name | Description | Example/Format | Required |
|-------------|-------------|----------------|----------|
| `AWS_ACCESS_KEY_ID` | AWS IAM access key for deployment | `AKIA...` | ✅ Yes |
| `AWS_SECRET_ACCESS_KEY` | AWS IAM secret key for deployment | `wJalr...` | ✅ Yes |
| `AWS_ACCOUNT_ID` | 12-digit AWS account ID | `123456789012` | ✅ Yes |

### External Service Credentials

| Secret Name | Description | Example/Format | Required |
|-------------|-------------|----------------|----------|
| `PINECONE_API_KEY` | Pinecone vector database API key | `pcsk_...` | ✅ Yes (POC) |
| `PINECONE_ENVIRONMENT` | Pinecone environment/region | `us-east-1-aws` | ✅ Yes (POC) |

### Amazon Connect Secrets

| Secret Name | Description | Example/Format | Required |
|-------------|-------------|----------------|----------|
| `CONNECT_INSTANCE_ID` | Amazon Connect instance ID | `aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee` | ✅ Yes |
| `CONNECT_INSTANCE_ARN` | Full ARN of Connect instance | `arn:aws:connect:us-east-1:123456789012:instance/...` | ✅ Yes |

### Notification Secrets (Optional)

| Secret Name | Description | Example/Format | Required |
|-------------|-------------|----------------|----------|
| `SLACK_WEBHOOK_URL` | Slack webhook for deployment notifications | `https://hooks.slack.com/services/...` | ❌ Optional |
| `PAGERDUTY_ROUTING_KEY` | PagerDuty integration key for alerts | `R0...` | ❌ Optional |

---

## Required Variables

Variables are visible in logs and used for non-sensitive configuration.

### AWS Region Configuration

| Variable Name | Description | Default | Required |
|---------------|-------------|---------|----------|
| `AWS_REGION` | Primary AWS region for deployment | `us-east-1` | ✅ Yes |
| `AWS_SECONDARY_REGION` | Secondary region (if multi-region) | `us-west-2` | ❌ Optional |

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

### Environment Configuration

| Variable Name | Description | Values | Required |
|---------------|-------------|--------|----------|
| `ENVIRONMENT` | Deployment environment | `dev`, `staging`, `prod` | ✅ Yes |
| `LOG_LEVEL` | Application log level | `DEBUG`, `INFO`, `WARN`, `ERROR` | ✅ Yes |
| `ENABLE_TRACING` | Enable X-Ray tracing | `true`, `false` | ✅ Yes |

### CI/CD Configuration

| Variable Name | Description | Example | Required |
|---------------|-------------|---------|----------|
| `SAM_S3_BUCKET` | S3 bucket for SAM artifacts | `sam-artifacts-123456789012` | ✅ Yes |
| `STACK_NAME` | CloudFormation stack name | `headset-agent-stack` | ✅ Yes |
| `ENABLE_TERMINATION_PROTECTION` | Protect stack from deletion | `true` | ❌ Optional |

---

## Environment-Specific Variables

### Development (`dev`)

```yaml
ENVIRONMENT: dev
LOG_LEVEL: DEBUG
ENABLE_TRACING: true
BEDROCK_MODEL_SUPERVISOR: anthropic.claude-3-5-haiku-20241022-v1:0  # Use Haiku for cost savings
BEDROCK_MODEL_SUBAGENT: anthropic.claude-3-5-haiku-20241022-v1:0
LAMBDA_MEMORY_MB: 128
```

### Staging (`staging`)

```yaml
ENVIRONMENT: staging
LOG_LEVEL: INFO
ENABLE_TRACING: true
BEDROCK_MODEL_SUPERVISOR: anthropic.claude-3-5-sonnet-20241022-v2:0
BEDROCK_MODEL_SUBAGENT: anthropic.claude-3-5-haiku-20241022-v1:0
LAMBDA_MEMORY_MB: 256
```

### Production (`prod`)

```yaml
ENVIRONMENT: prod
LOG_LEVEL: WARN
ENABLE_TRACING: true
BEDROCK_MODEL_SUPERVISOR: anthropic.claude-3-5-sonnet-20241022-v2:0
BEDROCK_MODEL_SUBAGENT: anthropic.claude-3-5-haiku-20241022-v1:0
LAMBDA_MEMORY_MB: 512
ENABLE_TERMINATION_PROTECTION: true
```

---

## Setting Up Variables in GitHub

### Via GitHub UI

1. Navigate to your repository on GitHub
2. Click **Settings** → **Secrets and variables** → **Actions**
3. Under **Secrets**, click **New repository secret** for each secret
4. Under **Variables**, click **New repository variable** for each variable

### Via GitHub CLI

```bash
# Set secrets
gh secret set AWS_ACCESS_KEY_ID --body "AKIA..."
gh secret set AWS_SECRET_ACCESS_KEY --body "wJalr..."
gh secret set AWS_ACCOUNT_ID --body "123456789012"
gh secret set PINECONE_API_KEY --body "pcsk_..."

# Set variables
gh variable set AWS_REGION --body "us-east-1"
gh variable set ENVIRONMENT --body "dev"
gh variable set SUPERVISOR_AGENT_NAME --body "TroubleshootingOrchestrator"
# ... continue for all variables
```

---

## Variable Validation

The CI/CD pipeline includes a validation step that checks for required variables before deployment. Missing variables will cause the pipeline to fail early with a clear error message.

```yaml
# Example validation in workflow
- name: Validate Required Variables
  run: |
    REQUIRED_VARS=(
      "AWS_REGION"
      "SUPERVISOR_AGENT_NAME"
      "KB_S3_BUCKET_NAME"
      "LAMBDA_FUNCTION_NAME"
    )
    
    for var in "${REQUIRED_VARS[@]}"; do
      if [ -z "${!var}" ]; then
        echo "❌ Missing required variable: $var"
        exit 1
      fi
    done
    
    echo "✅ All required variables are set"
```

---

## IAM Permissions Required

The AWS credentials must have the following permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "BedrockAccess",
      "Effect": "Allow",
      "Action": [
        "bedrock:*",
        "bedrock-agent:*",
        "bedrock-agentcore:*"
      ],
      "Resource": "*"
    },
    {
      "Sid": "LambdaAccess",
      "Effect": "Allow",
      "Action": [
        "lambda:*"
      ],
      "Resource": "*"
    },
    {
      "Sid": "ConnectAccess",
      "Effect": "Allow",
      "Action": [
        "connect:*"
      ],
      "Resource": "*"
    },
    {
      "Sid": "LexAccess",
      "Effect": "Allow",
      "Action": [
        "lex:*"
      ],
      "Resource": "*"
    },
    {
      "Sid": "S3Access",
      "Effect": "Allow",
      "Action": [
        "s3:*"
      ],
      "Resource": [
        "arn:aws:s3:::${KB_S3_BUCKET_NAME}",
        "arn:aws:s3:::${KB_S3_BUCKET_NAME}/*",
        "arn:aws:s3:::${SAM_S3_BUCKET}",
        "arn:aws:s3:::${SAM_S3_BUCKET}/*"
      ]
    },
    {
      "Sid": "DynamoDBAccess",
      "Effect": "Allow",
      "Action": [
        "dynamodb:*"
      ],
      "Resource": "arn:aws:dynamodb:*:*:table/${PERSONA_TABLE_NAME}"
    },
    {
      "Sid": "CloudFormationAccess",
      "Effect": "Allow",
      "Action": [
        "cloudformation:*"
      ],
      "Resource": "*"
    },
    {
      "Sid": "IAMPassRole",
      "Effect": "Allow",
      "Action": [
        "iam:PassRole",
        "iam:CreateRole",
        "iam:AttachRolePolicy",
        "iam:GetRole"
      ],
      "Resource": "*"
    },
    {
      "Sid": "PollyAccess",
      "Effect": "Allow",
      "Action": [
        "polly:SynthesizeSpeech"
      ],
      "Resource": "*"
    },
    {
      "Sid": "SecretsManagerAccess",
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue"
      ],
      "Resource": "*"
    },
    {
      "Sid": "CloudWatchAccess",
      "Effect": "Allow",
      "Action": [
        "logs:*",
        "cloudwatch:*"
      ],
      "Resource": "*"
    }
  ]
}
```

---

## Security Recommendations

1. **Rotate Credentials Regularly**: AWS access keys should be rotated every 90 days
2. **Use OIDC**: Consider using GitHub OIDC for AWS authentication instead of long-lived credentials
3. **Least Privilege**: Create deployment-specific IAM roles with minimal required permissions
4. **Secret Scanning**: Enable GitHub secret scanning to prevent accidental commits
5. **Environment Protection**: Use GitHub environment protection rules for production deployments

---

## Troubleshooting

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| `InvalidIdentityToken` | Expired AWS credentials | Regenerate and update secrets |
| `AccessDenied` | Missing IAM permissions | Review and update IAM policy |
| `ResourceNotFoundException` | Resource doesn't exist in region | Verify AWS_REGION is correct |
| `ValidationException` | Invalid variable format | Check variable format against examples |

---

## Changelog

| Date | Change | Author |
|------|--------|--------|
| 2025-12-31 | Initial creation | Claude Code |
