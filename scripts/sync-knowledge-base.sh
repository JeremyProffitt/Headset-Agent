#!/bin/bash
# Sync knowledge base documents to S3

set -e

ENVIRONMENT=${ENVIRONMENT:-dev}
REGION=${AWS_REGION:-us-east-1}

# Get bucket name from CloudFormation stack
STACK_NAME="headset-agent-stack-${ENVIRONMENT}"
KB_BUCKET=$(aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --query "Stacks[0].Outputs[?OutputKey=='KnowledgeBaseBucketName'].OutputValue" \
    --output text \
    --region "$REGION" 2>/dev/null || echo "")

if [ -z "$KB_BUCKET" ]; then
    echo "Error: Could not get KB bucket from stack. Checking variable..."
    KB_BUCKET="${KB_S3_BUCKET_NAME}-${ENVIRONMENT}"
fi

echo "=========================================="
echo "  Knowledge Base Sync"
echo "  Environment: ${ENVIRONMENT}"
echo "  Bucket: ${KB_BUCKET}"
echo "=========================================="

# Sync knowledge base documents
echo "Syncing knowledge base documents..."
aws s3 sync knowledge-base/ "s3://${KB_BUCKET}/" \
    --delete \
    --exclude ".git/*" \
    --exclude "*.DS_Store" \
    --region "$REGION"

echo ""
echo "Knowledge base sync complete!"
echo ""

# List synced files
echo "Synced files:"
aws s3 ls "s3://${KB_BUCKET}/" --recursive --region "$REGION"
