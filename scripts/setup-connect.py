#!/usr/bin/env python3
"""
Amazon Connect Setup Script
Automates Connect instance configuration, phone number claiming, and contact flow deployment.
Designed to run in GitHub Actions pipeline.
"""

import argparse
import boto3
import json
import time
import sys
from botocore.exceptions import ClientError


def get_connect_client(region):
    """Create Connect client"""
    return boto3.client('connect', region_name=region)


def get_ssm_client(region):
    """Create SSM client"""
    return boto3.client('ssm', region_name=region)


def get_or_create_instance(client, instance_alias):
    """Get existing Connect instance or provide instructions"""
    try:
        # List existing instances
        response = client.list_instances()
        for instance in response.get('InstanceSummaryList', []):
            if instance.get('InstanceAlias') == instance_alias:
                print(f"Found existing Connect instance: {instance['Id']}")
                return instance['Id']

        # Check if any instance exists (use first one for dev)
        if response.get('InstanceSummaryList'):
            instance = response['InstanceSummaryList'][0]
            print(f"Using existing Connect instance: {instance['Id']} ({instance.get('InstanceAlias', 'unnamed')})")
            return instance['Id']

        print("ERROR: No Amazon Connect instance found.")
        print("Amazon Connect instances must be created manually in the AWS Console")
        print("due to identity management and security requirements.")
        print("")
        print("To create an instance:")
        print("1. Go to Amazon Connect console")
        print("2. Click 'Create instance'")
        print("3. Follow the setup wizard")
        print("4. Re-run this pipeline after instance creation")
        return None

    except ClientError as e:
        print(f"Error listing Connect instances: {e}")
        return None


def wait_for_phone_number_ready(client, phone_number_id, timeout=120):
    """Wait for phone number to be in CLAIMED status (ready for use)"""
    print(f"  Waiting for phone number to be provisioned...")
    start_time = time.time()

    while time.time() - start_time < timeout:
        try:
            response = client.describe_phone_number(PhoneNumberId=phone_number_id)
            status = response.get('ClaimedPhoneNumberSummary', {}).get('PhoneNumberStatus', {})
            status_value = status.get('Status', 'UNKNOWN')
            status_message = status.get('Message', '')

            if status_value == 'CLAIMED':
                print(f"  Phone number is ready (status: CLAIMED)")
                return True
            elif status_value in ['FAILED', 'CANCELLED']:
                print(f"  Phone number provisioning failed: {status_value}")
                if 'limit' in status_message.lower():
                    print("")
                    print("  ⚠️  PHONE NUMBER QUOTA ISSUE DETECTED")
                    print("  This is a known AWS issue. Even with 0 phone numbers,")
                    print("  the limit may be exceeded. Resolution requires AWS Support.")
                    print("")
                    print("  To resolve:")
                    print("  1. Go to AWS Console > Service Quotas > Amazon Connect")
                    print("  2. Search for 'Phone numbers per instance'")
                    print("  3. Request a quota increase for your Connect instance")
                    print("  4. Or open an AWS Support case for faster resolution")
                    print("")
                return False
            else:
                print(f"  Status: {status_value}, waiting...")
                time.sleep(5)
        except ClientError as e:
            print(f"  Error checking status: {e}")
            time.sleep(5)

    print(f"  Timeout waiting for phone number to be ready")
    return False


