#!/usr/bin/env python3
"""
Configure Amazon Nova Sonic for Lex V2 Bot
Enables speech-to-speech capabilities with Nova Sonic voices
"""

import argparse
import boto3
import json
import time
from botocore.exceptions import ClientError

# Nova Sonic voice mappings for personas
PERSONA_VOICES = {
    "tangerine": {
        "nova_sonic": "amy",      # British female (closest to Irish)
        "polly": "Niamh",
        "language": "en-GB"
    },
    "joseph": {
        "nova_sonic": "matthew",  # US male
        "polly": "Matthew",
        "language": "en-US"
    },
    "jennifer": {
        "nova_sonic": "tiffany",  # US female
        "polly": "Joanna",
        "language": "en-US"
    }
}

# Nova Sonic supported voices
NOVA_SONIC_VOICES = {
    "en-US": ["tiffany", "matthew"],
    "en-GB": ["amy"],
    "fr-FR": ["ambre", "florian"],
    "de-DE": ["greta", "lennart"],
    "es-ES": ["lupe", "carlos"],
    "it-IT": ["beatrice", "lorenzo"]
}


def get_lex_client(region):
    """Create Lex V2 client"""
    return boto3.client('lexv2-models', region_name=region)


def get_bot_id(client, bot_name):
    """Get bot ID by name"""
    try:
        response = client.list_bots(maxResults=50)
        for bot in response.get('botSummaries', []):
                if bot['botName'] == bot_name:
                    return bot['botId']
    except ClientError as e:
        print(f"Error listing bots: {e}")
    return None


def get_bot_locale(client, bot_id, locale_id='en_US'):
    """Get bot locale configuration"""
    try:
        response = client.describe_bot_locale(
            botId=bot_id,
            botVersion='DRAFT',
            localeId=locale_id
        )
        return response
    except ClientError as e:
        print(f"Error getting bot locale: {e}")
        return None


def update_bot_locale_voice(client, bot_id, locale_id, voice_id, engine='generative'):
    """Update bot locale with Nova Sonic voice settings"""
    try:
        # Get current locale config
        locale = get_bot_locale(client, bot_id, locale_id)
        if not locale:
            return False

        print(f"Updating bot locale {locale_id} with voice: {voice_id} (engine: {engine})")

        # Update the locale with new voice settings
        response = client.update_bot_locale(
            botId=bot_id,
            botVersion='DRAFT',
            localeId=locale_id,
            nluIntentConfidenceThreshold=locale.get('nluIntentConfidenceThreshold', 0.4),
            voiceSettings={
                'voiceId': voice_id,
                'engine': engine
            }
        )

        return response['botLocaleStatus'] in ['Creating', 'Building', 'Built', 'ReadyExpressTesting']

    except ClientError as e:
        print(f"Error updating bot locale: {e}")
        return False


def build_bot_locale(client, bot_id, locale_id='en_US'):
    """Build the bot locale after updates"""
    try:
        print(f"Building bot locale {locale_id}...")
        client.build_bot_locale(
            botId=bot_id,
            botVersion='DRAFT',
            localeId=locale_id
        )
        return True
    except ClientError as e:
        print(f"Error building bot locale: {e}")
        return False


def wait_for_bot_locale(client, bot_id, locale_id='en_US', timeout=300):
    """Wait for bot locale to be built"""
    print(f"Waiting for bot locale {locale_id} to be ready...")
    start_time = time.time()

    while time.time() - start_time < timeout:
        try:
            response = client.describe_bot_locale(
                botId=bot_id,
                botVersion='DRAFT',
                localeId=locale_id
            )
            status = response['botLocaleStatus']
            print(f"  Bot locale status: {status}")

            if status in ['Built', 'ReadyExpressTesting']:
                return True
            elif status == 'Failed':
                print(f"  Build failed: {response.get('failureReasons', 'Unknown')}")
                return False

        except ClientError as e:
            print(f"  Error checking status: {e}")

        time.sleep(10)

    print("Timeout waiting for bot locale")
    return False


