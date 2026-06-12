#!/usr/bin/env python3
"""
Create the Bedrock supervisor agent for the Headset Support System (A-07).

Single-agent topology: earlier revisions created three additional sub-agents
(DiagnosticAgent / PlatformAgent / EscalationAgent) that were never wired into
a collaborator hierarchy — they sat orphaned while the Lambda only ever
invoked the supervisor. This script now:

  1. deletes those orphaned sub-agents if they still exist,
  2. creates/updates ONE supervisor agent,
  3. associates the knowledge base (SSM /headset-agent/<env>/kb-id) with the
     agent's DRAFT version,
  4. prepares the agent and points the live alias at a freshly published
     version (so the KB association is actually served),
  5. stores supervisor-agent-id / supervisor-agent-alias in SSM (the same
     parameters the lex-lambda reads), and
  6. asserts exactly one prepared headset agent exists with the KB associated
     and that the orphans are gone.

Idempotent: find-before-create everywhere; running twice creates no dupes.
"""

import argparse
import boto3
import sys
import time
from botocore.exceptions import ClientError

# The single agent this system uses. The Lambda answers primarily via direct
# knowledge-base RetrieveAndGenerate (A-08); this agent is the legacy/backup
# conversational path and is grounded in the same knowledge base.
SUPERVISOR_AGENT = {
    "name": "TroubleshootingOrchestrator",
    "description": "Headset troubleshooting agent grounded in the headset support knowledge base",
    "instruction": """You are a friendly headset troubleshooting agent. Your role is to:
1. Greet users warmly and identify their headset issue
2. Answer questions using ONLY the associated headset support knowledge base — search it before answering
3. Never invent troubleshooting steps, settings, or menu paths that are not in the knowledge base
4. If the knowledge base does not cover the question, say so plainly and offer to connect the user to a human specialist
5. Detect escalation requests and acknowledge them empathetically
6. Maintain conversation context and persona consistency

Always respond in a helpful, patient manner, in two to three short spoken-style
sentences. Adapt your communication style based on the persona configuration
provided in session attributes. Never ask for or accept payment or card details.""",
}

# Orphaned sub-agent base names from the old multi-agent topology. They are
# deleted if found (idempotent: absent == already done).
ORPHANED_AGENT_NAMES = ["DiagnosticAgent", "PlatformAgent", "EscalationAgent"]

# Model configurations (supervisor only — sub-agents no longer exist).
MODELS = {
    "anthropic": {
        # claude-3-5-sonnet-20241022-v2:0 is END-OF-LIFE (retired) — every invoke
        # returned ResourceNotFoundException. Use the current Sonnet 4.6 profile
        # (verified invokable in this account/region).
        "supervisor": "us.anthropic.claude-sonnet-4-6",
    },
    "llama": {
        "supervisor": "us.meta.llama3-3-70b-instruct-v1:0",
    },
}


def get_bedrock_client(region):
    """Create Bedrock agent (control plane) client"""
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


def get_kb_id(ssm_client, environment):
    """Read the knowledge base ID from SSM (populated by CloudFormation)."""
    param_name = f"/headset-agent/{environment}/kb-id"
    try:
        response = ssm_client.get_parameter(Name=param_name)
        value = response['Parameter']['Value']
        if not value or value == 'PLACEHOLDER':
            print(f"ERROR: SSM parameter {param_name} has no usable value ({value!r})")
            return None
        return value
    except ClientError as e:
        print(f"ERROR: could not read SSM parameter {param_name}: {e}")
        return None


