# GitHub Actions Deployment Guide
## Claude Code Autonomous CI/CD Pipeline

**Version:** 1.0  
**Date:** December 31, 2025  
**Status:** MANDATORY FOR ALL DEPLOYMENTS

---

## ⚠️ CRITICAL: Claude Code Deployment Autonomy

### Autonomous Deployment Policy

```
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
```

### Deployment Rules

| Rule | Requirement |
|------|-------------|
| **Local Deployment** | ❌ PROHIBITED |
| **Manual AWS Console** | ❌ PROHIBITED |
| **Direct AWS CLI** | ❌ PROHIBITED |
| **GitHub Actions** | ✅ MANDATORY |
| **Claude Code Autonomy** | ✅ FULL AUTHORITY |

### Claude Code Responsibilities

1. **Build Automation**: Claude Code constructs all build configurations
2. **Pipeline Creation**: Claude Code creates and maintains all GitHub Actions workflows
3. **Failure Recovery**: Claude Code automatically fixes failed deployments
4. **Continuous Improvement**: Claude Code optimizes pipelines based on metrics
5. **Documentation**: Claude Code maintains deployment documentation

---

## Repository Structure

```
headset-support-agent/
├── .github/
│   └── workflows/
│       ├── deploy.yml              # Main deployment workflow
│       ├── pr-validation.yml       # PR checks
│       ├── knowledge-base-sync.yml # KB document sync
│       └── rollback.yml            # Emergency rollback
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
│   ├── template.yaml              # SAM template
│   ├── bedrock-agents.yaml        # Agent definitions
│   └── connect-flow.json          # Connect contact flow
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
```

---

## Main Deployment Workflow

### `.github/workflows/deploy.yml`

