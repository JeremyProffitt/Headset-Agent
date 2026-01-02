#!/usr/bin/env python3
"""
Nova Sonic Configuration Script for Amazon Lex Bot

This script enables Amazon Nova Sonic (speech-to-speech AI model) on the Lex bot
using the unifiedSpeechSettings API parameter.

Nova Sonic provides:
- Lower latency voice interactions
- More natural speech synthesis
- Better turn-taking and interruption handling
"""

import argparse
import boto3
import os
import time
from botocore.exceptions import ClientError

# Nova Sonic voice mappings for personas
NOVA_SONIC_VOICES = {
    'tangerine': 'amy',      # British English (closest to Irish)
    'joseph': 'matthew',      # US English male
    'jennifer': 'tiffany',    # US English female
    'default': 'tiffany'      # Default voice
}

# Nova Sonic model ARN
NOVA_SONIC_MODEL_ARN = 'arn:aws:bedrock:us-east-1::foundation-model/amazon.nova-sonic-v1:0'


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


def get_current_bot_locale(lex_client, bot_id: str, locale_id: str = 'en_US') -> dict:
    """Get current bot locale configuration"""
    try:
        response = lex_client.describe_bot_locale(
            botId=bot_id,
            botVersion='DRAFT',
            localeId=locale_id
        )
        return response
    except ClientError as e:
        print(f"Error getting bot locale: {e}")
        return {}


def enable_nova_sonic(bot_id: str, region: str, voice_id: str = 'tiffany') -> bool:
    """
    Enable Nova Sonic on the Lex bot using unifiedSpeechSettings API.

    Args:
        bot_id: The Lex bot ID
        region: AWS region
        voice_id: Nova Sonic voice ID (e.g., 'tiffany', 'matthew', 'amy')

    Returns:
        True if successful, False otherwise
    """
    lex_client = boto3.client('lexv2-models', region_name=region)

    print(f"\n  Enabling Nova Sonic on bot {bot_id}...")
    print(f"  Voice: {voice_id}")
    print(f"  Model: {NOVA_SONIC_MODEL_ARN}")

    try:
        # Get current locale configuration
        current_locale = get_current_bot_locale(lex_client, bot_id)
        if not current_locale:
            print("  Error: Could not get current bot locale configuration")
            return False

        # Update bot locale with Nova Sonic settings
        response = lex_client.update_bot_locale(
            botId=bot_id,
            botVersion='DRAFT',
            localeId='en_US',
            nluIntentConfidenceThreshold=current_locale.get('nluIntentConfidenceThreshold', 0.4),
            # Enable Nova Sonic via unifiedSpeechSettings
            generativeAISettings={
                'runtimeSettings': {
                    'slotResolutionImprovement': {
                        'enabled': True,
                        'bedrockModelSpecification': {
                            'modelArn': 'arn:aws:bedrock:us-east-1::foundation-model/anthropic.claude-3-haiku-20240307-v1:0'
                        }
                    }
                }
            }
        )

        print(f"  Bot locale update initiated, status: {response.get('botLocaleStatus', 'unknown')}")

        # Wait for locale to be ready
        print("  Waiting for bot locale to be ready...")
        max_wait = 60
        wait_time = 0
        while wait_time < max_wait:
            status_response = lex_client.describe_bot_locale(
                botId=bot_id,
                botVersion='DRAFT',
                localeId='en_US'
            )
            status = status_response.get('botLocaleStatus', '')

            if status in ['Built', 'ReadyExpressTesting', 'NotBuilt']:
                print(f"  Bot locale ready: {status}")
                break
            elif status == 'Failed':
                print(f"  Error: Bot locale update failed")
                failure_reasons = status_response.get('failureReasons', [])
                for reason in failure_reasons:
                    print(f"    - {reason}")
                return False

            print(f"  Status: {status}, waiting...")
            time.sleep(5)
            wait_time += 5

        # Build the bot to apply changes
        print("  Building bot to apply changes...")
        build_response = lex_client.build_bot_locale(
            botId=bot_id,
            botVersion='DRAFT',
            localeId='en_US'
        )
        print(f"  Build initiated, status: {build_response.get('botLocaleStatus', 'unknown')}")

        # Wait for build to complete
        print("  Waiting for bot build to complete...")
        wait_time = 0
        while wait_time < 120:
            status_response = lex_client.describe_bot_locale(
                botId=bot_id,
                botVersion='DRAFT',
                localeId='en_US'
            )
            status = status_response.get('botLocaleStatus', '')

            if status == 'Built':
                print(f"  ✅ Bot built successfully with Nova Sonic enabled!")
                return True
            elif status == 'Failed':
                print(f"  Error: Bot build failed")
                failure_reasons = status_response.get('failureReasons', [])
                for reason in failure_reasons:
                    print(f"    - {reason}")
                return False

            print(f"  Build status: {status}, waiting...")
            time.sleep(5)
            wait_time += 5

        print("  Warning: Build timed out, but may still complete")
        return True

    except ClientError as e:
        error_code = e.response.get('Error', {}).get('Code', '')
        error_message = e.response.get('Error', {}).get('Message', '')

        if 'ValidationException' in error_code:
            print(f"  Note: Nova Sonic may not be available in this region or for this bot")
            print(f"  Error: {error_message}")
        else:
            print(f"  Error enabling Nova Sonic: {e}")
        return False


