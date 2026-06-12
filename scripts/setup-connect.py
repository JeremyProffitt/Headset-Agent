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


def wait_for_instance_ready(client, instance_id, timeout=300):
    """Wait for Connect instance to be fully operational (able to list contact flows)"""
    print(f"  Waiting for Connect instance to be fully operational...")
    start_time = time.time()
    last_error = None

    while time.time() - start_time < timeout:
        try:
            # Try to list contact flows - this is a good indicator that the instance is ready
            client.list_contact_flows(
                InstanceId=instance_id,
                ContactFlowTypes=['CONTACT_FLOW'],
                MaxResults=1
            )
            print(f"  Connect instance is ready")
            return True
        except ClientError as e:
            error_message = str(e)
            if 'inactive' in error_message.lower() or 'ResourceNotFoundException' in error_message:
                last_error = e
                print(f"  Instance not ready yet, waiting...")
                time.sleep(15)
            else:
                print(f"  Error checking instance: {e}")
                return False

    print(f"  Timeout waiting for instance to be ready. Last error: {last_error}")
    return False


def wait_for_phone_number_ready(client, phone_number_id, timeout=180):
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
                if status_message:
                    print(f"  Message: {status_message}")
                if 'limit' in status_message.lower() or 'quota' in status_message.lower():
                    print("")
                    print("  ⚠️  PHONE NUMBER QUOTA ISSUE DETECTED")
                    print("  This is a known AWS issue. Resolution requires AWS Support.")
                    print("")
                return False
            elif status_value == 'IN_PROGRESS':
                print(f"  Status: IN_PROGRESS, waiting...")
                time.sleep(10)
            else:
                print(f"  Status: {status_value}, waiting...")
                time.sleep(5)
        except ClientError as e:
            print(f"  Error checking status: {e}")
            time.sleep(5)

    print(f"  Timeout waiting for phone number to be ready")
    return False


def release_phone_number(client, phone_number_id):
    """Release a phone number from Connect"""
    try:
        client.release_phone_number(PhoneNumberId=phone_number_id)
        print(f"  Released phone number: {phone_number_id}")
        # Wait for release to complete
        time.sleep(5)
        return True
    except ClientError as e:
        if 'ResourceNotFoundException' in str(e):
            print(f"  Phone number already released or not found")
            return True
        print(f"  Error releasing phone number: {e}")
        return False


def get_phone_number_status(client, phone_number_id):
    """Get the status of a phone number"""
    try:
        response = client.describe_phone_number(PhoneNumberId=phone_number_id)
        status = response.get('ClaimedPhoneNumberSummary', {}).get('PhoneNumberStatus', {})
        return status.get('Status', 'UNKNOWN')
    except ClientError as e:
        if 'ResourceNotFoundException' in str(e):
            return 'NOT_FOUND'
        print(f"  Error getting phone number status: {e}")
        return 'ERROR'


def find_and_cleanup_failed_phone_numbers(client, instance_id):
    """Find and release any phone numbers in FAILED state"""
    try:
        if instance_id.startswith('arn:'):
            target_arn = instance_id
        else:
            target_arn = f"arn:aws:connect:us-east-1:{get_account_id()}:instance/{instance_id}"

        response = client.list_phone_numbers_v2(TargetArn=target_arn)

        released_count = 0
        for phone in response.get('ListPhoneNumbersSummaryList', []):
            phone_id = phone.get('PhoneNumberId')
            if phone_id:
                status = get_phone_number_status(client, phone_id)
                if status in ['FAILED', 'CANCELLED']:
                    print(f"  Found failed phone number: {phone.get('PhoneNumber')} (status: {status})")
                    if release_phone_number(client, phone_id):
                        released_count += 1

        if released_count > 0:
            print(f"  Released {released_count} failed phone number(s)")
            # Wait for releases to fully propagate
            print("  Waiting for releases to propagate...")
            time.sleep(30)

        return released_count
    except ClientError as e:
        print(f"  Error cleaning up failed phone numbers: {e}")
        return 0


