#!/usr/bin/env python3
"""
Nova Sonic Configuration Script for Amazon Lex Bot

This script provides instructions for configuring Nova Sonic (Amazon's speech-to-speech
AI model) with the Lex bot after CloudFormation deployment.

IMPORTANT: Nova Sonic configuration requires MANUAL steps in the Amazon Connect
admin console. CloudFormation and the AWS CLI do not currently support:
- Enabling Nova Sonic on a Lex bot
- Configuring speech-to-speech AI settings
- Setting up voice customization for Nova Sonic

This script retrieves the Lex bot information from CloudFormation outputs or SSM
parameters and provides detailed instructions for manual configuration.
"""

import argparse
import boto3
import os
from botocore.exceptions import ClientError


def get_lex_bot_info_from_cloudformation(environment: str, region: str) -> dict:
    """Get Lex bot information from CloudFormation stack outputs"""
    cf_client = boto3.client('cloudformation', region_name=region)
    stack_name = f"headset-agent-stack-{environment}"

    try:
        response = cf_client.describe_stacks(StackName=stack_name)
        outputs = response['Stacks'][0]['Outputs']

        bot_info = {}
        for output in outputs:
            if output['OutputKey'] == 'LexBotId':
                bot_info['bot_id'] = output['OutputValue']
            elif output['OutputKey'] == 'LexBotAliasId':
                bot_info['alias_id'] = output['OutputValue']
            elif output['OutputKey'] == 'LexBotAliasArn':
                bot_info['alias_arn'] = output['OutputValue']
            elif output['OutputKey'] == 'LexBotName':
                bot_info['bot_name'] = output['OutputValue']
            elif output['OutputKey'] == 'ConnectInstanceId':
                bot_info['connect_instance_id'] = output['OutputValue']

        return bot_info
    except ClientError as e:
        print(f"Warning: Could not get Lex bot info from CloudFormation: {e}")
        return {}


def get_lex_bot_info_from_ssm(environment: str, region: str) -> dict:
    """Get Lex bot information from SSM parameters as fallback"""
    ssm_client = boto3.client('ssm', region_name=region)
    bot_info = {}

    param_mappings = {
        'bot_id': f'/headset-agent/{environment}/lex-bot-id',
        'alias_id': f'/headset-agent/{environment}/lex-bot-alias-id',
        'connect_instance_id': f'/headset-agent/{environment}/connect-instance-id',
    }

    for key, param_name in param_mappings.items():
        try:
            response = ssm_client.get_parameter(Name=param_name)
            bot_info[key] = response['Parameter']['Value']
        except ClientError:
            pass  # Parameter doesn't exist

    return bot_info


def get_lex_bot_details(bot_id: str, region: str) -> dict:
    """Get additional Lex bot details directly from the Lex API"""
    lex_client = boto3.client('lexv2-models', region_name=region)

    try:
        response = lex_client.describe_bot(botId=bot_id)
        return {
            'bot_name': response['botName'],
            'bot_status': response['botStatus'],
        }
    except ClientError as e:
        print(f"Warning: Could not get Lex bot details: {e}")
        return {}


def get_connect_instance_url(instance_id: str, region: str) -> str:
    """Get the Amazon Connect instance access URL"""
    connect_client = boto3.client('connect', region_name=region)

    try:
        response = connect_client.describe_instance(InstanceId=instance_id)
        instance_alias = response['Instance'].get('InstanceAlias', instance_id)
        # The Connect admin console URL format
        return f"https://{instance_alias}.my.connect.aws"
    except ClientError as e:
        print(f"Warning: Could not get Connect instance URL: {e}")
        return f"https://{region}.console.aws.amazon.com/connect/home"