def claim_phone_number(client, instance_id, country_code='US', phone_type='DID', description=''):
    """Claim a phone number for the Connect instance"""
    try:
        # Determine the target ARN - instance_id might already be an ARN
        if instance_id.startswith('arn:'):
            target_arn = instance_id
        else:
            target_arn = f"arn:aws:connect:us-east-1:{get_account_id()}:instance/{instance_id}"

        # Search for available phone numbers
        response = client.search_available_phone_numbers(
            TargetArn=target_arn,
            PhoneNumberCountryCode=country_code,
            PhoneNumberType=phone_type,
            MaxResults=1
        )

        available = response.get('AvailableNumbersList', [])
        if not available:
            print(f"No available {phone_type} phone numbers in {country_code}")
            return None

        phone_number = available[0]['PhoneNumber']
        phone_number_id = available[0].get('PhoneNumberId')

        # Claim the phone number
        claim_response = client.claim_phone_number(
            TargetArn=target_arn,
            PhoneNumber=phone_number,
            PhoneNumberDescription=description,
            Tags={
                'Environment': 'dev',
                'Project': 'HeadsetSupportAgent'
            }
        )

        claimed_phone_number_id = claim_response.get('PhoneNumberId')
        print(f"Claimed phone number: {phone_number} (ID: {claimed_phone_number_id})")

        # Wait for phone number to be fully provisioned
        if claimed_phone_number_id:
            if not wait_for_phone_number_ready(client, claimed_phone_number_id):
                print(f"  WARNING: Phone number may not be fully provisioned")

        return {
            'PhoneNumber': phone_number,
            'PhoneNumberId': claimed_phone_number_id,
            'PhoneNumberArn': claim_response.get('PhoneNumberArn')
        }

    except ClientError as e:
        if 'ResourceNotFoundException' in str(e):
            print(f"No phone numbers available or instance not found")
        else:
            print(f"Error claiming phone number: {e}")
        return None


def get_account_id():
    """Get current AWS account ID"""
    sts = boto3.client('sts')
    return sts.get_caller_identity()['Account']


def list_contact_flows(client, instance_id):
    """List existing contact flows"""
    try:
        response = client.list_contact_flows(
            InstanceId=instance_id,
            ContactFlowTypes=['CONTACT_FLOW']
        )
        return response.get('ContactFlowSummaryList', [])
    except ClientError as e:
        print(f"Error listing contact flows: {e}")
        return []


def create_lex_contact_flow(client, instance_id, flow_name, lex_bot_alias_arn, lambda_arn):
    """Create a contact flow for Lex path"""

    # Contact flow content for Lex integration
    flow_content = {
        "Version": "2019-10-30",
        "StartAction": "SetVoice",
        "Actions": [
            {
                "Identifier": "SetVoice",
                "Type": "UpdateContactData",
                "Parameters": {
                    "TextToSpeechVoice": "Joanna"
                },
                "Transitions": {
                    "NextAction": "PlayWelcome"
                }
            },
            {
                "Identifier": "PlayWelcome",
                "Type": "MessageParticipant",
                "Parameters": {
                    "Text": "Welcome to Headset Support. I'm here to help you troubleshoot your headset issues."
                },
                "Transitions": {
                    "NextAction": "GetCustomerInput"
                }
            },
            {
                "Identifier": "GetCustomerInput",
                "Type": "ConnectParticipantWithLexBot",
                "Parameters": {
                    "LexBot": {
                        "AliasArn": lex_bot_alias_arn
                    },
                    "LexSessionAttributes": {}
                },
                "Transitions": {
                    "NextAction": "CheckIntent",
                    "Errors": [
                        {
                            "NextAction": "TransferToQueue",
                            "ErrorType": "NoMatchingError"
                        }
                    ]
                }
            },
            {
                "Identifier": "CheckIntent",
                "Type": "CheckAttribute",
                "Parameters": {
                    "Attribute": "$.Lex.IntentResult.IntentName",
                    "Values": ["TroubleshootIntent"]
                },
                "Transitions": {
                    "NextAction": "GetCustomerInput",
                    "Conditions": [],
                    "Errors": [
                        {
                            "NextAction": "EndCall",
                            "ErrorType": "NoMatchingCondition"
                        }
                    ]
                }
            },
            {
                "Identifier": "TransferToQueue",
                "Type": "TransferToQueue",
                "Parameters": {},
                "Transitions": {
                    "NextAction": "EndCall",
                    "Errors": [
                        {
                            "NextAction": "EndCall",
                            "ErrorType": "NoMatchingError"
                        }
                    ]
                }
            },
            {
                "Identifier": "EndCall",
                "Type": "DisconnectParticipant",
                "Parameters": {},
                "Transitions": {}
            }
        ]
    }

    try:
        response = client.create_contact_flow(
            InstanceId=instance_id,
            Name=flow_name,
            Type='CONTACT_FLOW',
            Description='Headset troubleshooting flow using Lex bot (Path A)',
            Content=json.dumps(flow_content),
            Tags={
                'Environment': 'dev',
                'Project': 'HeadsetSupportAgent',
                'Path': 'Lex'
            }
        )
        print(f"Created contact flow: {flow_name}")
        return response.get('ContactFlowId')
    except ClientError as e:
        if 'DuplicateResourceException' in str(e):
            print(f"Contact flow {flow_name} already exists")
            # Find existing flow
            flows = list_contact_flows(client, instance_id)
            for flow in flows:
                if flow['Name'] == flow_name:
                    return flow['Id']
        else:
            print(f"Error creating contact flow: {e}")
        return None


