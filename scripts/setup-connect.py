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


def claim_phone_number(client, instance_id, country_code='US', phone_type='DID', description=''):
    """Claim a phone number for the Connect instance"""
    try:
        # Search for available phone numbers
        response = client.search_available_phone_numbers(
            TargetArn=f"arn:aws:connect:us-east-1:{get_account_id()}:instance/{instance_id}",
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
            TargetArn=f"arn:aws:connect:us-east-1:{get_account_id()}:instance/{instance_id}",
            PhoneNumber=phone_number,
            PhoneNumberDescription=description,
            Tags={
                'Environment': 'dev',
                'Project': 'HeadsetSupportAgent'
            }
        )

        print(f"Claimed phone number: {phone_number}")
        return {
            'PhoneNumber': phone_number,
            'PhoneNumberId': claim_response.get('PhoneNumberId'),
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


def associate_phone_with_flow(client, instance_id, phone_number_id, contact_flow_id):
    """Associate phone number with contact flow"""
    try:
        client.update_phone_number(
            PhoneNumberId=phone_number_id,
            TargetArn=f"arn:aws:connect:us-east-1:{get_account_id()}:instance/{instance_id}",
            ContactFlowId=contact_flow_id
        )
        print(f"Associated phone number with contact flow")
        return True
    except ClientError as e:
        print(f"Error associating phone with flow: {e}")
        return False


def save_to_ssm(ssm_client, param_name, value, description=''):
    """Save value to SSM Parameter Store"""
    try:
        ssm_client.put_parameter(
            Name=param_name,
            Value=value,
            Type='String',
            Description=description,
            Overwrite=True,
            Tags=[
                {'Key': 'Environment', 'Value': 'dev'},
                {'Key': 'Project', 'Value': 'HeadsetSupportAgent'}
            ]
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


def main():
    parser = argparse.ArgumentParser(description='Setup Amazon Connect for Headset Support Agent')
    parser.add_argument('--environment', '-e', default='dev', choices=['dev', 'staging', 'prod'])
    parser.add_argument('--region', '-r', default='us-east-1')
    parser.add_argument('--instance-alias', default='headset-support')
    parser.add_argument('--skip-phone-numbers', action='store_true',
                        help='Skip phone number claiming (useful if already claimed)')
    parser.add_argument('--dry-run', action='store_true')

    args = parser.parse_args()

    print(f"=== Amazon Connect Setup ===")
    print(f"Environment: {args.environment}")
    print(f"Region: {args.region}")

    if args.dry_run:
        print("*** DRY RUN - No changes will be made ***")
        return 0

    connect_client = get_connect_client(args.region)
    ssm_client = get_ssm_client(args.region)

    # Step 1: Get or verify Connect instance
    print("\n--- Step 1: Connect Instance ---")
    instance_id = get_or_create_instance(connect_client, args.instance_alias)
    if not instance_id:
        print("WARN: No Connect instance available. Skipping Connect setup.")
        print("      Create a Connect instance in the AWS Console and re-run.")
        return 0  # Don't fail pipeline - Connect instance creation is manual

    # Save instance ID to SSM
    save_to_ssm(ssm_client,
                f"/headset-agent/{args.environment}/connect/instance-id",
                instance_id,
                "Amazon Connect Instance ID")

    # Step 2: Get Lex bot info from SSM/CloudFormation
    print("\n--- Step 2: Get Resource ARNs ---")
    lex_bot_alias_arn = get_ssm_parameter(ssm_client,
                                           f"/headset-agent/{args.environment}/lex-bot-alias-arn")
    if not lex_bot_alias_arn:
        # Try to get from CloudFormation outputs
        cf_client = boto3.client('cloudformation', region_name=args.region)
        try:
            response = cf_client.describe_stacks(StackName=f"agent-headset-{args.environment}")
            for output in response['Stacks'][0].get('Outputs', []):
                if output['OutputKey'] == 'LexBotAliasArn':
                    lex_bot_alias_arn = output['OutputValue']
                    break
        except ClientError:
            pass

    nova_sonic_lambda_arn = get_ssm_parameter(ssm_client,
                                               f"/headset-agent/{args.environment}/nova-sonic-lambda-arn")
    if not nova_sonic_lambda_arn:
        try:
            response = cf_client.describe_stacks(StackName=f"agent-headset-{args.environment}")
            for output in response['Stacks'][0].get('Outputs', []):
                if output['OutputKey'] == 'NovaSonicLambdaFunctionArn':
                    nova_sonic_lambda_arn = output['OutputValue']
                    break
        except ClientError:
            pass

    print(f"Lex Bot Alias ARN: {lex_bot_alias_arn or 'Not found'}")
    print(f"Nova Sonic Lambda ARN: {nova_sonic_lambda_arn or 'Not found'}")

    # Step 3: Associate Lex bot and Lambda with Connect
    print("\n--- Step 3: Associate Resources ---")
    if lex_bot_alias_arn:
        associate_lex_bot(connect_client, instance_id, lex_bot_alias_arn)

    if nova_sonic_lambda_arn:
        associate_lambda(connect_client, instance_id, nova_sonic_lambda_arn)

    # Step 4: Create contact flows
    print("\n--- Step 4: Contact Flows ---")
    lex_flow_id = None
    nova_flow_id = None

    if lex_bot_alias_arn:
        lex_flow_id = create_lex_contact_flow(
            connect_client, instance_id,
            f"HeadsetSupport-Lex-{args.environment}",
            lex_bot_alias_arn,
            None  # Not using Lambda in Lex path directly
        )

    if nova_sonic_lambda_arn:
        nova_flow_id = create_nova_sonic_contact_flow(
            connect_client, instance_id,
            f"HeadsetSupport-NovaSonic-{args.environment}",
            nova_sonic_lambda_arn
        )

    # Step 5: Claim phone numbers (if not skipped)
    print("\n--- Step 5: Phone Numbers ---")
    if not args.skip_phone_numbers:
        # Check if we already have phone numbers
        existing_lex_phone = get_ssm_parameter(ssm_client,
                                                f"/headset-agent/{args.environment}/connect/phone-number-lex")
        existing_nova_phone = get_ssm_parameter(ssm_client,
                                                 f"/headset-agent/{args.environment}/connect/phone-number-nova-sonic")

        if existing_lex_phone and existing_lex_phone != 'PLACEHOLDER':
            print(f"Lex path phone number already claimed: {existing_lex_phone}")
        elif lex_flow_id:
            lex_phone = claim_phone_number(connect_client, instance_id,
                                           description=f"Headset Support - Lex Path ({args.environment})")
            if lex_phone:
                save_to_ssm(ssm_client,
                           f"/headset-agent/{args.environment}/connect/phone-number-lex",
                           lex_phone['PhoneNumber'],
                           "Phone number for Lex path (Path A)")
                # Associate with contact flow
                if lex_phone.get('PhoneNumberId'):
                    associate_phone_with_flow(connect_client, instance_id,
                                             lex_phone['PhoneNumberId'], lex_flow_id)

        if existing_nova_phone and existing_nova_phone != 'PLACEHOLDER':
            print(f"Nova Sonic path phone number already claimed: {existing_nova_phone}")
        elif nova_flow_id:
            nova_phone = claim_phone_number(connect_client, instance_id,
                                            description=f"Headset Support - Nova Sonic Path ({args.environment})")
            if nova_phone:
                save_to_ssm(ssm_client,
                           f"/headset-agent/{args.environment}/connect/phone-number-nova-sonic",
                           nova_phone['PhoneNumber'],
                           "Phone number for Nova Sonic path (Path B)")
                # Associate with contact flow
                if nova_phone.get('PhoneNumberId'):
                    associate_phone_with_flow(connect_client, instance_id,
                                             nova_phone['PhoneNumberId'], nova_flow_id)
    else:
        print("Skipping phone number claiming (--skip-phone-numbers)")

    # Summary
    print("\n=== Connect Setup Summary ===")
    print(f"Instance ID: {instance_id}")
    print(f"Lex Contact Flow: {lex_flow_id or 'Not created'}")
    print(f"Nova Sonic Contact Flow: {nova_flow_id or 'Not created'}")

    lex_phone = get_ssm_parameter(ssm_client, f"/headset-agent/{args.environment}/connect/phone-number-lex")
    nova_phone = get_ssm_parameter(ssm_client, f"/headset-agent/{args.environment}/connect/phone-number-nova-sonic")
    print(f"Lex Phone: {lex_phone or 'Not assigned'}")
    print(f"Nova Sonic Phone: {nova_phone or 'Not assigned'}")

    print("\n=== Setup Complete ===")
    return 0


if __name__ == '__main__':
    sys.exit(main())
