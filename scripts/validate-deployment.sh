#!/bin/bash
# Validate deployment health

set -e

ENVIRONMENT=${ENVIRONMENT:-dev}
REGION=${AWS_REGION:-us-east-1}

echo "=========================================="
echo "  Deployment Validation"
echo "  Environment: ${ENVIRONMENT}"
echo "  Region: ${REGION}"
echo "=========================================="

STACK_NAME="headset-agent-stack-${ENVIRONMENT}"

# Check CloudFormation stack status
echo ""
echo "Checking CloudFormation stack..."
STACK_STATUS=$(aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --query "Stacks[0].StackStatus" \
    --output text \
    --region "$REGION" 2>/dev/null || echo "NOT_FOUND")

if [ "$STACK_STATUS" = "NOT_FOUND" ]; then
    echo "  ERROR: Stack not found"
    exit 1
elif [[ "$STACK_STATUS" == *"FAILED"* ]] || [[ "$STACK_STATUS" == *"ROLLBACK"* ]]; then
    echo "  ERROR: Stack in failed state: $STACK_STATUS"
    exit 1
else
    echo "  Stack status: $STACK_STATUS"
fi

# Get Lambda function name
LAMBDA_NAME=$(aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --query "Stacks[0].Outputs[?OutputKey=='LambdaFunctionName'].OutputValue" \
    --output text \
    --region "$REGION")

echo ""
echo "Testing Lambda function: $LAMBDA_NAME"

# Test Lambda invocation
RESPONSE=$(aws lambda invoke \
    --function-name "$LAMBDA_NAME" \
    --payload '{"sessionState":{"sessionAttributes":{"test":"true"}},"inputTranscript":""}' \
    --cli-binary-format raw-in-base64-out \
    --region "$REGION" \
    /tmp/response.json 2>&1)

STATUS_CODE=$(echo "$RESPONSE" | grep -o '"StatusCode": [0-9]*' | grep -o '[0-9]*' || echo "0")

if [ "$STATUS_CODE" = "200" ]; then
    echo "  Lambda invocation: SUCCESS"
    cat /tmp/response.json
else
    echo "  Lambda invocation: FAILED"
    echo "  Response: $RESPONSE"
    exit 1
fi

# Check SSM parameters
echo ""
echo "Checking SSM parameters..."
AGENT_ID=$(aws ssm get-parameter \
    --name "/headset-agent/${ENVIRONMENT}/supervisor-agent-id" \
    --query "Parameter.Value" \
    --output text \
    --region "$REGION" 2>/dev/null || echo "NOT_SET")

if [ "$AGENT_ID" = "PLACEHOLDER" ] || [ "$AGENT_ID" = "NOT_SET" ]; then
    echo "  WARNING: Supervisor agent ID not configured"
    echo "  Run: python scripts/create-agents.py --environment $ENVIRONMENT"
else
    echo "  Supervisor Agent ID: $AGENT_ID"
fi

# Check DynamoDB tables
echo ""
echo "Checking DynamoDB tables..."
PERSONA_TABLE=$(aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --query "Stacks[0].Outputs[?OutputKey=='PersonaTableName'].OutputValue" \
    --output text \
    --region "$REGION")

ITEM_COUNT=$(aws dynamodb describe-table \
    --table-name "$PERSONA_TABLE" \
    --query "Table.ItemCount" \
    --output text \
    --region "$REGION" 2>/dev/null || echo "0")

echo "  Persona table: $PERSONA_TABLE"
echo "  Item count: $ITEM_COUNT"

if [ "$ITEM_COUNT" = "0" ]; then
    echo "  WARNING: No personas configured. Deploy personas!"
fi

# Summary
echo ""
echo "=========================================="
echo "  VALIDATION SUMMARY"
echo "=========================================="
echo "  Stack Status: $STACK_STATUS"
echo "  Lambda: PASSED"
echo "  Agent ID: $AGENT_ID"
echo "  Personas: $ITEM_COUNT configured"
echo "=========================================="