def get_guardrail_config(ssm_client, environment):
    """Read the guardrail ID and version from SSM (A-09, populated by CloudFormation).

    Returns (guardrail_id, guardrail_version) when both are present and usable,
    or (None, None) when absent or still PLACEHOLDER. A missing guardrail is
    non-fatal: the agent will be created/updated without one.
    """
    id_param = f"/headset-agent/{environment}/guardrail-id"
    ver_param = f"/headset-agent/{environment}/guardrail-version"
    try:
        id_resp = ssm_client.get_parameter(Name=id_param)
        guardrail_id = id_resp['Parameter']['Value']
        if not guardrail_id or guardrail_id == 'PLACEHOLDER':
            print(f"WARNING: SSM parameter {id_param} has no usable value ({guardrail_id!r}) — skipping guardrail attach")
            return None, None
    except ClientError as e:
        print(f"WARNING: could not read SSM parameter {id_param}: {e} — skipping guardrail attach")
        return None, None

    try:
        ver_resp = ssm_client.get_parameter(Name=ver_param)
        guardrail_version = ver_resp['Parameter']['Value']
        if not guardrail_version or guardrail_version == 'PLACEHOLDER':
            print(f"WARNING: SSM parameter {ver_param} has no usable value ({guardrail_version!r}) — skipping guardrail attach")
            return None, None
        return guardrail_id, guardrail_version
    except ClientError as e:
        print(f"WARNING: could not read SSM parameter {ver_param}: {e} — skipping guardrail attach")
        return None, None


def check_agent_exists(client, agent_name):
    """Check if an agent with the given name exists; return its ID or None"""
    try:
        paginator = client.get_paginator('list_agents')
        for page in paginator.paginate():
            for agent in page.get('agentSummaries', []):
                if agent['agentName'] == agent_name:
                    return agent['agentId']
    except ClientError as e:
        print(f"Error listing agents: {e}")
    return None


def delete_orphaned_agents(client, environment):
    """Delete the legacy sub-agents from the old multi-agent topology.

    Idempotent: agents that are already gone are skipped. Returns the list of
    orphan names that still exist after the deletion attempts (empty = clean).
    """
    remaining = []
    for base_name in ORPHANED_AGENT_NAMES:
        agent_name = f"{base_name}-{environment}"
        agent_id = check_agent_exists(client, agent_name)
        if not agent_id:
            print(f"Orphaned agent {agent_name}: not found (already deleted)")
            continue
        print(f"Deleting orphaned agent {agent_name} (ID: {agent_id})...")
        try:
            client.delete_agent(agentId=agent_id, skipResourceInUseCheck=True)
        except ClientError as e:
            print(f"  Error deleting {agent_name}: {e}")
            remaining.append(agent_name)
            continue
        # Wait for the deletion to complete so the final assertion is accurate.
        deadline = time.time() + 120
        while time.time() < deadline:
            if check_agent_exists(client, agent_name) is None:
                print(f"  Deleted {agent_name}")
                break
            time.sleep(5)
        else:
            print(f"  Timeout waiting for {agent_name} deletion")
            remaining.append(agent_name)
    return remaining


def create_or_update_agent(client, agent_config, role_arn, model_id, environment,
                           guardrail_id=None, guardrail_version=None):
    """Create the supervisor agent, or update it in place if it exists.

    When guardrail_id and guardrail_version are both provided, the guardrail is
    attached to the agent (belt-and-suspenders alongside the RetrieveAndGenerate
    guardrail). When absent the kwarg is omitted entirely so existing behaviour
    is unchanged. A guardrail-attach failure prints a warning but does not crash.
    """
    agent_name = f"{agent_config['name']}-{environment}"
    existing_id = check_agent_exists(client, agent_name)

    # Build optional guardrail kwarg only when both values are present.
    guardrail_kwargs = {}
    if guardrail_id and guardrail_version:
        guardrail_kwargs['guardrailConfiguration'] = {
            'guardrailIdentifier': guardrail_id,
            'guardrailVersion': guardrail_version,
        }

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
                idleSessionTTLInSeconds=600,
                **guardrail_kwargs
            )
        except ClientError as e:
            if guardrail_kwargs:
                print(f"WARNING: failed to attach guardrail during update — continuing without it: {e}")
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
                except ClientError as e2:
                    print(f"Error updating agent: {e2}")
            else:
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
            idleSessionTTLInSeconds=600,
            **guardrail_kwargs
        )
        return response['agent']['agentId']
    except ClientError as e:
        if guardrail_kwargs:
            print(f"WARNING: failed to attach guardrail during create — retrying without it: {e}")
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
            except ClientError as e2:
                print(f"Error creating agent {agent_name}: {e2}")
                return None
        print(f"Error creating agent {agent_name}: {e}")
        return None


