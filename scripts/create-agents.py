#!/usr/bin/env python3
"""
Bedrock Agent Creation Script
Creates supervisor and sub-agents for the Headset Support Agent

This script handles resources that cannot be created via SAM/CloudFormation:
- Bedrock Agents
- Agent Collaborators (Multi-Agent)
- Knowledge Base associations
"""

import argparse
import boto3
import json
import time
import os
from botocore.exceptions import ClientError

# =============================================================================
# MODEL PROVIDER CONFIGURATION
# =============================================================================
# Set to True to use AWS native models (Amazon Titan) - no use case form required
# Set to False to use Anthropic Claude models - requires Anthropic use case form
USE_AWS_NATIVE_MODELS = False

# Anthropic Claude Models (require use case form submission in AWS console)
CLAUDE_SUPERVISOR_MODEL = "us.anthropic.claude-3-5-sonnet-20241022-v2:0"
CLAUDE_SUBAGENT_MODEL = "us.anthropic.claude-3-5-haiku-20241022-v1:0"

# AWS Native Models - use inference profiles for cross-region support
# Meta Llama 3.3 - excellent instruction following
NATIVE_SUPERVISOR_MODEL = "us.meta.llama3-3-70b-instruct-v1:0"
NATIVE_SUBAGENT_MODEL = "us.meta.llama3-2-11b-instruct-v1:0"
# =============================================================================


def get_agent_role_arn(environment: str, region: str) -> str:
    """Get the Bedrock agent role ARN from CloudFormation stack outputs"""
    cf_client = boto3.client('cloudformation', region_name=region)
    stack_name = f"headset-agent-stack-{environment}"

    try:
        response = cf_client.describe_stacks(StackName=stack_name)
        outputs = response['Stacks'][0]['Outputs']
        for output in outputs:
            if output['OutputKey'] == 'BedrockAgentRoleArn':
                return output['OutputValue']
    except ClientError as e:
        print(f"Warning: Could not get role from stack: {e}")

    # Fallback to constructing the ARN
    sts = boto3.client('sts')
    account_id = sts.get_caller_identity()['Account']
    return f"arn:aws:iam::{account_id}:role/BedrockAgentRole-{environment}"


def create_agent(client, name: str, role_arn: str, model_id: str, instruction: str) -> dict:
    """Create or update a Bedrock agent"""
    print(f"Creating agent: {name}")

    try:
        # Check if agent already exists
        agents = client.list_agents()
        for agent in agents.get('agentSummaries', []):
            if agent['agentName'] == name:
                agent_id = agent['agentId']
                print(f"  Agent {name} already exists with ID: {agent_id}")

                # Update the agent with new model/instruction
                print(f"  Updating agent with model: {model_id}")
                client.update_agent(
                    agentId=agent_id,
                    agentName=name,
                    agentResourceRoleArn=role_arn,
                    foundationModel=model_id,
                    instruction=instruction,
                    idleSessionTTLInSeconds=600,
                    description=f"Headset Support Agent - {name}"
                )

                # Re-prepare the agent after update
                print(f"  Re-preparing agent after update...")
                client.prepare_agent(agentId=agent_id)
                wait_for_agent(client, agent_id)

                return {'agentId': agent_id, 'exists': True, 'updated': True}

        response = client.create_agent(
            agentName=name,
            agentResourceRoleArn=role_arn,
            foundationModel=model_id,
            instruction=instruction,
            idleSessionTTLInSeconds=600,
            description=f"Headset Support Agent - {name}"
        )

        agent_id = response['agent']['agentId']
        print(f"  Created agent with ID: {agent_id}")

        # Wait for agent to be ready
        wait_for_agent(client, agent_id)

        return {'agentId': agent_id, 'exists': False}

    except ClientError as e:
        print(f"  Error creating agent {name}: {e}")
        raise


def wait_for_agent(client, agent_id: str, timeout: int = 120):
    """Wait for agent to be in PREPARED or NOT_PREPARED status"""
    print(f"  Waiting for agent {agent_id} to be ready...")

    start_time = time.time()
    while time.time() - start_time < timeout:
        response = client.get_agent(agentId=agent_id)
        status = response['agent']['agentStatus']

        if status in ['PREPARED', 'NOT_PREPARED']:
            print(f"  Agent status: {status}")
            return

        print(f"  Current status: {status}, waiting...")
        time.sleep(5)

    raise TimeoutError(f"Agent {agent_id} did not become ready within {timeout} seconds")