```yaml
name: Deploy Headset Support Agent

on:
  push:
    branches:
      - main
      - 'release/*'
  pull_request:
    branches:
      - main
  workflow_dispatch:
    inputs:
      environment:
        description: 'Target environment'
        required: true
        default: 'dev'
        type: choice
        options:
          - dev
          - staging
          - prod
      force_deploy:
        description: 'Force deployment even if no changes'
        required: false
        default: false
        type: boolean

env:
  AWS_REGION: ${{ vars.AWS_REGION }}
  GO_VERSION: '1.22'
  SAM_CLI_TELEMETRY: 0

permissions:
  id-token: write
  contents: read
  actions: read
  checks: write

jobs:
  # ============================================================
  # JOB 1: Validate Configuration
  # ============================================================
  validate:
    name: Validate Configuration
    runs-on: ubuntu-latest
    outputs:
      environment: ${{ steps.set-env.outputs.environment }}
    
    steps:
      - name: Checkout Code
        uses: actions/checkout@v4
      
      - name: Validate Required Variables
        run: |
          echo "🔍 Validating required variables..."
          
          REQUIRED_VARS=(
            "AWS_REGION"
            "SUPERVISOR_AGENT_NAME"
            "KB_S3_BUCKET_NAME"
            "LAMBDA_FUNCTION_NAME"
            "PERSONA_TABLE_NAME"
            "LEX_BOT_NAME"
            "STACK_NAME"
            "SAM_S3_BUCKET"
          )
          
          MISSING_VARS=()
          
          for var in "${REQUIRED_VARS[@]}"; do
            if [ -z "${{ vars[var] }}" ]; then
              MISSING_VARS+=("$var")
            fi
          done
          
          if [ ${#MISSING_VARS[@]} -gt 0 ]; then
            echo "❌ Missing required variables:"
            printf '   - %s\n' "${MISSING_VARS[@]}"
            echo ""
            echo "📖 See docs/variables.md for setup instructions"
            exit 1
          fi
          
          echo "✅ All required variables are configured"
      
      - name: Validate Required Secrets
        run: |
          echo "🔐 Validating required secrets..."
          
          if [ -z "${{ secrets.AWS_ACCESS_KEY_ID }}" ]; then
            echo "❌ Missing AWS_ACCESS_KEY_ID secret"
            exit 1
          fi
          
          if [ -z "${{ secrets.AWS_SECRET_ACCESS_KEY }}" ]; then
            echo "❌ Missing AWS_SECRET_ACCESS_KEY secret"
            exit 1
          fi
          
          echo "✅ All required secrets are configured"
      
      - name: Determine Environment
        id: set-env
        run: |
          if [ "${{ github.event_name }}" == "workflow_dispatch" ]; then
            echo "environment=${{ inputs.environment }}" >> $GITHUB_OUTPUT
          elif [ "${{ github.ref }}" == "refs/heads/main" ]; then
            echo "environment=staging" >> $GITHUB_OUTPUT
          elif [[ "${{ github.ref }}" == refs/heads/release/* ]]; then
            echo "environment=prod" >> $GITHUB_OUTPUT
          else
            echo "environment=dev" >> $GITHUB_OUTPUT
          fi

  # ============================================================
  # JOB 2: Build Lambda Function
  # ============================================================
  build:
    name: Build Lambda Function
    runs-on: ubuntu-latest
    needs: validate
    
    steps:
      - name: Checkout Code
        uses: actions/checkout@v4
      
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
      
      - name: Run Tests
        run: |
          echo "🧪 Running unit tests..."
          go test -v -race -coverprofile=coverage.out ./...
          
          echo "📊 Test coverage:"
          go tool cover -func=coverage.out
      
      - name: Build Lambda Binary
        run: |
          echo "🔨 Building Lambda binary for arm64..."
          
          GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build \
            -ldflags="-s -w" \
            -o bootstrap \
            ./cmd/lambda/
          
          echo "✅ Binary built successfully"
          ls -la bootstrap
      
      - name: Upload Build Artifact
        uses: actions/upload-artifact@v4
        with:
          name: lambda-binary
          path: bootstrap
          retention-days: 1

  # ============================================================
  # JOB 3: Deploy Infrastructure
  # ============================================================
  deploy:
    name: Deploy to ${{ needs.validate.outputs.environment }}
    runs-on: ubuntu-latest
    needs: [validate, build]
    environment: ${{ needs.validate.outputs.environment }}
    
    steps:
      - name: Checkout Code
        uses: actions/checkout@v4
      
      - name: Download Build Artifact
        uses: actions/download-artifact@v4
        with:
          name: lambda-binary
          path: ./cmd/lambda/
      
      - name: Configure AWS Credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: ${{ vars.AWS_REGION }}
      
      - name: Setup SAM CLI
        uses: aws-actions/setup-sam@v2
        with:
          use-installer: true
      
      - name: SAM Build
        run: |
          echo "📦 Building SAM application..."
          sam build --use-container
      
      - name: SAM Deploy
        id: sam-deploy
        run: |
          echo "🚀 Deploying to ${{ needs.validate.outputs.environment }}..."
          
          sam deploy \
            --stack-name ${{ vars.STACK_NAME }}-${{ needs.validate.outputs.environment }} \
            --s3-bucket ${{ vars.SAM_S3_BUCKET }} \
            --s3-prefix ${{ vars.STACK_NAME }} \
            --region ${{ vars.AWS_REGION }} \
            --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM \
            --no-confirm-changeset \
            --no-fail-on-empty-changeset \
            --parameter-overrides \
              Environment=${{ needs.validate.outputs.environment }} \
              SupervisorAgentName=${{ vars.SUPERVISOR_AGENT_NAME }} \
              DiagnosticAgentName=${{ vars.DIAGNOSTIC_AGENT_NAME }} \
              PlatformAgentName=${{ vars.PLATFORM_AGENT_NAME }} \
              EscalationAgentName=${{ vars.ESCALATION_AGENT_NAME }} \
              KBBucketName=${{ vars.KB_S3_BUCKET_NAME }} \
              PersonaTableName=${{ vars.PERSONA_TABLE_NAME }} \
              LexBotName=${{ vars.LEX_BOT_NAME }} \
              BedrockModelSupervisor=${{ vars.BEDROCK_MODEL_SUPERVISOR }} \
              BedrockModelSubagent=${{ vars.BEDROCK_MODEL_SUBAGENT }}
          
          echo "✅ SAM deployment complete"
      
      - name: Deploy Bedrock Agents
        run: |
          echo "🤖 Deploying Bedrock agents..."
          
          pip install boto3
          python scripts/create-agents.py \
            --environment ${{ needs.validate.outputs.environment }} \
            --region ${{ vars.AWS_REGION }}
          
          echo "✅ Bedrock agents deployed"
      
      - name: Sync Knowledge Base
        run: |
          echo "📚 Syncing knowledge base documents..."
          
          aws s3 sync knowledge-base/ s3://${{ vars.KB_S3_BUCKET_NAME }}/ \
            --delete \
            --exclude ".git/*"
          
          echo "✅ Knowledge base synced"
      
      - name: Deploy Persona Configurations
        run: |
          echo "👤 Deploying persona configurations..."
          
          for persona_file in personas/*.json; do
            persona_id=$(basename "$persona_file" .json)
            echo "  Deploying persona: $persona_id"
            
            aws dynamodb put-item \
              --table-name ${{ vars.PERSONA_TABLE_NAME }} \
              --item file://"$persona_file"
          done
          
          echo "✅ Personas deployed"
      
      - name: Capture Deployment Outputs
        id: outputs
        run: |
          STACK_NAME="${{ vars.STACK_NAME }}-${{ needs.validate.outputs.environment }}"
          
          LAMBDA_ARN=$(aws cloudformation describe-stacks \
            --stack-name "$STACK_NAME" \
            --query "Stacks[0].Outputs[?OutputKey=='LambdaFunctionArn'].OutputValue" \
            --output text)
          
          echo "lambda_arn=$LAMBDA_ARN" >> $GITHUB_OUTPUT
          echo "📋 Lambda ARN: $LAMBDA_ARN"

  # ============================================================
  # JOB 4: Validate Deployment
  # ============================================================
  validate-deployment:
    name: Validate Deployment
    runs-on: ubuntu-latest
    needs: [validate, deploy]
    
    steps:
      - name: Checkout Code
        uses: actions/checkout@v4
      
      - name: Configure AWS Credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: ${{ vars.AWS_REGION }}
      
      - name: Test Lambda Function
        run: |
          echo "🧪 Testing Lambda function..."
          
          RESPONSE=$(aws lambda invoke \
            --function-name ${{ vars.LAMBDA_FUNCTION_NAME }}-${{ needs.validate.outputs.environment }} \
            --payload '{"test": true}' \
            --cli-binary-format raw-in-base64-out \
            response.json)
          
          STATUS_CODE=$(echo $RESPONSE | jq -r '.StatusCode')
          
          if [ "$STATUS_CODE" != "200" ]; then
            echo "❌ Lambda invocation failed with status: $STATUS_CODE"
            cat response.json
            exit 1
          fi
          
          echo "✅ Lambda function responding correctly"
      
      - name: Test Bedrock Agent
        run: |
          echo "🤖 Testing Bedrock supervisor agent..."
          
          # Get agent ID from parameter store or stack outputs
          AGENT_ID=$(aws ssm get-parameter \
            --name "/headset-agent/${{ needs.validate.outputs.environment }}/supervisor-agent-id" \
            --query "Parameter.Value" \
            --output text 2>/dev/null || echo "")
          
          if [ -z "$AGENT_ID" ]; then
            echo "⚠️ Agent ID not found, skipping agent test"
            exit 0
          fi
          
          echo "✅ Bedrock agent configured: $AGENT_ID"
      
      - name: Health Check Summary
        run: |
          echo ""
          echo "═══════════════════════════════════════════════"
          echo "    DEPLOYMENT VALIDATION COMPLETE"
          echo "═══════════════════════════════════════════════"
          echo ""
          echo "Environment: ${{ needs.validate.outputs.environment }}"
          echo "Region: ${{ vars.AWS_REGION }}"
          echo "Stack: ${{ vars.STACK_NAME }}-${{ needs.validate.outputs.environment }}"
          echo ""
          echo "✅ All validation checks passed"
          echo ""

  # ============================================================
  # JOB 5: Notification
  # ============================================================
  notify:
    name: Send Notifications
    runs-on: ubuntu-latest
    needs: [validate, deploy, validate-deployment]
    if: always()
    
    steps:
      - name: Notify Success
        if: ${{ needs.deploy.result == 'success' }}
        run: |
          echo "🎉 Deployment successful!"
          
          # Slack notification (if configured)
          if [ -n "${{ secrets.SLACK_WEBHOOK_URL }}" ]; then
            curl -X POST ${{ secrets.SLACK_WEBHOOK_URL }} \
              -H 'Content-type: application/json' \
              --data '{
                "text": "✅ Headset Agent deployed to ${{ needs.validate.outputs.environment }}",
                "blocks": [
                  {
                    "type": "section",
                    "text": {
                      "type": "mrkdwn",
                      "text": "*Deployment Successful* ✅\n\n*Environment:* ${{ needs.validate.outputs.environment }}\n*Branch:* ${{ github.ref_name }}\n*Commit:* ${{ github.sha }}"
                    }
                  }
                ]
              }'
          fi
      
      - name: Notify Failure
        if: ${{ needs.deploy.result == 'failure' || needs.validate-deployment.result == 'failure' }}
        run: |
          echo "❌ Deployment failed!"
          echo ""
          echo "┌─────────────────────────────────────────────────────────────────┐"
          echo "│  🤖 CLAUDE CODE AUTONOMOUS RECOVERY WILL BE TRIGGERED          │"
          echo "│                                                                  │"
          echo "│  Claude Code will:                                              │"
          echo "│  1. Review failure logs                                         │"
          echo "│  2. Identify root cause                                         │"
          echo "│  3. Implement fix                                               │"
          echo "│  4. Push fix and re-trigger pipeline                           │"
          echo "│  5. Repeat until successful                                     │"
          echo "└─────────────────────────────────────────────────────────────────┘"
          
          # Slack notification (if configured)
          if [ -n "${{ secrets.SLACK_WEBHOOK_URL }}" ]; then
            curl -X POST ${{ secrets.SLACK_WEBHOOK_URL }} \
              -H 'Content-type: application/json' \
              --data '{
                "text": "❌ Headset Agent deployment FAILED - Claude Code recovery initiated",
                "blocks": [
                  {
                    "type": "section",
                    "text": {
                      "type": "mrkdwn",
                      "text": "*Deployment Failed* ❌\n\n*Environment:* ${{ needs.validate.outputs.environment }}\n*Branch:* ${{ github.ref_name }}\n\n🤖 *Claude Code autonomous recovery initiated*"
                    }
                  }
                ]
              }'
          fi
```

