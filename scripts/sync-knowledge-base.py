#!/usr/bin/env python3
"""
WS-A-06: Knowledge Base ingestion automation for the Headset Support Agent.

This script REPLACES the former bare `aws s3 sync` step in deploy.yml. It:
  1. Syncs the local knowledge-base/ tree to the KB docs S3 bucket
     (upload changed/new objects, delete objects no longer present locally),
     excluding VCS/OS cruft.
  2. Starts a Bedrock Knowledge Base ingestion job against the S3 data source.
  3. Polls the ingestion job until it reaches COMPLETE.

It is idempotent: re-running with no doc changes simply re-ingests (Bedrock
skips unchanged chunks) and the SSM-sourced identifiers are read fresh each run.

Fail-closed: ANY failure (missing config, sync error, ingestion FAILED/STOPPED,
or poll timeout) exits non-zero so the GitHub Actions step fails. There is no
continue-on-error / `|| true` fallback and no stubbed success.

Resolution of KB identifiers:
  - Bucket:        SSM /headset-agent/<env>/  (KnowledgeBaseBucket output name pattern)
                   actually resolved from --bucket or the kb-id's KB describe call.
  - kb-id:         SSM /headset-agent/<env>/kb-id            (written by CloudFormation)
  - data-source-id SSM /headset-agent/<env>/kb-data-source-id (written by CloudFormation)
"""

import argparse
import hashlib
import os
import sys
import time

import boto3
from botocore.exceptions import ClientError

# Local doc tree relative to the repo root.
KB_LOCAL_DIR = "knowledge-base"

# Patterns excluded from the S3 sync (mirrors the old deploy.yml excludes).
EXCLUDE_SUFFIXES = (".DS_Store",)
EXCLUDE_DIR_PARTS = (".git",)

# Ingestion job is normally quick for a small corpus; cap generously.
POLL_TIMEOUT_SECONDS = 1800  # 30 minutes
POLL_INTERVAL_SECONDS = 10

TERMINAL_OK = {"COMPLETE"}
TERMINAL_BAD = {"FAILED", "STOPPED"}


def ssm_get(ssm, name):
    """Read an SSM parameter value, returning None when absent."""
    try:
        resp = ssm.get_parameter(Name=name)
        return resp["Parameter"]["Value"]
    except ClientError as exc:
        if exc.response["Error"]["Code"] == "ParameterNotFound":
            return None
        raise


def resolve_config(args, ssm, bedrock_agent):
    """Resolve bucket, kb-id and data-source-id from args + SSM + KB describe."""
    env = args.environment
    prefix = f"/headset-agent/{env}"

    kb_id = args.kb_id or ssm_get(ssm, f"{prefix}/kb-id")
    if not kb_id or kb_id == "PLACEHOLDER":
        sys.exit(
            f"ERROR: knowledge base id not available at {prefix}/kb-id "
            f"(value: {kb_id!r}). Deploy the SAM stack (WS-A-05) first."
        )

    ds_id = args.data_source_id or ssm_get(ssm, f"{prefix}/kb-data-source-id")
    if not ds_id or ds_id == "PLACEHOLDER":
        sys.exit(
            f"ERROR: data source id not available at {prefix}/kb-data-source-id "
            f"(value: {ds_id!r}). Deploy the SAM stack (WS-A-05) first."
        )

    bucket = args.bucket
    if not bucket:
        # Derive the docs bucket from the data source's S3 configuration so we
        # never sync to the wrong bucket. The BucketArn is arn:aws:s3:::<name>.
        ds = bedrock_agent.get_data_source(
            knowledgeBaseId=kb_id, dataSourceId=ds_id
        )
        s3_cfg = ds["dataSource"]["dataSourceConfiguration"]["s3Configuration"]
        bucket_arn = s3_cfg["bucketArn"]
        bucket = bucket_arn.split(":::", 1)[1]

    return bucket, kb_id, ds_id


def iter_local_docs(root):
    """Yield (absolute_path, s3_key) for every doc to upload."""
    for dirpath, dirnames, filenames in os.walk(root):
        # Prune excluded directories in place.
        dirnames[:] = [d for d in dirnames if d not in EXCLUDE_DIR_PARTS]
        for name in filenames:
            if name.endswith(EXCLUDE_SUFFIXES):
                continue
            abspath = os.path.join(dirpath, name)
            rel = os.path.relpath(abspath, root).replace(os.sep, "/")
            yield abspath, rel