def prepare_agent(client, agent_id: str):
    """Prepare an agent for deployment"""
    print(f"  Preparing agent {agent_id}...")

    try:
        client.prepare_agent(agentId=agent_id)
        wait_for_agent(client, agent_id)
        print(f"  Agent {agent_id} prepared successfully")
    except ClientError as e:
        print(f"  Error preparing agent: {e}")
        raise


def create_or_update_agent_alias(client, agent_id: str, alias_name: str = "prod", force_update: bool = False) -> str:
    """Create or update an alias for the agent to point to latest prepared version"""
    print(f"  Managing alias '{alias_name}' for agent {agent_id}...")

    try:
        # Check if alias exists
        aliases = client.list_agent_aliases(agentId=agent_id)
        for alias in aliases.get('agentAliasSummaries', []):
            if alias['agentAliasName'] == alias_name:
                alias_id = alias['agentAliasId']
                print(f"  Alias already exists: {alias_id}")

                if force_update:
                    # Update the alias to point to the latest prepared version
                    print(f"  Updating alias to point to latest agent version...")
                    client.update_agent_alias(
                        agentId=agent_id,
                        agentAliasId=alias_id,
                        agentAliasName=alias_name,
                        description=f"Production alias for agent (updated)"
                    )
                    # Wait for alias to be ready
                    wait_for_alias(client, agent_id, alias_id)
                    print(f"  Alias updated successfully")

                return alias_id

        response = client.create_agent_alias(
            agentId=agent_id,
            agentAliasName=alias_name,
            description=f"Production alias for agent"
        )

        alias_id = response['agentAlias']['agentAliasId']
        print(f"  Created alias: {alias_id}")

        # Wait for alias to be ready
        wait_for_alias(client, agent_id, alias_id)

        return alias_id

    except ClientError as e:
        print(f"  Error managing alias: {e}")
        raise


def wait_for_alias(client, agent_id: str, alias_id: str, timeout: int = 120):
    """Wait for agent alias to be in PREPARED status"""
    print(f"  Waiting for alias {alias_id} to be ready...")

    start_time = time.time()
    while time.time() - start_time < timeout:
        response = client.get_agent_alias(agentId=agent_id, agentAliasId=alias_id)
        status = response['agentAlias']['agentAliasStatus']

        if status == 'PREPARED':
            print(f"  Alias status: {status}")
            return

        if status == 'FAILED':
            raise Exception(f"Alias {alias_id} failed to prepare")

        print(f"  Current alias status: {status}, waiting...")
        time.sleep(5)

    raise TimeoutError(f"Alias {alias_id} did not become ready within {timeout} seconds")


def update_ssm_parameter(param_name: str, value: str, region: str):
    """Update SSM parameter with agent ID"""
    ssm = boto3.client('ssm', region_name=region)

    try:
        ssm.put_parameter(
            Name=param_name,
            Value=value,
            Type='String',
            Overwrite=True
        )
        print(f"  Updated SSM parameter: {param_name}")
    except ClientError as e:
        print(f"  Error updating SSM parameter: {e}")
        raise