---

## PR Validation Workflow

### `.github/workflows/pr-validation.yml`

```yaml
name: PR Validation

on:
  pull_request:
    branches:
      - main
      - 'release/*'

jobs:
  lint:
    name: Lint & Format
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      
      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest
      
      - name: Check formatting
        run: |
          if [ -n "$(gofmt -l .)" ]; then
            echo "❌ Code is not formatted. Run 'gofmt -w .'"
            gofmt -l .
            exit 1
          fi
          echo "✅ Code formatting is correct"

  test:
    name: Unit Tests
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      
      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...
      
      - name: Check coverage
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Coverage: ${COVERAGE}%"
          
          if (( $(echo "$COVERAGE < 70" | bc -l) )); then
            echo "❌ Coverage below 70%"
            exit 1
          fi
          echo "✅ Coverage acceptable"

  validate-templates:
    name: Validate SAM Template
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup SAM CLI
        uses: aws-actions/setup-sam@v2
      
      - name: Validate SAM template
        run: sam validate --lint
```

---

## Claude Code Failure Recovery Protocol

### Automatic Failure Detection

When a GitHub Actions workflow fails, Claude Code will automatically:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     CLAUDE CODE FAILURE RECOVERY PROTOCOL                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  STEP 1: DETECT FAILURE                                                     │
│  ─────────────────────                                                      │
│  • Monitor GitHub Actions workflow status                                   │
│  • Identify which job failed                                                │
│  • Extract failure logs                                                     │
│                                                                              │
│  STEP 2: ANALYZE ROOT CAUSE                                                 │
│  ──────────────────────────                                                 │
│  • Parse error messages from logs                                           │
│  • Identify error category:                                                 │
│    - Build error (Go compilation)                                           │
│    - Test failure (unit tests)                                              │
│    - Deployment error (SAM/CloudFormation)                                  │
│    - Configuration error (missing variables/secrets)                        │
│    - Infrastructure error (AWS service issues)                              │
│                                                                              │
│  STEP 3: IMPLEMENT FIX                                                      │
│  ─────────────────────                                                      │
│  • Generate fix based on error analysis                                     │
│  • Create or modify affected files                                          │
│  • Update documentation if needed                                           │
│                                                                              │
│  STEP 4: PUSH AND VERIFY                                                    │
│  ───────────────────────                                                    │
│  • Commit fix with descriptive message                                      │
│  • Push to trigger new workflow run                                         │
│  • Monitor new workflow                                                     │
│                                                                              │
│  STEP 5: ITERATE UNTIL SUCCESS                                              │
│  ─────────────────────────────                                              │
│  • If still failing, return to STEP 2                                       │
│  • Maximum iterations: 5 (then escalate to human)                           │
│  • Log all attempts for audit trail                                         │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Common Failure Patterns & Auto-Fixes

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
| **Quota exceeded** | `LimitExceededException` | Request quota increase or optimize |

