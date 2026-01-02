#!/usr/bin/env python3
"""
Create Bedrock Agents for Headset Support System
Supports both Anthropic Claude and AWS native Llama models
"""

import argparse
import boto3
import json
import time
import sys
from botocore.exceptions import ClientError

# Agent configurations
AGENTS = {
    "supervisor": {
        "name": "TroubleshootingOrchestrator",
        "description": "Supervisor agent that orchestrates headset troubleshooting conversations using sub-agents",
        "instruction": """You are a friendly headset troubleshooting supervisor. Your role is to:
1. Greet users warmly and identify their headset issue
2. Route diagnostic questions to the DiagnosticAgent
3. Route platform/software questions to the PlatformAgent
4. Detect escalation requests and route to EscalationAgent
5. Maintain conversation context and persona consistency

Always respond in a helpful, patient manner. Adapt your communication style based on the persona configuration provided in session attributes.""",
        "model_type": "supervisor"
    },
    "diagnostic": {
        "name": "DiagnosticAgent",
        "description": "Sub-agent specialized in hardware diagnosis for USB, Bluetooth, and wireless headsets",
        "instruction": """You are a hardware diagnostic specialist. Your role is to:
1. Identify the headset connection type (USB, Bluetooth, DECT)
2. Run through diagnostic steps for hardware issues
3. Check physical connections and power status
4. Verify device recognition in the operating system
5. Test audio input and output functionality

Be thorough but efficient. Ask one question at a time and wait for user response.""",
        "model_type": "subagent"
    },
    "platform": {
        "name": "PlatformAgent",
        "description": "Sub-agent specialized in Windows and application configuration for headsets",
        "instruction": """You are a platform configuration specialist. Your role is to:
1. Guide users through Windows Sound Settings
2. Help configure application-specific audio settings (Genesys Cloud, Teams, etc.)
3. Troubleshoot driver issues in Device Manager
4. Configure default audio devices
5. Set up WebRTC and browser permissions

Provide clear, step-by-step instructions. Reference specific Windows menu paths.""",
        "model_type": "subagent"
    },
    "escalation": {
        "name": "EscalationAgent",
        "description": "Sub-agent that handles escalation to human agents",
        "instruction": """You are an escalation coordinator. Your role is to:
1. Acknowledge the user's request for human assistance
2. Summarize the troubleshooting steps already attempted
3. Collect any additional context needed for the human agent
4. Prepare the handoff with all relevant information
5. Reassure the user that help is on the way

Be empathetic and efficient. The goal is a smooth transition to human support.""",
        "model_type": "subagent"
    }
}

# Model configurations
MODELS = {
    "anthropic": {
        "supervisor": "us.anthropic.claude-3-5-sonnet-20241022-v2:0",
        "subagent": "us.anthropic.claude-3-5-haiku-20241022-v1:0"
    },
    "llama": {
        "supervisor": "us.meta.llama3-3-70b-instruct-v1:0",
        "subagent": "us.meta.llama3-2-11b-instruct-v1:0"
    }
}


def get_bedrock_client(region):
    """Create Bedrock client"""
    return boto3.client('bedrock-agent', region_name=region)


def get_ssm_client(region):
    """Create SSM client"""
    return boto3.client('ssm', region_name=region)


def get_iam_client(region):
    """Create IAM client"""
    return boto3.client('iam', region_name=region)


def get_agent_role_arn(iam_client, environment):
    """Get the Bedrock agent role ARN"""
    role_name = f"BedrockAgentRole-{environment}"
    try:
        response = iam_client.get_role(RoleName=role_name)
        return response['Role']['Arn']
    except ClientError as e:
        print(f"Error getting role {role_name}: {e}")
        return None


def check_agent_exists(client, agent_name):
    """Check if an agent with the given name exists"""
    try:
        paginator = client.get_paginator('list_agents')
        for page in paginator.paginate():
            for agent in page.get('agentSummaries', []):
                if agent['agentName'] == agent_name:
                    return agent['agentId']
    except ClientError as e:
        print(f"Error listing agents: {e}")
    return None


def create_agent(client, agent_config, role_arn, model_id, environment):
    """Create or update a Bedrock agent"""
    agent_name = f"{agent_config['name']}-{environment}"

    # Check if agent already exists
    existing_id = check_agent_exists(client, agent_name)

    if existing_id:
        print(f"Agent {agent_name} already exists (ID: {existing_id}), updating...")
        try:
            client.update_agent(
                agentId=existing_id,
                agentName=agent_name,
                agentResourceRoleArn=role_arn,
                description=agent_config['description'],
                instruction=agent_config['instruction'],
                foundationModel=model_id,
                idleSessionTTLInSeconds=600
            )
            return existing_id
        except ClientError as e:
            print(f"Error updating agent: {e}")
            return existing_id

    print(f"Creating agent: {agent_name}")
    try:
        response = client.create_agent(
            agentName=agent_name,
            agentResourceRoleArn=role_arn,
            description=agent_config['description'],
            instruction=agent_config['instruction'],
            foundationModel=model_id,
            idleSessionTTLInSeconds=600
        )
        return response['agent']['agentId']
    except ClientError as e:
        print(f"Error creating agent {agent_name}: {e}")
        return None