def claim_phone_number(client, instance_id, country_code='US', phone_type='DID', description='', max_retries=3):
    """Claim a phone number for the Connect instance with retry logic"""

    for attempt in range(max_retries):
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

            # Claim the phone number
            claim_response = client.claim_phone_number(
                TargetArn=target_arn,
                PhoneNumber=phone_number,
                PhoneNumberDescription=description,
                Tags={
                    'Environment': 'prod',
                    'Project': 'HeadsetSupportAgent'
                }
            )

            claimed_phone_number_id = claim_response.get('PhoneNumberId')
            print(f"Claimed phone number: {phone_number} (ID: {claimed_phone_number_id})")

            # Wait for phone number to be fully provisioned
            if claimed_phone_number_id:
                if wait_for_phone_number_ready(client, claimed_phone_number_id):
                    # SUCCESS - phone is ready
                    return {
                        'PhoneNumber': phone_number,
                        'PhoneNumberId': claimed_phone_number_id,
                        'PhoneNumberArn': claim_response.get('PhoneNumberArn'),
                        'Status': 'CLAIMED'
                    }
                else:
                    # Provisioning failed - release the phone number and retry
                    print(f"  Provisioning failed, releasing phone number...")
                    release_phone_number(client, claimed_phone_number_id)

                    if attempt < max_retries - 1:
                        wait_time = 30 * (attempt + 1)  # Exponential backoff
                        print(f"  Waiting {wait_time}s before retry (attempt {attempt + 2}/{max_retries})...")
                        time.sleep(wait_time)
                    continue

            return None

        except ClientError as e:
            if 'ResourceNotFoundException' in str(e):
                print(f"No phone numbers available or instance not found")
            elif 'LimitExceededException' in str(e) or 'ServiceQuotaExceededException' in str(e):
                print(f"  Phone number quota exceeded")
                if attempt < max_retries - 1:
                    wait_time = 60 * (attempt + 1)
                    print(f"  Waiting {wait_time}s before retry...")
                    time.sleep(wait_time)
                    continue
            else:
                print(f"Error claiming phone number: {e}")
            return None

    print(f"  Failed to claim phone number after {max_retries} attempts")
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


# WS-C-05: contact-flow creation/update has been REMOVED from this script.
# The inline AWS::Connect::ContactFlow resources in infrastructure/template.yaml
# are the single source of truth for both the Lex and Nova Sonic flows. This
# script only claims phone numbers and associates them to the CFN-created flows
# (looked up by name via get_contact_flow_id_by_name). Do not re-introduce flow
# definitions here.


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
    """Verify if a phone number is claimed and active in Connect (not failed)"""
    try:
        # Get target ARN
        if instance_id.startswith('arn:'):
            target_arn = instance_id
        else:
            target_arn = f"arn:aws:connect:us-east-1:{get_account_id()}:instance/{instance_id}"

        response = client.list_phone_numbers_v2(TargetArn=target_arn)

        for phone in response.get('ListPhoneNumbersSummaryList', []):
            if phone.get('PhoneNumber') == phone_number:
                # Also check the phone number status
                phone_id = phone.get('PhoneNumberId')
                if phone_id:
                    status = get_phone_number_status(client, phone_id)
                    if status == 'CLAIMED':
                        return True
                    else:
                        print(f"  Phone number exists but status is {status}")
                        return False
                return True
        return False
    except ClientError as e:
        print(f"Error verifying phone number: {e}")
        return False


def list_all_instance_phone_numbers(client, instance_id):
    """
    Return a list of dicts for every phone number claimed to this instance.
    Each dict has: PhoneNumberId, PhoneNumber, PhoneNumberArn, Status,
    and ContactFlowId (may be None if not associated with any flow).
    Only returns numbers whose status is CLAIMED.
    """
    try:
        if instance_id.startswith('arn:'):
            target_arn = instance_id
        else:
            target_arn = f"arn:aws:connect:us-east-1:{get_account_id()}:instance/{instance_id}"

        results = []
        paginator_token = None
        while True:
            kwargs = {'TargetArn': target_arn, 'MaxResults': 100}
            if paginator_token:
                kwargs['NextToken'] = paginator_token
            response = client.list_phone_numbers_v2(**kwargs)
            for item in response.get('ListPhoneNumbersSummaryList', []):
                phone_id = item.get('PhoneNumberId')
                if not phone_id:
                    continue
                # Only care about CLAIMED numbers
                status = get_phone_number_status(client, phone_id)
                if status != 'CLAIMED':
                    continue
                # Determine associated contact flow (if any)
                contact_flow_id = get_phone_number_contact_flow(client, phone_id)
                results.append({
                    'PhoneNumberId': phone_id,
                    'PhoneNumber': item.get('PhoneNumber'),
                    'PhoneNumberArn': item.get('PhoneNumberArn'),
                    'Status': 'CLAIMED',
                    'ContactFlowId': contact_flow_id,
                })
            paginator_token = response.get('NextToken')
            if not paginator_token:
                break
        return results
    except ClientError as e:
        print(f"Error listing instance phone numbers: {e}")
        return []


def get_phone_number_contact_flow(client, phone_number_id):
    """
    Return the ContactFlowId that a phone number is associated with,
    or None if it is not associated with any flow.
    Returns the sentinel string 'UNKNOWN' if the association status
    cannot be determined (caller should treat this as in-use / do not release).
    """
    try:
        response = client.describe_phone_number(PhoneNumberId=phone_number_id)
        summary = response.get('ClaimedPhoneNumberSummary', {})
        # The ContactFlowId field is present when the number is associated
        flow_id = summary.get('ContactFlowId')
        return flow_id  # None means not associated
    except ClientError as e:
        print(f"  Warning: could not describe phone number {phone_number_id}: {e}")
        return 'UNKNOWN'