### Recovery Commit Message Format

```
fix(ci): [AUTO-RECOVERY] Fix <error_type>

## Failure Details
- Workflow: <workflow_name>
- Job: <job_name>
- Error: <error_message>

## Root Cause
<analysis>

## Fix Applied
<description of changes>

## Recovery Attempt
Attempt #<n> of 5

Automated fix by Claude Code
```

---

## Manual Workflow Triggers

### Trigger Deployment Manually

```bash
# Via GitHub CLI
gh workflow run deploy.yml \
  --field environment=staging \
  --field force_deploy=true
```

### Trigger Rollback

```bash
# Via GitHub CLI
gh workflow run rollback.yml \
  --field environment=prod \
  --field target_version=v1.2.3
```

---

## Environment Protection Rules

### Production Environment

Configure in GitHub Settings → Environments → `prod`:

| Setting | Value |
|---------|-------|
| Required reviewers | 1 (optional for POC) |
| Wait timer | 0 minutes |
| Deployment branches | `release/*` only |
| Prevent self-review | No |

### Staging Environment

Configure in GitHub Settings → Environments → `staging`:

| Setting | Value |
|---------|-------|
| Required reviewers | 0 |
| Wait timer | 0 minutes |
| Deployment branches | `main` |

---

## Deployment Checklist