def s3_etag_md5(path):
    """Compute the MD5 hex digest used by S3 ETag for non-multipart objects."""
    h = hashlib.md5()
    with open(path, "rb") as fh:
        for chunk in iter(lambda: fh.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def sync_docs(s3, bucket, root):
    """Upload new/changed docs and delete S3 objects no longer present locally.

    Returns the number of objects uploaded. Raises on any S3 error so the
    caller can fail the step.
    """
    if not os.path.isdir(root):
        sys.exit(f"ERROR: local knowledge-base directory not found: {root}")

    # Current remote objects -> ETag.
    remote = {}
    paginator = s3.get_paginator("list_objects_v2")
    for page in paginator.paginate(Bucket=bucket):
        for obj in page.get("Contents", []):
            remote[obj["Key"]] = obj["ETag"].strip('"')

    local_keys = set()
    uploaded = 0
    for abspath, key in iter_local_docs(root):
        local_keys.add(key)
        local_md5 = s3_etag_md5(abspath)
        if remote.get(key) == local_md5:
            continue  # unchanged
        print(f"  upload: {key}")
        s3.upload_file(abspath, bucket, key)
        uploaded += 1

    # Delete remote objects that no longer exist locally (the old --delete).
    stale = [k for k in remote if k not in local_keys]
    if stale:
        print(f"  deleting {len(stale)} stale object(s) from s3://{bucket}/")
        # delete_objects handles up to 1000 keys per call.
        for i in range(0, len(stale), 1000):
            batch = [{"Key": k} for k in stale[i : i + 1000]]
            s3.delete_objects(Bucket=bucket, Delete={"Objects": batch})

    print(
        f"Sync complete: {uploaded} uploaded, {len(stale)} deleted, "
        f"{len(local_keys)} total local docs."
    )
    return uploaded


def start_and_wait(bedrock_agent, kb_id, ds_id):
    """Start an ingestion job and poll until COMPLETE; fail otherwise."""
    print(f"Starting ingestion job (kb={kb_id}, dataSource={ds_id})...")
    resp = bedrock_agent.start_ingestion_job(
        knowledgeBaseId=kb_id,
        dataSourceId=ds_id,
        description="Automated ingestion from sync-knowledge-base.py (WS-A-06)",
    )
    job_id = resp["ingestionJob"]["ingestionJobId"]
    print(f"  ingestionJobId: {job_id}")

    deadline = time.time() + POLL_TIMEOUT_SECONDS
    while time.time() < deadline:
        job = bedrock_agent.get_ingestion_job(
            knowledgeBaseId=kb_id,
            dataSourceId=ds_id,
            ingestionJobId=job_id,
        )["ingestionJob"]
        status = job["status"]
        stats = job.get("statistics", {})
        print(f"  status: {status} stats: {stats}")

        if status in TERMINAL_OK:
            print("Ingestion COMPLETE.")
            return
        if status in TERMINAL_BAD:
            reasons = job.get("failureReasons", [])
            sys.exit(
                f"ERROR: ingestion job {job_id} ended in {status}. "
                f"Reasons: {reasons}"
            )
        time.sleep(POLL_INTERVAL_SECONDS)

    sys.exit(
        f"ERROR: ingestion job {job_id} did not reach COMPLETE within "
        f"{POLL_TIMEOUT_SECONDS}s (last status polled above)."
    )


def main():
    parser = argparse.ArgumentParser(
        description="Sync KB docs to S3 and run a Bedrock ingestion job (WS-A-06)."
    )
    parser.add_argument("--environment", "-e", default="prod", choices=["prod"])
    parser.add_argument("--region", "-r", default="us-east-1")
    parser.add_argument(
        "--local-dir", default=KB_LOCAL_DIR, help="Local knowledge-base directory"
    )
    parser.add_argument("--bucket", default=None, help="Override KB docs bucket name")
    parser.add_argument("--kb-id", default=None, help="Override knowledge base id")
    parser.add_argument(
        "--data-source-id", default=None, help="Override data source id"
    )
    parser.add_argument(
        "--skip-sync",
        action="store_true",
        help="Skip the S3 sync and only run the ingestion job",
    )
    args = parser.parse_args()

    print(f"WS-A-06 knowledge base sync — env={args.environment} region={args.region}")

    ssm = boto3.client("ssm", region_name=args.region)
    s3 = boto3.client("s3", region_name=args.region)
    bedrock_agent = boto3.client("bedrock-agent", region_name=args.region)

    bucket, kb_id, ds_id = resolve_config(args, ssm, bedrock_agent)
    print(f"Resolved: bucket={bucket} kb_id={kb_id} data_source_id={ds_id}")

    if args.skip_sync:
        print("Skipping S3 sync (--skip-sync).")
    else:
        sync_docs(s3, bucket, args.local_dir)

    start_and_wait(bedrock_agent, kb_id, ds_id)
    print("=== Knowledge base sync + ingestion succeeded ===")


if __name__ == "__main__":
    main()