def main():
    parser = argparse.ArgumentParser(description='Create Bedrock agents for Headset Support')
    parser.add_argument('--environment', '-e', default='dev', choices=['dev', 'staging', 'prod'])
    parser.add_argument('--region', '-r', default=os.environ.get('AWS_REGION', 'us-east-1'))
    parser.add_argument('--skip-subagents', action='store_true', help='Only create supervisor agent')
    args = parser.parse_args()

    print(f"\n{'='*60}")
    print(f"  BEDROCK AGENT CREATION")
    print(f"  Environment: {args.environment}")
    print(f"  Region: {args.region}")
    print(f"{'='*60}\n")

    client = boto3.client('bedrock-agent', region_name=args.region)
    role_arn = get_agent_role_arn(args.environment, args.region)

    print(f"Using role ARN: {role_arn}\n")

    # Select models based on configuration
    if USE_AWS_NATIVE_MODELS:
        supervisor_model = NATIVE_SUPERVISOR_MODEL
        subagent_model = NATIVE_SUBAGENT_MODEL
        print(f"Using AWS Native Models (Meta Llama)")
    else:
        supervisor_model = CLAUDE_SUPERVISOR_MODEL
        subagent_model = CLAUDE_SUBAGENT_MODEL
        print(f"Using Anthropic Claude Models")

    print(f"  Supervisor: {supervisor_model}")
    print(f"  Subagent: {subagent_model}\n")

    # Agent instructions
    agents_config = {
        "TroubleshootingOrchestrator": {
            "model": supervisor_model,
            "instruction": """You are a friendly headset support specialist. Help customers fix their headset problems.

WHEN USER REPORTS A PROBLEM, IMMEDIATELY HELP BY:
1. Acknowledging their issue
2. Asking ONE specific diagnostic question OR suggesting ONE troubleshooting step

COMMON HEADSET PROBLEMS AND SOLUTIONS:

NO AUDIO:
- Check if headset is powered on
- Verify volume is not muted (check headset and computer)
- Ensure correct audio output is selected in system settings
- Try unplugging and reconnecting the headset
- Test on another device to isolate the problem

BLUETOOTH WON'T CONNECT:
- Make sure headset is in pairing mode (usually hold power button)
- Turn Bluetooth off and on again on the device
- Remove old pairing and re-pair fresh
- Check headset battery level
- Move closer to the device

MICROPHONE NOT WORKING:
- Check if mic is muted on headset
- Verify microphone permissions in app settings
- Select correct input device in system settings
- Test mic in another application

RESPONSE STYLE:
- Be warm and conversational
- Give ONE step at a time
- Keep responses to 2-3 sentences
- Ask if the step helped before moving on
- NEVER include JSON, function calls, or code in your responses
- Respond in plain conversational English only

If user asks for a human agent, acknowledge and say you'll transfer them."""
        },
        "DiagnosticAgent": {
            "model": subagent_model,
            "instruction": """You are a hardware diagnostic specialist for headsets. Your expertise covers:
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
4. Detailed component isolation (left/right, mic, speakers)"""
        },
        "PlatformAgent": {
            "model": subagent_model,
            "instruction": """You are a platform configuration specialist for audio devices. Your expertise covers:
- Windows 10/11 audio device management
- macOS audio preferences and permissions
- Application-specific audio settings (Teams, Zoom, Genesys Cloud)

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
5. Test with built-in OS tools before application"""
        },
        "EscalationAgent": {
            "model": subagent_model,
            "instruction": """You are an escalation specialist responsible for smooth handoffs to human agents.

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

ESCALATION PROTOCOL:
1. Confirm user wants to proceed with escalation
2. Generate conversation summary
3. Collect any additional required information
4. Create ticket or initiate transfer
5. Provide user with reference number and expectations"""
        }
    }

    created_agents = {}

    # Create supervisor agent first
    supervisor_name = f"TroubleshootingOrchestrator-{args.environment}"
    supervisor = create_agent(
        client,
        supervisor_name,
        role_arn,
        agents_config["TroubleshootingOrchestrator"]["model"],
        agents_config["TroubleshootingOrchestrator"]["instruction"]
    )
    created_agents["supervisor"] = supervisor

    if not supervisor.get('exists'):
        prepare_agent(client, supervisor['agentId'])

    # Create or update alias for supervisor (force update if agent was updated)
    alias_id = create_or_update_agent_alias(
        client,
        supervisor['agentId'],
        force_update=supervisor.get('updated', False)
    )

    # Update SSM parameters
    update_ssm_parameter(
        f"/headset-agent/{args.environment}/supervisor-agent-id",
        supervisor['agentId'],
        args.region
    )
    update_ssm_parameter(
        f"/headset-agent/{args.environment}/supervisor-agent-alias",
        alias_id,
        args.region
    )

    if not args.skip_subagents:
        # Create sub-agents
        for agent_name in ["DiagnosticAgent", "PlatformAgent", "EscalationAgent"]:
            full_name = f"{agent_name}-{args.environment}"
            config = agents_config[agent_name]

            agent = create_agent(
                client,
                full_name,
                role_arn,
                config["model"],
                config["instruction"]
            )
            created_agents[agent_name] = agent

            if not agent.get('exists'):
                prepare_agent(client, agent['agentId'])
            # Create or update alias (force update if agent was updated)
            create_or_update_agent_alias(
                client,
                agent['agentId'],
                force_update=agent.get('updated', False)
            )

    print(f"\n{'='*60}")
    print("  AGENT CREATION COMPLETE")
    print(f"{'='*60}")
    print(f"\nCreated agents:")
    for name, agent in created_agents.items():
        status = "existing" if agent.get('exists') else "new"
        print(f"  - {name}: {agent['agentId']} ({status})")

    print(f"\nSSM Parameters updated:")
    print(f"  - /headset-agent/{args.environment}/supervisor-agent-id")
    print(f"  - /headset-agent/{args.environment}/supervisor-agent-alias")
    print()


if __name__ == "__main__":
    main()