def wait_for_agent_ready(client, agent_id, target_states, timeout=120):
    """Wait for agent to reach one of the target states"""
    print(f"Waiting for agent {agent_id} to reach state: {target_states}...")
    start_time = time.time()

    while time.time() - start_time < timeout:
        try:
            response = client.get_agent(agentId=agent_id)
            status = response['agent']['agentStatus']
            print(f"  Agent status: {status}")

            if status in target_states:
                return status
            elif status == 'FAILED':
                print(f"  Agent failed: {response['agent'].get('failureReasons', 'Unknown')}")
                return status

        except ClientError as e:
            print(f"  Error checking agent status: {e}")

        time.sleep(5)

    print(f"Timeout waiting for agent {agent_id}")
    return None


def kb_association_state(client, agent_id, kb_id):
    """Return the knowledgeBaseState of the DRAFT association, or None."""
    try:
        paginator = client.get_paginator('list_agent_knowledge_bases')
        for page in paginator.paginate(agentId=agent_id, agentVersion='DRAFT'):
            for kb in page.get('agentKnowledgeBaseSummaries', []):
                if kb['knowledgeBaseId'] == kb_id:
                    return kb.get('knowledgeBaseState', 'ENABLED')
    except ClientError as e:
        print(f"Error listing agent knowledge bases: {e}")
    return None


def associate_knowledge_base(client, agent_id, kb_id):
    """Associate (or re-enable) the knowledge base on the agent's DRAFT version.

    Idempotent: an existing ENABLED association is left alone; a DISABLED one
    is re-enabled; otherwise a new association is created. Returns True on
    success.
    """
    description = ("Headset troubleshooting knowledge base — decision trees, "
                   "brand guides, and platform/app configuration docs. "
                   "Search it before answering any troubleshooting question.")
    state = kb_association_state(client, agent_id, kb_id)
    if state == 'ENABLED':
        print(f"Knowledge base {kb_id} already associated and ENABLED")
        return True
    if state is not None:
        print(f"Knowledge base {kb_id} associated but {state}; re-enabling...")
        try:
            client.update_agent_knowledge_base(
                agentId=agent_id,
                agentVersion='DRAFT',
                knowledgeBaseId=kb_id,
                description=description,
                knowledgeBaseState='ENABLED'
            )
            return True
        except ClientError as e:
            print(f"Error re-enabling knowledge base association: {e}")
            return False

    print(f"Associating knowledge base {kb_id} with agent {agent_id}...")
    try:
        client.associate_agent_knowledge_base(
            agentId=agent_id,
            agentVersion='DRAFT',
            knowledgeBaseId=kb_id,
            description=description,
            knowledgeBaseState='ENABLED'
        )
        return True
    except ClientError as e:
        print(f"Error associating knowledge base: {e}")
        return False


def prepare_agent(client, agent_id):
    """Prepare the agent so the DRAFT changes (instruction + KB) take effect."""
    print(f"Waiting for agent {agent_id} to finish creating...")
    ready_status = wait_for_agent_ready(
        client, agent_id, ['NOT_PREPARED', 'PREPARED', 'FAILED'], timeout=120)

    if ready_status == 'FAILED':
        print(f"Agent {agent_id} is in FAILED state, cannot prepare")
        return 'FAILED'
    if ready_status is None:
        print(f"Timeout waiting for agent {agent_id} to finish creating")
        return None

    # Always (re-)prepare: the instruction and/or KB association may have
    # changed on DRAFT even when the status still says PREPARED.
    print(f"Preparing agent {agent_id}...")
    try:
        client.prepare_agent(agentId=agent_id)
        return wait_for_agent_ready(client, agent_id, ['PREPARED'], timeout=120)
    except ClientError as e:
        print(f"Error preparing agent: {e}")
        return None