def resolve_phone_for_path(
    client, ssm_client, environment,
    path_name, flow_id, ssm_param,
    instance_id, all_claimed_numbers, already_assigned_ids
):
    """
    Idempotent resolution of a phone number for one path (Lex or Nova Sonic).

    Strategy (in order):
      1. Reuse SSM: if the SSM param holds a number that is still CLAIMED in
         this instance, keep it and ensure it is associated with flow_id. Return
         the PhoneNumberId without claiming anything new.
      2. Reuse existing claimed: pick any CLAIMED number that is not yet assigned
         to one of our flows (i.e., ContactFlowId is None / unset), associate it
         with flow_id, write the SSM param. Return the PhoneNumberId.
      3. Claim new: only if no reusable number exists.

    Returns the PhoneNumberId that was assigned (str), or None on failure.
    Modifies already_assigned_ids in-place by adding the assigned ID.
    """
    # --- Step 1: Reuse SSM ---
    existing_ssm_value = get_ssm_parameter(ssm_client, ssm_param)
    if existing_ssm_value and existing_ssm_value not in ('PLACEHOLDER', 'PENDING'):
        # Find the corresponding claimed record
        for rec in all_claimed_numbers:
            if rec['PhoneNumber'] == existing_ssm_value:
                print(f"  [{path_name}] SSM param matches claimed number {existing_ssm_value} - reusing")
                # Ensure it is associated with the correct flow
                if flow_id and rec.get('ContactFlowId') != flow_id:
                    print(f"  [{path_name}] Re-associating phone to flow {flow_id}")
                    associate_phone_with_flow(client, instance_id, rec['PhoneNumberId'], flow_id)
                already_assigned_ids.add(rec['PhoneNumberId'])
                return rec['PhoneNumberId']
        # SSM value not found among claimed numbers - fall through to reuse/claim
        print(f"  [{path_name}] SSM value {existing_ssm_value} not found among claimed numbers - will reuse or claim")

    # --- Step 2: Reuse an existing claimed number not yet assigned to any flow ---
    for rec in all_claimed_numbers:
        if rec['PhoneNumberId'] in already_assigned_ids:
            continue  # already spoken for
        flow = rec.get('ContactFlowId')
        if flow == 'UNKNOWN':
            continue  # can't determine status - skip safely
        if flow is None:
            # Unassociated - take it
            print(f"  [{path_name}] Reusing unassigned claimed number {rec['PhoneNumber']} (ID: {rec['PhoneNumberId']})")
            if flow_id:
                associate_phone_with_flow(client, instance_id, rec['PhoneNumberId'], flow_id)
            save_to_ssm(ssm_client, ssm_param, rec['PhoneNumber'],
                        f"Phone number for {path_name} path")
            already_assigned_ids.add(rec['PhoneNumberId'])
            return rec['PhoneNumberId']

    # --- Step 3: Claim new ---
    print(f"  [{path_name}] No reusable number found - claiming a new one")
    if not flow_id:
        print(f"  [{path_name}] Contact flow not found - skipping claim")
        return None

    new_phone = claim_phone_number(
        client, instance_id,
        phone_type='TOLL_FREE',
        description=f"Headset Support - {path_name} Path ({environment})"
    )
    if new_phone and new_phone.get('Status') == 'CLAIMED':
        save_to_ssm(ssm_client, ssm_param, new_phone['PhoneNumber'],
                    f"Phone number for {path_name} path")
        phone_id = new_phone.get('PhoneNumberId')
        if phone_id:
            associate_phone_with_flow(client, instance_id, phone_id, flow_id)
            # Add to the in-memory list so orphan cleanup sees it
            all_claimed_numbers.append({
                'PhoneNumberId': phone_id,
                'PhoneNumber': new_phone['PhoneNumber'],
                'PhoneNumberArn': new_phone.get('PhoneNumberArn'),
                'Status': 'CLAIMED',
                'ContactFlowId': flow_id,
            })
            already_assigned_ids.add(phone_id)
        print(f"  [{path_name}] Phone number ready: {new_phone['PhoneNumber']}")
        return phone_id
    else:
        print(f"  [{path_name}] Failed to claim phone number")
        return None