def create_nova_sonic_contact_flow(client, instance_id, flow_name, lambda_arn):
    """Create a contact flow for Nova Sonic path"""

    # Contact flow for Nova Sonic (invokes Lambda for streaming)
    flow_content = {
        "Version": "2019-10-30",
        "StartAction": "SetVoice",
        "Actions": [
            {
                "Identifier": "SetVoice",
                "Type": "UpdateContactData",
                "Parameters": {
                    "TextToSpeechVoice": "Joanna"
                },
                "Transitions": {
                    "NextAction": "PlayWelcome"
                }
            },
            {
                "Identifier": "PlayWelcome",
                "Type": "MessageParticipant",
                "Parameters": {
                    "Text": "Welcome to Headset Support with Nova Sonic. How can I help you today?"
                },
                "Transitions": {
                    "NextAction": "InvokeLambda"
                }
            },
            {
                "Identifier": "InvokeLambda",
                "Type": "InvokeLambdaFunction",
                "Parameters": {
                    "LambdaFunctionARN": lambda_arn,
                    "InvocationTimeLimitSeconds": "8",
                    "LambdaInvocationAttributes": {
                        "action": "start",
                        "path": "nova-sonic"
                    }
                },
                "Transitions": {
                    "NextAction": "CheckLambdaResponse",
                    "Errors": [
                        {
                            "NextAction": "HandleError",
                            "ErrorType": "NoMatchingError"
                        }
                    ]
                }
            },
            {
                "Identifier": "CheckLambdaResponse",
                "Type": "CheckAttribute",
                "Parameters": {
                    "Attribute": "$.External.success",
                    "Values": ["true"]
                },
                "Transitions": {
                    "NextAction": "ContinueConversation",
                    "Conditions": [],
                    "Errors": [
                        {
                            "NextAction": "HandleError",
                            "ErrorType": "NoMatchingCondition"
                        }
                    ]
                }
            },
            {
                "Identifier": "ContinueConversation",
                "Type": "Loop",
                "Parameters": {
                    "LoopCount": "10"
                },
                "Transitions": {
                    "NextAction": "InvokeLambda",
                    "Conditions": [
                        {
                            "NextAction": "EndCall",
                            "Condition": {
                                "Operand": "$.LoopCount",
                                "Operator": "Equals",
                                "Value": "0"
                            }
                        }
                    ]
                }
            },
            {
                "Identifier": "HandleError",
                "Type": "MessageParticipant",
                "Parameters": {
                    "Text": "I apologize, but I'm having trouble processing your request. Let me transfer you to an agent."
                },
                "Transitions": {
                    "NextAction": "TransferToQueue"
                }
            },
            {
                "Identifier": "TransferToQueue",
                "Type": "TransferToQueue",
                "Parameters": {},
                "Transitions": {
                    "NextAction": "EndCall",
                    "Errors": [
                        {
                            "NextAction": "EndCall",
                            "ErrorType": "NoMatchingError"
                        }
                    ]
                }
            },
            {
                "Identifier": "EndCall",
                "Type": "DisconnectParticipant",
                "Parameters": {},
                "Transitions": {}
            }
        ]
    }

    try:
        response = client.create_contact_flow(
            InstanceId=instance_id,
            Name=flow_name,
            Type='CONTACT_FLOW',
            Description='Headset troubleshooting flow using Nova Sonic (Path B)',
            Content=json.dumps(flow_content),
            Tags={
                'Environment': 'dev',
                'Project': 'HeadsetSupportAgent',
                'Path': 'NovaSonic'
            }
        )
        print(f"Created contact flow: {flow_name}")
        return response.get('ContactFlowId')
    except ClientError as e:
        if 'DuplicateResourceException' in str(e):
            print(f"Contact flow {flow_name} already exists")
            flows = list_contact_flows(client, instance_id)
            for flow in flows:
                if flow['Name'] == flow_name:
                    return flow['Id']
        else:
            print(f"Error creating contact flow: {e}")
        return None