def ensure_agent_alias(client, agent_id, alias_name, environment):
    """Create the live alias, or update it so a new version is published from
    the freshly prepared DRAFT (update_agent_alias without an explicit routing
    configuration publishes a new version). Returns the alias ID or None."""
    full_alias_name = f"{alias_name}-{environment}"

    alias_id = None
    try:
        response = client.list_agent_aliases(agentId=agent_id)
        for alias in response.get('agentAliasSummaries', []):
            if alias['agentAliasName'] == full_alias_name:
                alias_id = alias['agentAliasId']
                break
    except ClientError as e:
        print(f"Error listing aliases: {e}")

    if alias_id:
        print(f"Alias {full_alias_name} exists ({alias_id}); publishing new version...")
        try:
            client.update_agent_alias(
                agentId=agent_id,
                agentAliasId=alias_id,
                agentAliasName=full_alias_name
            )
        except ClientError as e:
            print(f"Error updating alias {full_alias_name}: {e}")
    else:
        print(f"Creating alias: {full_alias_name}")
        try:
            response = client.create_agent_alias(
                agentId=agent_id,
                agentAliasName=full_alias_name
            )
            alias_id = response['agentAlias']['agentAliasId']
        except ClientError as e:
            print(f"Error creating alias: {e}")
            return None

    # Wait for the alias to finish updating/creating.
    deadline = time.time() + 120
    while time.time() < deadline:
        try:
            response = client.get_agent_alias(agentId=agent_id, agentAliasId=alias_id)
            status = response['agentAlias']['agentAliasStatus']
            print(f"  Alias status: {status}")
            if status == 'PREPARED':
                return alias_id
            if status == 'FAILED':
                print(f"  Alias failed: {response['agentAlias'].get('failureReasons', 'Unknown')}")
                return alias_id
        except ClientError as e:
            print(f"  Error checking alias status: {e}")
        time.sleep(5)

    print(f"Timeout waiting for alias {alias_id}")
    return alias_id


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


def assert_single_prepared_agent(client, agent_id, kb_id, environment, orphans_remaining):
    """Final invariant check: exactly one prepared headset agent, KB attached,
    orphans gone. Returns True when everything holds."""
    ok = True

    try:
        status = client.get_agent(agentId=agent_id)['agent']['agentStatus']
    except ClientError as e:
        print(f"ASSERT FAIL: could not read supervisor agent {agent_id}: {e}")
        return False
    if status != 'PREPARED':
        print(f"ASSERT FAIL: supervisor agent status is {status}, want PREPARED")
        ok = False
    else:
        print(f"ASSERT OK: supervisor agent {agent_id} is PREPARED")

    state = kb_association_state(client, agent_id, kb_id)
    if state != 'ENABLED':
        print(f"ASSERT FAIL: knowledge base {kb_id} association state is {state}, want ENABLED")
        ok = False
    else:
        print(f"ASSERT OK: knowledge base {kb_id} is associated and ENABLED")

    if orphans_remaining:
        print(f"ASSERT FAIL: orphaned sub-agents still present: {orphans_remaining}")
        ok = False
    else:
        for base_name in ORPHANED_AGENT_NAMES:
            agent_name = f"{base_name}-{environment}"
            if check_agent_exists(client, agent_name):
                print(f"ASSERT FAIL: orphaned agent {agent_name} still exists")
                ok = False
        if ok:
            print("ASSERT OK: no orphaned sub-agents remain")

    return ok