def print_nova_sonic_instructions(bot_info: dict, region: str):
    """Print detailed instructions for configuring Nova Sonic"""

    bot_id = bot_info.get('bot_id', '<BOT_ID>')
    bot_name = bot_info.get('bot_name', 'HeadsetTroubleshooterBot')
    alias_id = bot_info.get('alias_id', '<ALIAS_ID>')
    connect_instance_id = bot_info.get('connect_instance_id', '<INSTANCE_ID>')

    connect_url = get_connect_instance_url(connect_instance_id, region) if connect_instance_id != '<INSTANCE_ID>' else 'https://console.aws.amazon.com/connect'
    lex_console_url = f"https://{region}.console.aws.amazon.com/lexv2/home?region={region}#bot/{bot_id}/overview"

    print(f"""
{'='*70}
  NOVA SONIC CONFIGURATION FOR LEX BOT
{'='*70}

  Bot Name: {bot_name}
  Bot ID: {bot_id}
  Alias ID: {alias_id}
  Region: {region}
  Connect Instance: {connect_instance_id}

{'='*70}

IMPORTANT: Nova Sonic (Amazon's speech-to-speech AI model) configuration
requires MANUAL steps in the Amazon Connect admin console.

CloudFormation and the AWS CLI do not currently support:
  - Enabling Nova Sonic on a Lex bot
  - Configuring speech-to-speech AI settings
  - Setting up voice customization for Nova Sonic

{'='*70}
  MANUAL CONFIGURATION STEPS
{'='*70}

STEP 1: Open Amazon Connect Admin Console
-----------------------------------------
URL: {connect_url}

1. Log in to your Amazon Connect instance
2. Navigate to "Channels" in the left sidebar
3. Select "Phone numbers" to verify your phone configuration

STEP 2: Configure Lex Bot in Amazon Connect
-------------------------------------------
1. In the Connect admin console, go to "Routing" > "Contact flows"
2. Open your "Headset Support" contact flow
3. In the contact flow, find the "Get customer input" block
4. Verify the Lex bot is configured:
   - Bot Name: {bot_name}
   - Bot Alias: prod

STEP 3: Enable Nova Sonic (if available in your region)
--------------------------------------------------------
Note: Nova Sonic availability varies by region. Check AWS documentation
for current availability.

1. In the Amazon Connect admin console, go to "Analytics and optimization"
2. Navigate to "Voice Intelligence" or "AI Services"
3. Look for "Nova Sonic" or "Speech-to-Speech AI" options
4. If available, enable Nova Sonic for your contact flows

STEP 4: Configure Voice Settings for Nova Sonic
------------------------------------------------
If Nova Sonic is enabled:

1. Go to the contact flow editor
2. In the "Get customer input" block, check for Nova Sonic settings
3. Configure speech parameters:
   - Speech recognition sensitivity
   - Response voice (neural voices work best)
   - Conversation timeout settings

STEP 5: Test the Configuration
------------------------------
1. Call your Amazon Connect phone number
2. Verify the voice interaction is smooth and natural
3. Test various headset troubleshooting scenarios
4. Monitor CloudWatch logs for any errors

{'='*70}
  ADDITIONAL RESOURCES
{'='*70}

Lex Console (Bot Configuration):
  {lex_console_url}

Amazon Connect Console:
  {connect_url}

AWS Documentation:
  - Amazon Connect Voice AI: https://docs.aws.amazon.com/connect/latest/adminguide/
  - Amazon Lex V2: https://docs.aws.amazon.com/lexv2/latest/dg/
  - Nova Sonic (when available): Check AWS What's New announcements

{'='*70}
  SSM PARAMETERS REFERENCE
{'='*70}

The following SSM parameters contain your Lex bot information:
  /headset-agent/{bot_info.get('environment', 'dev')}/lex-bot-id
  /headset-agent/{bot_info.get('environment', 'dev')}/lex-bot-alias-id
  /headset-agent/{bot_info.get('environment', 'dev')}/connect-instance-id

{'='*70}
""")


def store_nova_sonic_config_reference(environment: str, region: str, bot_info: dict):
    """Store a reference in SSM indicating Nova Sonic needs manual configuration"""
    ssm_client = boto3.client('ssm', region_name=region)

    try:
        ssm_client.put_parameter(
            Name=f'/headset-agent/{environment}/nova-sonic-status',
            Value='manual-configuration-required',
            Type='String',
            Overwrite=True,
            Description='Nova Sonic configuration status - requires manual setup in Connect admin console'
        )
        print(f"  Updated SSM parameter: /headset-agent/{environment}/nova-sonic-status")
    except ClientError as e:
        print(f"  Warning: Could not update SSM parameter: {e}")


def main():
    parser = argparse.ArgumentParser(
        description='Configure Nova Sonic for Headset Support Lex Bot',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Example usage:
  python configure-nova-sonic.py --environment dev --region us-east-1

Note: This script provides instructions for manual configuration.
Nova Sonic settings cannot be automated via CloudFormation or CLI.
        """
    )
    parser.add_argument(
        '--environment', '-e',
        default='dev',
        choices=['dev', 'staging', 'prod'],
        help='Target environment (default: dev)'
    )
    parser.add_argument(
        '--region', '-r',
        default=os.environ.get('AWS_REGION', 'us-east-1'),
        help='AWS region (default: us-east-1 or AWS_REGION env var)'
    )
    parser.add_argument(
        '--quiet', '-q',
        action='store_true',
        help='Suppress detailed instructions, only show bot info'
    )
    args = parser.parse_args()

    print(f"\n{'='*70}")
    print(f"  NOVA SONIC CONFIGURATION HELPER")
    print(f"  Environment: {args.environment}")
    print(f"  Region: {args.region}")
    print(f"{'='*70}\n")

    # Try to get bot info from CloudFormation first
    print("Retrieving Lex bot information from CloudFormation...")
    bot_info = get_lex_bot_info_from_cloudformation(args.environment, args.region)

    # Fall back to SSM if CloudFormation didn't have all info
    if not bot_info.get('bot_id'):
        print("Trying SSM parameters as fallback...")
        ssm_info = get_lex_bot_info_from_ssm(args.environment, args.region)
        bot_info.update(ssm_info)

    # Get additional details from Lex API if we have the bot ID
    if bot_info.get('bot_id'):
        print(f"Found Lex bot: {bot_info.get('bot_id')}")
        lex_details = get_lex_bot_details(bot_info['bot_id'], args.region)
        bot_info.update(lex_details)
    else:
        print("Warning: Could not find Lex bot ID. Using placeholder values.")

    # Add environment to bot_info for reference
    bot_info['environment'] = args.environment

    # Store reference in SSM
    print("\nUpdating SSM parameters...")
    store_nova_sonic_config_reference(args.environment, args.region, bot_info)

    # Print instructions
    if not args.quiet:
        print_nova_sonic_instructions(bot_info, args.region)
    else:
        print(f"\nLex Bot Information:")
        print(f"  Bot ID: {bot_info.get('bot_id', 'Not found')}")
        print(f"  Bot Name: {bot_info.get('bot_name', 'Not found')}")
        print(f"  Alias ID: {bot_info.get('alias_id', 'Not found')}")
        print(f"  Connect Instance: {bot_info.get('connect_instance_id', 'Not found')}")
        print(f"\nManual Nova Sonic configuration required in Amazon Connect admin console.")

    print("\n" + "="*70)
    print("  Nova Sonic configuration helper complete.")
    print("  Please follow the manual steps above to enable Nova Sonic.")
    print("="*70 + "\n")


if __name__ == "__main__":
    main()