def release_orphaned_phone_numbers(client, all_claimed_numbers, assigned_ids):
    """
    Release phone numbers that are:
      (a) claimed to this instance,
      (b) NOT one of the assigned numbers (Lex / Nova), AND
      (c) NOT associated with any contact flow (ContactFlowId is None).

    Numbers whose ContactFlowId is 'UNKNOWN' are skipped (fail-safe: cannot
    determine association, so do not release).

    Per-number try/except: one failure does not abort the rest.
    """
    released = 0
    for rec in all_claimed_numbers:
        phone_id = rec['PhoneNumberId']
        phone_num = rec.get('PhoneNumber', phone_id)

        if phone_id in assigned_ids:
            continue  # in use by one of our paths

        flow = rec.get('ContactFlowId')
        if flow == 'UNKNOWN':
            print(f"  SKIP release of {phone_num}: association status unknown (fail-safe)")
            continue
        if flow is not None:
            print(f"  SKIP release of {phone_num}: associated with flow {flow} (in use)")
            continue

        # Safe to release: claimed, not assigned to us, not associated with any flow
        print(f"  Releasing orphaned phone number {phone_num} (ID: {phone_id}) - claimed but not associated with any flow")
        try:
            if release_phone_number(client, phone_id):
                released += 1
        except Exception as e:
            print(f"  Error releasing {phone_num}: {e} - skipping")

    if released:
        print(f"  Released {released} orphaned phone number(s)")
    else:
        print(f"  No orphaned phone numbers to release")


def main():
    parser = argparse.ArgumentParser(description='Setup Amazon Connect for Headset Support Agent')
    parser.add_argument('--environment', '-e', default='prod', choices=['prod'])
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

    # Wait for Connect instance to be fully operational
    print("\nWaiting for Connect instance to be fully operational...")
    if not wait_for_instance_ready(connect_client, instance_id, timeout=300):
        print("WARN: Connect instance is not fully operational yet.")
        print("      Phone numbers cannot be claimed until the instance is ready.")
        print("      Re-run the pipeline in a few minutes.")
        return 0

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

    # Step 3: Phone numbers — reuse before claiming, then release orphans
    print("\n--- Step 3: Phone Numbers ---")
    if args.skip_phone_numbers:
        print("Skipping phone number claiming (--skip-phone-numbers)")
    else:
        # First, clean up any failed phone numbers from previous attempts
        print("Checking for failed phone numbers to clean up...")
        find_and_cleanup_failed_phone_numbers(connect_client, instance_id)

        # Build a single snapshot of all CLAIMED numbers on this instance.
        # This is used for both the reuse logic and the orphan-release pass.
        print("Loading all claimed phone numbers for this instance...")
        all_claimed = list_all_instance_phone_numbers(connect_client, instance_id)
        print(f"  Found {len(all_claimed)} CLAIMED phone number(s) on this instance")

        # Track which phone number IDs we intentionally assign so the orphan
        # pass knows what to leave alone.
        assigned_ids = set()

        # --- Lex path (always needed when lex_flow_id is present) ---
        print("\nResolving phone number for Lex path...")
        lex_phone_id = resolve_phone_for_path(
            client=connect_client,
            ssm_client=ssm_client,
            environment=args.environment,
            path_name="Lex",
            flow_id=lex_flow_id,
            ssm_param=f"/headset-agent/{args.environment}/connect/phone-number-lex",
            instance_id=instance_id,
            all_claimed_numbers=all_claimed,
            already_assigned_ids=assigned_ids,
        )
        if lex_phone_id:
            print(f"  Lex path phone number assigned (ID: {lex_phone_id})")
        else:
            print("  Lex path phone number could not be assigned")

        # --- Nova Sonic path (needed only when its contact flow exists) ---
        # The flow presence is the authoritative signal: if CloudFormation
        # deployed the Nova Sonic flow, nova_flow_id will be non-None.
        if nova_flow_id:
            print("\nResolving phone number for Nova Sonic path...")
            nova_phone_id = resolve_phone_for_path(
                client=connect_client,
                ssm_client=ssm_client,
                environment=args.environment,
                path_name="Nova Sonic",
                flow_id=nova_flow_id,
                ssm_param=f"/headset-agent/{args.environment}/connect/phone-number-nova-sonic",
                instance_id=instance_id,
                all_claimed_numbers=all_claimed,
                already_assigned_ids=assigned_ids,
            )
            if nova_phone_id:
                print(f"  Nova Sonic path phone number assigned (ID: {nova_phone_id})")
            else:
                print("  Nova Sonic path phone number could not be assigned")
        else:
            print("\nNova Sonic contact flow not found - skipping Nova Sonic phone number")

        # --- Release orphaned numbers ---
        # Release any number that is claimed to this instance, is NOT one of
        # our two assigned numbers, and is NOT associated with any contact flow.
        # Numbers with unknown association status are skipped (fail-safe).
        print("\nChecking for orphaned phone numbers to release...")
        release_orphaned_phone_numbers(connect_client, all_claimed, assigned_ids)

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