### Pre-Deployment

- [ ] All variables configured (see `variables.md`)
- [ ] All secrets configured
- [ ] AWS region confirmed (see `regions.md`)
- [ ] SAM S3 bucket exists
- [ ] Knowledge base S3 bucket exists
- [ ] IAM permissions configured

### Post-Deployment Validation

- [ ] Lambda function responding
- [ ] Bedrock agents created
- [ ] Knowledge bases synced
- [ ] Personas deployed to DynamoDB
- [ ] Connect flow updated (if applicable)
- [ ] End-to-end test call successful

---

## Monitoring & Observability

### CloudWatch Dashboards

The deployment creates CloudWatch dashboards for:

- Lambda invocation metrics
- Bedrock agent latency
- Error rates
- Cost tracking

### Alerts

Configure alerts for:

| Metric | Threshold | Action |
|--------|-----------|--------|
| Lambda errors | > 5/min | PagerDuty |
| Latency P99 | > 5s | Slack |
| Cost | > $100/day | Email |

---

## Summary

This deployment system ensures:

1. **All deployments go through GitHub Actions** - No manual deployments allowed
2. **Claude Code has full autonomy** - Can build, deploy, and fix issues automatically
3. **Automatic failure recovery** - Claude Code reviews failures and pushes fixes
4. **Continuous deployment** - Changes to main trigger staging, release/* triggers prod
5. **Full auditability** - All changes tracked in Git, all deployments in GitHub Actions

---

## Document History

| Date | Change | Author |
|------|--------|--------|
| 2025-12-31 | Initial creation | Claude Code |
