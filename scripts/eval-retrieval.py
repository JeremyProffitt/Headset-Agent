#!/usr/bin/env python3
"""
A-10: Retrieval evaluation gate for the Headset Support Agent knowledge base.

Runs a golden-question suite against the live Bedrock Knowledge Base and
asserts that the hit rate meets a minimum threshold (default 90%).

A question PASSES when any of the top-K retrieved results matches the
expected target:
  - expect_tree_id: metadata['tree_id'] == expected value
  - expect_source:  location.s3Location.uri contains the expected substring

Exit codes:
  0 — hit rate >= threshold (gate passes)
  1 — hit rate < threshold OR any unrecoverable error (gate fails)

Usage in CI:
  python scripts/eval-retrieval.py --region us-east-1
"""

import argparse
import json
import os
import sys

import boto3
from botocore.exceptions import ClientError

# Default golden-question file path relative to the repo root.
DEFAULT_GOLDEN = "tests/retrieval/golden.json"
DEFAULT_THRESHOLD = 0.90
DEFAULT_TOP_K = 3
DEFAULT_REGION = "us-east-1"
SSM_KB_ID_PARAM = "/headset-agent/prod/kb-id"


def resolve_kb_id(args, region: str) -> str:
    """Return the KB id from --kb-id arg, SSM parameter, or KB_ID env var."""
    if args.kb_id:
        return args.kb_id

    # Try SSM first.
    try:
        ssm = boto3.client("ssm", region_name=region)
        resp = ssm.get_parameter(Name=SSM_KB_ID_PARAM)
        kb_id = resp["Parameter"]["Value"]
        print(f"Resolved KB id from SSM ({SSM_KB_ID_PARAM}): {kb_id}")
        return kb_id
    except ClientError as exc:
        code = exc.response["Error"]["Code"]
        if code != "ParameterNotFound":
            sys.exit(
                f"ERROR: SSM error resolving {SSM_KB_ID_PARAM}: {exc}"
            )

    # Fall back to env var.
    kb_id = os.environ.get("KB_ID", "")
    if kb_id:
        print(f"Resolved KB id from KB_ID env var: {kb_id}")
        return kb_id

    sys.exit(
        f"ERROR: KB id not found. Provide --kb-id, set the SSM parameter "
        f"{SSM_KB_ID_PARAM}, or export KB_ID=<id>."
    )


def load_golden(path: str) -> list:
    """Load and validate the golden-question file."""
    try:
        with open(path, encoding="utf-8") as fh:
            data = json.load(fh)
    except FileNotFoundError:
        sys.exit(f"ERROR: golden file not found: {path}")
    except json.JSONDecodeError as exc:
        sys.exit(f"ERROR: invalid JSON in {path}: {exc}")

    if not isinstance(data, list) or not data:
        sys.exit(f"ERROR: {path} must be a non-empty JSON array.")

    for i, item in enumerate(data):
        has_tree = "expect_tree_id" in item
        has_src = "expect_source" in item
        if not item.get("q"):
            sys.exit(f"ERROR: entry {i} is missing 'q'.")
        if has_tree == has_src:  # both present or neither present
            sys.exit(
                f"ERROR: entry {i} must have exactly one of "
                f"'expect_tree_id' or 'expect_source', not both/neither."
            )

    return data


def retrieve(client, kb_id: str, query: str, top_k: int) -> list:
    """Call Bedrock retrieve; return list of result dicts (metadata + uri)."""
    try:
        resp = client.retrieve(
            knowledgeBaseId=kb_id,
            retrievalQuery={"text": query},
            retrievalConfiguration={
                "vectorSearchConfiguration": {"numberOfResults": top_k}
            },
        )
    except ClientError as exc:
        sys.exit(
            f"ERROR: bedrock-agent-runtime.retrieve failed: {exc}\n"
            f"  Query: {query!r}\n"
            f"  Make sure AWS credentials are configured and the KB id is correct."
        )
    return resp.get("retrievalResults", [])


def evaluate(client, kb_id: str, golden: list, top_k: int, threshold: float):
    """Run the full eval suite; return (passes, total, failures)."""
    passes = 0
    failures = []

    for item in golden:
        q = item["q"]
        results = retrieve(client, kb_id, q, top_k)

        tree_ids = [r.get("metadata", {}).get("tree_id", "") for r in results]
        uris = [
            r.get("location", {}).get("s3Location", {}).get("uri", "")
            for r in results
        ]

        passed = False
        if "expect_tree_id" in item:
            expected = item["expect_tree_id"]
            if expected in tree_ids:
                passed = True
        else:
            expected = item["expect_source"]
            if any(expected in uri for uri in uris):
                passed = True

        if passed:
            passes += 1
        else:
            failures.append((q, tree_ids, uris, item))

    total = len(golden)
    hit_rate = passes / total if total else 0.0

    # Print failures.
    for (q, tree_ids, uris, item) in failures:
        if "expect_tree_id" in item:
            print(
                f"FAIL: {q!r}\n"
                f"  expected tree_id={item['expect_tree_id']!r}, "
                f"got tree_ids={tree_ids}"
            )
        else:
            print(
                f"FAIL: {q!r}\n"
                f"  expected source containing {item['expect_source']!r}, "
                f"got uris={uris}"
            )

    print(
        f"\nRetrieval eval: {passes}/{total} passed "
        f"({hit_rate * 100:.1f}%) threshold {threshold * 100:.1f}%"
    )

    if hit_rate < threshold:
        print(
            f"GATE FAILED: hit rate {hit_rate * 100:.1f}% is below "
            f"the {threshold * 100:.1f}% threshold."
        )
        return False

    print(f"GATE PASSED: hit rate {hit_rate * 100:.1f}% meets the threshold.")
    return True


def main():
    parser = argparse.ArgumentParser(
        description=(
            "A-10 retrieval eval gate: run golden questions against the "
            "Bedrock Knowledge Base and fail if hit rate < threshold."
        )
    )
    parser.add_argument(
        "--kb-id",
        default=None,
        help=(
            f"Knowledge base ID (default: read SSM {SSM_KB_ID_PARAM}, "
            f"fall back to env KB_ID)"
        ),
    )
    parser.add_argument(
        "--region",
        default=DEFAULT_REGION,
        help=f"AWS region (default: {DEFAULT_REGION})",
    )
    parser.add_argument(
        "--threshold",
        type=float,
        default=DEFAULT_THRESHOLD,
        help=f"Minimum required hit rate 0.0-1.0 (default: {DEFAULT_THRESHOLD})",
    )
    parser.add_argument(
        "--top-k",
        type=int,
        default=DEFAULT_TOP_K,
        help=f"Number of results to retrieve per question (default: {DEFAULT_TOP_K})",
    )
    parser.add_argument(
        "--golden",
        default=DEFAULT_GOLDEN,
        help=f"Path to golden question JSON file (default: {DEFAULT_GOLDEN})",
    )
    args = parser.parse_args()

    if not (0.0 < args.threshold <= 1.0):
        sys.exit(f"ERROR: --threshold must be in range (0, 1], got {args.threshold}")

    print(
        f"A-10 retrieval eval — region={args.region} "
        f"top_k={args.top_k} threshold={args.threshold * 100:.1f}%"
    )

    golden = load_golden(args.golden)
    print(f"Loaded {len(golden)} golden questions from {args.golden}")

    kb_id = resolve_kb_id(args, args.region)

    client = boto3.client("bedrock-agent-runtime", region_name=args.region)

    passed = evaluate(client, kb_id, golden, args.top_k, args.threshold)
    sys.exit(0 if passed else 1)


if __name__ == "__main__":
    main()