def configure_nova_sonic_for_connect(connect_client, instance_id, bot_id, bot_alias_id):
    """Configure Nova Sonic for Amazon Connect integration"""
    try:
        # Note: This requires Connect admin console configuration
        # Nova Sonic is enabled per-locale in the Lex bot and Connect flow
        print("Note: Nova Sonic must also be enabled in Amazon Connect admin console:")
        print(f"  1. Go to Connect instance: {instance_id}")
        print(f"  2. Navigate to Bots > Configuration")
        print(f"  3. Select locale and set Model type to 'Speech-to-Speech'")
        print(f"  4. Set Voice provider to 'Amazon Nova Sonic'")
        return True
    except Exception as e:
        print(f"Error: {e}")
        return False


def main():
    parser = argparse.ArgumentParser(description='Configure Nova Sonic for Lex Bot')
    parser.add_argument('--environment', '-e', default='dev', choices=['dev', 'staging', 'prod'],
                        help='Deployment environment')
    parser.add_argument('--region', '-r', default='us-east-1', help='AWS region')
    parser.add_argument('--bot-name', '-b', default='HeadsetTroubleshooterBot',
                        help='Lex bot name (without environment suffix)')
    parser.add_argument('--persona', '-p', default='tangerine',
                        choices=['tangerine', 'joseph', 'jennifer'],
                        help='Default persona for voice configuration')
    parser.add_argument('--voice-engine', default='generative',
                        choices=['standard', 'neural', 'generative'],
                        help='Voice engine (generative for Nova Sonic)')
    parser.add_argument('--dry-run', action='store_true',
                        help='Print what would be done without making changes')

    args = parser.parse_args()

    full_bot_name = f"{args.bot_name}-{args.environment}"
    persona_config = PERSONA_VOICES.get(args.persona, PERSONA_VOICES['tangerine'])

    print(f"Configuring Nova Sonic for bot: {full_bot_name}")
    print(f"Region: {args.region}")
    print(f"Persona: {args.persona}")
    print(f"Voice: {persona_config['nova_sonic']} (Nova Sonic) / {persona_config['polly']} (Polly)")
    print(f"Engine: {args.voice_engine}")

    if args.dry_run:
        print("\n*** DRY RUN - No changes will be made ***")
        return

    # Initialize client
    lex_client = get_lex_client(args.region)

    # Get bot ID
    bot_id = get_bot_id(lex_client, full_bot_name)
    if not bot_id:
        print(f"ERROR: Bot {full_bot_name} not found")
        return
    print(f"Found bot ID: {bot_id}")

    # For Nova Sonic, we use generative engine with appropriate voice
    # The voice ID for Nova Sonic enabled bots uses Polly voice names
    # but the engine determines whether Nova Sonic is used
    voice_id = persona_config['polly']

    # Update bot locale with voice settings
    if update_bot_locale_voice(lex_client, bot_id, 'en_US', voice_id, args.voice_engine):
        print("Voice settings updated successfully")

        # Build the bot
        if build_bot_locale(lex_client, bot_id, 'en_US'):
            if wait_for_bot_locale(lex_client, bot_id, 'en_US'):
                print("\n=== Nova Sonic Configuration Complete ===")
                print(f"Bot: {full_bot_name}")
                print(f"Voice: {voice_id}")
                print(f"Engine: {args.voice_engine}")
                print("\nNote: For full Nova Sonic speech-to-speech:")
                print("1. Enable in Amazon Connect admin console")
                print("2. Set contact flow voice to 'Generative'")
            else:
                print("WARNING: Bot build did not complete successfully")
        else:
            print("ERROR: Failed to build bot")
    else:
        print("ERROR: Failed to update voice settings")


if __name__ == '__main__':
    main()