def main():
    parser = argparse.ArgumentParser(description='Create the Bedrock supervisor agent for Headset Support')
    parser.add_argument('--environment', '-e', default='prod', choices=['prod'],
                        help='Deployment environment')
    parser.add_argument('--region', '-r', default='us-east-1', help='AWS region')
    parser.add_argument('--model-provider', '-m', default='anthropic', choices=['anthropic', 'llama'],
                        help='Model provider (anthropic or llama)')
    parser.add_argument('--dry-run', action='store_true', help='Print what would be done without making changes')

    args = parser.parse_args()

    model_id = MODELS[args.model_provider]['supervisor']

    print(f"Configuring Bedrock supervisor agent for environment: {args.environment}")
    print(f"Region: {args.region}")
    print(f"Model provider: {args.model_provider} ({model_id})")

    if args.dry_run:
        print("\n*** DRY RUN - No changes will be made ***\n")
        print(f"Would delete orphaned sub-agents (if present): "
              f"{[f'{n}-{args.environment}' for n in ORPHANED_AGENT_NAMES]}")
        print(f"Would create/update agent: {SUPERVISOR_AGENT['name']}-{args.environment}")
        print(f"  Model: {model_id}")
        print(f"  Description: {SUPERVISOR_AGENT['description'][:60]}...")
        print(f"Would associate knowledge base from SSM /headset-agent/{args.environment}/kb-id")
        print("Would prepare the agent, publish the live alias, and update SSM parameters")
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

    # Knowledge base ID is mandatory: the agent must be KB-grounded (A-07).
    kb_id = get_kb_id(ssm_client, args.environment)
    if not kb_id:
        print("ERROR: Knowledge base ID not available. Deploy infrastructure (KB stack) first.")
        sys.exit(1)
    print(f"Using knowledge base: {kb_id}")

    # A-09: guardrail is optional — absence is non-fatal, agent runs without it.
    guardrail_id, guardrail_version = get_guardrail_config(ssm_client, args.environment)
    if guardrail_id and guardrail_version:
        print(f"Using guardrail: {guardrail_id} (version {guardrail_version})")
    else:
        print("Guardrail not configured — agent will be created/updated without one")

    # 1. Remove the orphaned sub-agents from the old multi-agent topology.
    orphans_remaining = delete_orphaned_agents(bedrock_client, args.environment)

    # 2. Create/update the single supervisor agent.
    agent_id = create_or_update_agent(
        bedrock_client, SUPERVISOR_AGENT, role_arn, model_id, args.environment,
        guardrail_id=guardrail_id, guardrail_version=guardrail_version)
    if not agent_id:
        print("ERROR: Could not create or update the supervisor agent.")
        sys.exit(1)

    # The agent must exist (not CREATING) before the KB can be associated.
    if wait_for_agent_ready(bedrock_client, agent_id,
                            ['NOT_PREPARED', 'PREPARED'], timeout=120) is None:
        print("ERROR: Supervisor agent never became ready for configuration.")
        sys.exit(1)

    # 3. Associate the knowledge base with DRAFT before preparing so the
    #    prepared version serves it.
    if not associate_knowledge_base(bedrock_client, agent_id, kb_id):
        print("ERROR: Could not associate the knowledge base with the agent.")
        sys.exit(1)

    # 4. Prepare and publish via the live alias.
    status = prepare_agent(bedrock_client, agent_id)
    if status != 'PREPARED':
        print(f"ERROR: Supervisor agent is not prepared (status: {status})")
        sys.exit(1)

    alias_id = ensure_agent_alias(bedrock_client, agent_id, "live", args.environment)

    # 5. Store the parameters the lex-lambda reads.
    store_ssm_parameter(
        ssm_client,
        f"/headset-agent/{args.environment}/supervisor-agent-id",
        agent_id,
        "Bedrock Supervisor Agent ID"
    )
    if alias_id:
        store_ssm_parameter(
            ssm_client,
            f"/headset-agent/{args.environment}/supervisor-agent-alias",
            alias_id,
            "Bedrock Supervisor Agent Alias ID"
        )
    else:
        print("ERROR: Alias was not created — SSM alias parameter left untouched.")
        sys.exit(1)

    # 6. Assert the final topology: one prepared agent, KB attached, no orphans.
    print("\n=== Verifying final topology ===")
    if not assert_single_prepared_agent(bedrock_client, agent_id, kb_id,
                                        args.environment, orphans_remaining):
        print("ERROR: Final topology assertion failed.")
        sys.exit(1)

    print("\n=== Agent Configuration Complete ===")
    print(f"  supervisor: {agent_id} (alias: {alias_id}, knowledge base: {kb_id})")


if __name__ == '__main__':
    main()
