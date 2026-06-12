"""Post-deploy integration smoke tests (WS-G-05 / WS-G-10).

These run in the deploy pipeline AFTER the SAM stack is deployed. They exercise
real, deployed AWS resources and FAIL the deploy on a broken deployment.

Reliability note: the Lex health-check path (BuildTestResponse) does NOT depend
on the Bedrock agent being configured, so this smoke is deterministic regardless
of agent/KB readiness. The richer "/chat returns a non-placeholder grounded
answer with the API token" assertion is added in M3 once WS-D-08 provisions the
shared API token (tracked in PLAN.md WS-G-05 deps: D-08).
"""

import json
import os

import boto3
import pytest

ENV = os.environ.get("ENVIRONMENT", "prod")
REGION = os.environ.get("AWS_REGION", "us-east-1")

LEX_FUNCTION = f"headset-lex-orchestrator-{ENV}"


@pytest.fixture(scope="module")
def lambda_client():
    return boto3.client("lambda", region_name=REGION)


def test_lex_lambda_health_check(lambda_client):
    """The Lex orchestrator must return the health-check response for the
    test invocation payload. A non-200 status, a function error, or a missing
    'Health check successful' message fails the deploy."""
    payload = {"sessionState": {"sessionAttributes": {"test": "true"}}}

    resp = lambda_client.invoke(
        FunctionName=LEX_FUNCTION,
        Payload=json.dumps(payload).encode("utf-8"),
    )

    assert resp["StatusCode"] == 200, f"Unexpected Lambda status: {resp['StatusCode']}"
    assert "FunctionError" not in resp, (
        f"Lambda returned a function error: "
        f"{resp['Payload'].read().decode('utf-8')}"
    )

    body = json.loads(resp["Payload"].read().decode("utf-8"))
    messages = body.get("messages", [])
    assert messages, f"No messages in Lex response: {body}"
    contents = " ".join(m.get("content", "") for m in messages)
    assert "Health check successful" in contents, (
        f"Health-check message not found in Lex response: {body}"
    )