def associate_lex_bot(client, instance_id, lex_bot_alias_arn):
    """Associate Lex bot with Connect instance"""
    try:
        client.associate_lex_bot(
            InstanceId=instance_id,
            LexBot={
                'LexRegion': 'us-east-1'
            },
            LexV2Bot={
                'AliasArn': lex_bot_alias_arn
            }
        )
        print(f"Associated Lex bot with Connect instance")
        return True
    except ClientError as e:
        if 'ResourceExistsException' in str(e) or 'DuplicateResourceException' in str(e):
            print("Lex bot already associated with Connect instance")
            return True
        print(f"Error associating Lex bot: {e}")
        return False


def associate_lambda(client, instance_id, lambda_arn):
    """Associate Lambda function with Connect instance"""
    try:
        client.associate_lambda_function(
            InstanceId=instance_id,
            FunctionArn=lambda_arn
        )
        print(f"Associated Lambda function with Connect instance")
        return True
    except ClientError as e:
        if 'ResourceExistsException' in str(e) or 'DuplicateResourceException' in str(e):
            print("Lambda function already associated with Connect instance")
            return True
        print(f"Error associating Lambda: {e}")
        return False


def associate_phone_with_flow(client, instance_id, phone_number_id, contact_flow_id, max_retries=3):
    """Associate phone number with contact flow with retry logic"""
    # Extract instance ID from ARN if needed
    if instance_id.startswith('arn:'):
        # Extract just the instance ID from the ARN
        # ARN format: arn:aws:connect:region:account:instance/instance-id
        instance_id_only = instance_id.split('/')[-1]
    else:
        instance_id_only = instance_id

    for attempt in range(max_retries):
        try:
            # Use associate_phone_number_contact_flow API
            client.associate_phone_number_contact_flow(
                PhoneNumberId=phone_number_id,
                InstanceId=instance_id_only,
                ContactFlowId=contact_flow_id
            )
            print(f"Associated phone number with contact flow")
            return True
        except ClientError as e:
            if 'ResourceNotFoundException' in str(e) and attempt < max_retries - 1:
                print(f"  Phone number not ready yet, retrying in 10 seconds... (attempt {attempt + 1}/{max_retries})")
                time.sleep(10)
            else:
                print(f"Error associating phone with flow: {e}")
                return False
    return False


def save_to_ssm(ssm_client, param_name, value, description=''):
    """Save value to SSM Parameter Store"""
    try:
        # Cannot use tags with overwrite - update existing parameter
        ssm_client.put_parameter(
            Name=param_name,
            Value=value,
            Type='String',
            Description=description,
            Overwrite=True
        )
        print(f"Saved parameter: {param_name}")
        return True
    except ClientError as e:
        print(f"Error saving SSM parameter: {e}")
        return False


def get_ssm_parameter(ssm_client, param_name):
    """Get value from SSM Parameter Store"""
    try:
        response = ssm_client.get_parameter(Name=param_name)
        return response['Parameter']['Value']
    except ClientError:
        return None