def wait_for_agent(client, agent_id, timeout=120):
    """Wait for agent to be in PREPARED or NOT_PREPARED state"""
    print(f"Waiting for agent {agent_id} to be ready...")
    start_time = time.time()

    while time.time() - start_time < timeout:
        try:
            response = client.get_agent(agentId=agent_id)
            status = response['agent']['agentStatus']
            print(f"  Agent status: {status}")

            if status in ['PREPARED', 'NOT_PREPARED']:
                return status
            elif status == 'FAILED':
                print(f"  Agent failed: {response['agent'].get('failureReasons', 'Unknown')}")
                return status

        except ClientError as e:
            print(f"  Error checking agent status: {e}")

        time.sleep(5)

    print(f"Timeout waiting for agent {agent_id}")
    return None


def prepare_agent(client, agent_id):
    """Prepare an agent for deployment"""
    print(f"Preparing agent {agent_id}...")
    try:
        client.prepare_agent(agentId=agent_id)
        return wait_for_agent(client, agent_id)
    except ClientError as e:
        print(f"Error preparing agent: {e}")
        return None


def create_agent_alias(client, agent_id, alias_name, environment):
    """Create or update an agent alias"""
    full_alias_name = f"{alias_name}-{environment}"

    # Check for existing aliases
    try:
        response = client.list_agent_aliases(agentId=agent_id)
        for alias in response.get('agentAliasSummaries', []):
            if alias['agentAliasName'] == full_alias_name:
                print(f"Alias {full_alias_name} already exists")
                return alias['agentAliasId']
    except ClientError:
        pass

    print(f"Creating alias: {full_alias_name}")
    try:
        response = client.create_agent_alias(
            agentId=agent_id,
            agentAliasName=full_alias_name
        )
        return response['agentAlias']['agentAliasId']
    except ClientError as e:
        print(f"Error creating alias: {e}")
        return None


def store_ssm_parameter(ssm_client, name, value, description):
    """Store a parameter in SSM Parameter Store"""
    try:
        ssm_client.put_parameter(
            Name=name,
            Value=value,
            Type='String',
            Description=description,
            Overwrite=True
        )
        print(f"Stored SSM parameter: {name}")
    except ClientError as e:
        print(f"Error storing SSM parameter {name}: {e}")


def main():
    parser = argparse.ArgumentParser(description='Create Bedrock agents for Headset Support')
    parser.add_argument('--environment', '-e', default='dev', choices=['dev', 'staging', 'prod'],
                        help='Deployment environment')
    parser.add_argument('--region', '-r', default='us-east-1', help='AWS region')
    parser.add_argument('--model-provider', '-m', default='anthropic', choices=['anthropic', 'llama'],
                        help='Model provider (anthropic or llama)')
    parser.add_argument('--dry-run', action='store_true', help='Print what would be done without making changes')

    args = parser.parse_args()

    print(f"Creating Bedrock agents for environment: {args.environment}")
    print(f"Region: {args.region}")
    print(f"Model provider: {args.model_provider}")

    if args.dry_run:
        print("\n*** DRY RUN - No changes will be made ***\n")
        for agent_key, agent_config in AGENTS.items():
            model_type = agent_config['model_type']
            model_id = MODELS[args.model_provider][model_type if model_type == 'supervisor' else 'subagent']
            print(f"Would create agent: {agent_config['name']}-{args.environment}")
            print(f"  Model: {model_id}")
            print(f"  Description: {agent_config['description'][:50]}...")
        return

    # Initialize clients
    bedrock_client = get_bedrock_client(args.region)
    ssm_client = get_ssm_client(args.region)
    iam_client = get_iam_client(args.region)

    # Get agent role ARN
    role_arn = get_agent_role_arn(iam_client, args.environment)
    if not role_arn:
        print("ERROR: Could not find Bedrock agent role. Deploy infrastructure first.")
        sys.exit(1)
    print(f"Using role: {role_arn}")

    # Create agents
    agent_ids = {}
    for agent_key, agent_config in AGENTS.items():
        model_type = agent_config['model_type']
        model_id = MODELS[args.model_provider][model_type if model_type == 'supervisor' else 'subagent']

        agent_id = create_agent(bedrock_client, agent_config, role_arn, model_id, args.environment)
        if agent_id:
            agent_ids[agent_key] = agent_id

            # Prepare the agent
            status = prepare_agent(bedrock_client, agent_id)
            if status != 'PREPARED':
                print(f"Warning: Agent {agent_key} is not prepared (status: {status})")

    # Create aliases and store in SSM
    for agent_key, agent_id in agent_ids.items():
        alias_id = create_agent_alias(bedrock_client, agent_id, "live", args.environment)

        if agent_key == "supervisor":
            # Store supervisor agent info in SSM
            store_ssm_parameter(
                ssm_client,
                f"/headset-agent/{args.environment}/supervisor-agent-id",
                agent_id,
                "Bedrock Supervisor Agent ID"
            )
            store_ssm_parameter(
                ssm_client,
                f"/headset-agent/{args.environment}/supervisor-agent-alias",
                alias_id,
                "Bedrock Supervisor Agent Alias ID"
            )

    print("\n=== Agent Creation Complete ===")
    for agent_key, agent_id in agent_ids.items():
        print(f"  {agent_key}: {agent_id}")


if __name__ == '__main__':
    main()