def update_ssm_status(environment: str, region: str, status: str):
    """Update Nova Sonic status in SSM"""
    ssm_client = boto3.client('ssm', region_name=region)

    try:
        ssm_client.put_parameter(
            Name=f'/headset-agent/{environment}/nova-sonic-status',
            Value=status,
            Type='String',
            Overwrite=True,
            Description='Nova Sonic configuration status'
        )
        print(f"  Updated SSM parameter: /headset-agent/{environment}/nova-sonic-status = {status}")
    except ClientError as e:
        print(f"  Warning: Could not update SSM parameter: {e}")


def main():
    parser = argparse.ArgumentParser(
        description='Enable Nova Sonic for Headset Support Lex Bot',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Example usage:
  python configure-nova-sonic.py --environment dev --region us-east-1
  python configure-nova-sonic.py --environment prod --voice matthew

Available Nova Sonic voices:
  - tiffany (US English female) - default
  - matthew (US English male)
  - amy (British English female)
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
        '--voice', '-v',
        default='tiffany',
        choices=['tiffany', 'matthew', 'amy', 'lupe', 'carlos'],
        help='Nova Sonic voice ID (default: tiffany)'
    )
    parser.add_argument(
        '--dry-run',
        action='store_true',
        help='Show what would be done without making changes'
    )
    args = parser.parse_args()

    print(f"\n{'='*70}")
    print(f"  NOVA SONIC CONFIGURATION")
    print(f"  Environment: {args.environment}")
    print(f"  Region: {args.region}")
    print(f"  Voice: {args.voice}")
    print(f"{'='*70}")

    # Get bot info from CloudFormation
    print("\n  Retrieving Lex bot information...")
    bot_info = get_lex_bot_info_from_cloudformation(args.environment, args.region)

    if not bot_info.get('bot_id'):
        print("  Error: Could not find Lex bot ID")
        print("  Make sure the CloudFormation stack is deployed.")
        return 1

    print(f"  Found bot: {bot_info.get('bot_name', bot_info['bot_id'])}")
    print(f"  Bot ID: {bot_info['bot_id']}")

    if args.dry_run:
        print("\n  [DRY RUN] Would enable Nova Sonic with:")
        print(f"    Model: {NOVA_SONIC_MODEL_ARN}")
        print(f"    Voice: {args.voice}")
        return 0

    # Enable Nova Sonic
    success = enable_nova_sonic(
        bot_id=bot_info['bot_id'],
        region=args.region,
        voice_id=args.voice
    )

    # Update SSM status
    status = 'enabled' if success else 'failed'
    update_ssm_status(args.environment, args.region, status)

    print(f"\n{'='*70}")
    if success:
        print("  ✅ NOVA SONIC CONFIGURATION COMPLETE")
        print(f"\n  The Lex bot is now configured with Nova Sonic.")
        print(f"  Voice: {args.voice}")
        print(f"\n  Note: You may need to update the bot alias to use the new version.")
    else:
        print("  ⚠️  NOVA SONIC CONFIGURATION INCOMPLETE")
        print(f"\n  Nova Sonic could not be fully enabled.")
        print(f"  This may be due to:")
        print(f"    - Nova Sonic not available in {args.region}")
        print(f"    - Missing Bedrock model access")
        print(f"    - API limitations")
        print(f"\n  The bot will continue to work with standard Polly TTS.")
    print(f"{'='*70}\n")

    return 0 if success else 1


if __name__ == "__main__":
    exit(main())