def get_contact_flow_id_by_name(client, instance_id, flow_name):
    """Get contact flow ID by name"""
    try:
        flows = list_contact_flows(client, instance_id)
        for flow in flows:
            if flow['Name'] == flow_name:
                return flow['Id']
    except ClientError as e:
        print(f"Error finding contact flow {flow_name}: {e}")
    return None


def verify_phone_number_exists(client, instance_id, phone_number):
    """Verify if a phone number is claimed and active in Connect"""
    try:
        # Get target ARN
        if instance_id.startswith('arn:'):
            target_arn = instance_id
        else:
            target_arn = f"arn:aws:connect:us-east-1:{get_account_id()}:instance/{instance_id}"

        response = client.list_phone_numbers_v2(TargetArn=target_arn)

        for phone in response.get('ListPhoneNumbersSummaryList', []):
            if phone.get('PhoneNumber') == phone_number:
                return True
        return False
    except ClientError as e:
        print(f"Error verifying phone number: {e}")
        return False


def main():
    parser = argparse.ArgumentParser(description='Setup Amazon Connect for Headset Support Agent')
    parser.add_argument('--environment', '-e', default='dev', choices=['dev', 'staging', 'prod'])
    parser.add_argument('--region', '-r', default='us-east-1')
    parser.add_argument('--skip-phone-numbers', action='store_true',
                        help='Skip phone number claiming (useful if already claimed)')
    parser.add_argument('--dry-run', action='store_true')

    args = parser.parse_args()

    print(f"=== Amazon Connect Setup (Phone Number Claiming) ===")
    print(f"Environment: {args.environment}")
    print(f"Region: {args.region}")

    if args.dry_run:
        print("*** DRY RUN - No changes will be made ***")
        return 0

    connect_client = get_connect_client(args.region)
    ssm_client = get_ssm_client(args.region)

    # Step 1: Get Connect instance - prefer querying Connect API directly for ACTIVE instances
    print("\n--- Step 1: Get Connect Instance ---")

    # First try to find an ACTIVE instance directly from Connect API
    instance_id = None
    try:
        response = connect_client.list_instances()
        for instance in response.get('InstanceSummaryList', []):
            if instance.get('InstanceStatus') == 'ACTIVE':
                instance_id = instance['Id']
                print(f"Found ACTIVE Connect instance: {instance_id} (alias: {instance.get('InstanceAlias', 'none')})")
                break
    except Exception as e:
        print(f"Error listing Connect instances: {e}")

    # Fallback to SSM parameter if no active instance found via API
    if not instance_id:
        instance_id = get_ssm_parameter(ssm_client, f"/headset-agent/{args.environment}/connect/instance-id")
        if instance_id:
            print(f"Using Connect instance from SSM: {instance_id}")

    if not instance_id:
        print("WARN: No active Connect instance found.")
        print("      The SAM stack must complete successfully first.")
        return 0  # Don't fail - CloudFormation might still be running

    print(f"Connect Instance ID: {instance_id}")

    # Step 2: Get contact flow IDs from Connect (created by CloudFormation)
    print("\n--- Step 2: Get Contact Flows ---")
    lex_flow_id = get_contact_flow_id_by_name(
        connect_client, instance_id, f"HeadsetSupport-Lex-{args.environment}"
    )
    nova_flow_id = get_contact_flow_id_by_name(
        connect_client, instance_id, f"HeadsetSupport-NovaSonic-{args.environment}"
    )

    print(f"Lex Contact Flow ID: {lex_flow_id or 'Not found'}")
    print(f"Nova Sonic Contact Flow ID: {nova_flow_id or 'Not found'}")

    # Step 3: Claim phone numbers and associate with flows
    print("\n--- Step 3: Phone Numbers ---")
    if args.skip_phone_numbers:
        print("Skipping phone number claiming (--skip-phone-numbers)")
    else:
        # Check existing phone numbers
        existing_lex_phone = get_ssm_parameter(
            ssm_client, f"/headset-agent/{args.environment}/connect/phone-number-lex"
        )
        existing_nova_phone = get_ssm_parameter(
            ssm_client, f"/headset-agent/{args.environment}/connect/phone-number-nova-sonic"
        )

        # Claim phone for Lex path
        if existing_lex_phone and existing_lex_phone not in ['PLACEHOLDER', 'PENDING']:
            # Verify the phone number actually exists in Connect
            if verify_phone_number_exists(connect_client, instance_id, existing_lex_phone):
                print(f"Lex path phone number already claimed and verified: {existing_lex_phone}")
            else:
                print(f"Lex path phone number {existing_lex_phone} no longer exists in Connect, reclaiming...")
                existing_lex_phone = None  # Force reclaim

        if not existing_lex_phone and lex_flow_id:
            print("Claiming phone number for Lex path...")
            lex_phone = claim_phone_number(
                connect_client, instance_id,
                phone_type='TOLL_FREE',
                description=f"Headset Support - Lex Path ({args.environment})"
            )
            if lex_phone:
                save_to_ssm(
                    ssm_client,
                    f"/headset-agent/{args.environment}/connect/phone-number-lex",
                    lex_phone['PhoneNumber'],
                    "Phone number for Lex path (Path A)"
                )
                # Associate with contact flow
                if lex_phone.get('PhoneNumberId'):
                    associate_phone_with_flow(
                        connect_client, instance_id,
                        lex_phone['PhoneNumberId'], lex_flow_id
                    )
            else:
                print("Failed to claim Lex phone number - may need manual claiming")
        else:
            print("Lex contact flow not found - skipping phone number")

        # Claim phone for Nova Sonic path
        if existing_nova_phone and existing_nova_phone not in ['PLACEHOLDER', 'PENDING']:
            # Verify the phone number actually exists in Connect
            if verify_phone_number_exists(connect_client, instance_id, existing_nova_phone):
                print(f"Nova Sonic path phone number already claimed and verified: {existing_nova_phone}")
            else:
                print(f"Nova Sonic path phone number {existing_nova_phone} no longer exists in Connect, reclaiming...")
                existing_nova_phone = None  # Force reclaim

        if not existing_nova_phone and nova_flow_id:
            print("Claiming phone number for Nova Sonic path...")
            nova_phone = claim_phone_number(
                connect_client, instance_id,
                phone_type='TOLL_FREE',
                description=f"Headset Support - Nova Sonic Path ({args.environment})"
            )
            if nova_phone:
                save_to_ssm(
                    ssm_client,
                    f"/headset-agent/{args.environment}/connect/phone-number-nova-sonic",
                    nova_phone['PhoneNumber'],
                    "Phone number for Nova Sonic path (Path B)"
                )
                # Associate with contact flow
                if nova_phone.get('PhoneNumberId'):
                    associate_phone_with_flow(
                        connect_client, instance_id,
                        nova_phone['PhoneNumberId'], nova_flow_id
                    )
            else:
                print("Failed to claim Nova Sonic phone number - may need manual claiming")
        else:
            print("Nova Sonic contact flow not found - skipping phone number")

    # Summary
    print("\n=== Connect Setup Summary ===")
    print(f"Instance ID: {instance_id}")
    print(f"Lex Contact Flow: {lex_flow_id or 'Not found'}")
    print(f"Nova Sonic Contact Flow: {nova_flow_id or 'Not found'}")

    lex_phone = get_ssm_parameter(ssm_client, f"/headset-agent/{args.environment}/connect/phone-number-lex")
    nova_phone = get_ssm_parameter(ssm_client, f"/headset-agent/{args.environment}/connect/phone-number-nova-sonic")
    print(f"Lex Phone: {lex_phone if lex_phone not in ['PLACEHOLDER', 'PENDING', None] else 'Not assigned'}")
    print(f"Nova Sonic Phone: {nova_phone if nova_phone not in ['PLACEHOLDER', 'PENDING', None] else 'Not assigned'}")

    print("\n=== Setup Complete ===")
    return 0


if __name__ == '__main__':
    sys.exit(main())
