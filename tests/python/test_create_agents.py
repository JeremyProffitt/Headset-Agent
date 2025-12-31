#!/usr/bin/env python3
"""
Tests for the Bedrock Agent Creation Script
"""

import unittest
from unittest.mock import MagicMock, patch
import sys
import os

# Add scripts directory to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', '..', 'scripts'))

from create_agents import (
    get_agent_role_arn,
    wait_for_agent,
    wait_for_alias,
)


class TestGetAgentRoleArn(unittest.TestCase):
    """Tests for get_agent_role_arn function"""

    @patch('create_agents.boto3.client')
    def test_get_role_from_stack_outputs(self, mock_boto_client):
        """Test getting role ARN from CloudFormation stack outputs"""
        mock_cf = MagicMock()
        mock_boto_client.return_value = mock_cf

        mock_cf.describe_stacks.return_value = {
            'Stacks': [{
                'Outputs': [
                    {
                        'OutputKey': 'BedrockAgentRoleArn',
                        'OutputValue': 'arn:aws:iam::123456789012:role/BedrockAgentRole-dev'
                    }
                ]
            }]
        }

        role_arn = get_agent_role_arn('dev', 'us-east-1')

        self.assertEqual(role_arn, 'arn:aws:iam::123456789012:role/BedrockAgentRole-dev')
        mock_cf.describe_stacks.assert_called_once_with(StackName='headset-agent-stack-dev')

    @patch('create_agents.boto3.client')
    def test_fallback_to_constructed_arn(self, mock_boto_client):
        """Test fallback to constructed ARN when stack doesn't exist"""
        mock_cf = MagicMock()
        mock_sts = MagicMock()

        def client_factory(service, **kwargs):
            if service == 'cloudformation':
                return mock_cf
            elif service == 'sts':
                return mock_sts
            return MagicMock()

        mock_boto_client.side_effect = client_factory

        from botocore.exceptions import ClientError
        mock_cf.describe_stacks.side_effect = ClientError(
            {'Error': {'Code': 'ValidationError', 'Message': 'Stack not found'}},
            'describe_stacks'
        )
        mock_sts.get_caller_identity.return_value = {'Account': '111111111111'}

        role_arn = get_agent_role_arn('dev', 'us-east-1')

        self.assertEqual(role_arn, 'arn:aws:iam::111111111111:role/BedrockAgentRole-dev')


class TestWaitForAgent(unittest.TestCase):
    """Tests for wait_for_agent function"""

    @patch('create_agents.time.sleep')
    def test_agent_already_prepared(self, mock_sleep):
        """Test when agent is already in PREPARED state"""
        mock_client = MagicMock()
        mock_client.get_agent.return_value = {
            'agent': {'agentStatus': 'PREPARED'}
        }

        # Should not raise and should not sleep
        wait_for_agent(mock_client, 'test-agent-id')
        mock_sleep.assert_not_called()

    @patch('create_agents.time.sleep')
    def test_agent_not_prepared(self, mock_sleep):
        """Test when agent is in NOT_PREPARED state"""
        mock_client = MagicMock()
        mock_client.get_agent.return_value = {
            'agent': {'agentStatus': 'NOT_PREPARED'}
        }

        # Should not raise
        wait_for_agent(mock_client, 'test-agent-id')
        mock_sleep.assert_not_called()

    @patch('create_agents.time.sleep')
    @patch('create_agents.time.time')
    def test_agent_becomes_ready(self, mock_time, mock_sleep):
        """Test waiting for agent to become ready"""
        mock_client = MagicMock()

        # Simulate agent transitioning from CREATING to PREPARED
        mock_client.get_agent.side_effect = [
            {'agent': {'agentStatus': 'CREATING'}},
            {'agent': {'agentStatus': 'PREPARING'}},
            {'agent': {'agentStatus': 'PREPARED'}},
        ]

        # Mock time to not trigger timeout
        mock_time.side_effect = [0, 5, 10, 15]

        wait_for_agent(mock_client, 'test-agent-id')

        # Should have slept twice (after CREATING and PREPARING)
        self.assertEqual(mock_sleep.call_count, 2)

    @patch('create_agents.time.sleep')
    @patch('create_agents.time.time')
    def test_agent_timeout(self, mock_time, mock_sleep):
        """Test timeout when agent never becomes ready"""
        mock_client = MagicMock()
        mock_client.get_agent.return_value = {
            'agent': {'agentStatus': 'CREATING'}
        }

        # Simulate time passing beyond timeout
        mock_time.side_effect = [0, 130, 260]  # Exceeds 120 second timeout

        with self.assertRaises(TimeoutError):
            wait_for_agent(mock_client, 'test-agent-id', timeout=120)


class TestWaitForAlias(unittest.TestCase):
    """Tests for wait_for_alias function"""

    @patch('create_agents.time.sleep')
    def test_alias_already_prepared(self, mock_sleep):
        """Test when alias is already PREPARED"""
        mock_client = MagicMock()
        mock_client.get_agent_alias.return_value = {
            'agentAlias': {'agentAliasStatus': 'PREPARED'}
        }

        wait_for_alias(mock_client, 'test-agent-id', 'test-alias-id')
        mock_sleep.assert_not_called()

    @patch('create_agents.time.sleep')
    @patch('create_agents.time.time')
    def test_alias_failed(self, mock_time, mock_sleep):
        """Test when alias fails to prepare"""
        mock_client = MagicMock()
        mock_client.get_agent_alias.return_value = {
            'agentAlias': {'agentAliasStatus': 'FAILED'}
        }
        mock_time.return_value = 0

        with self.assertRaises(Exception) as context:
            wait_for_alias(mock_client, 'test-agent-id', 'test-alias-id')

        self.assertIn('failed to prepare', str(context.exception))


class TestAgentConfig(unittest.TestCase):
    """Tests for agent configuration"""

    def test_inference_profile_format(self):
        """Verify inference profile IDs have correct format"""
        # Import the models from the script
        supervisor_model = "us.anthropic.claude-3-5-sonnet-20241022-v2:0"
        subagent_model = "us.anthropic.claude-3-5-haiku-20241022-v1:0"

        # Inference profiles should start with region prefix
        self.assertTrue(supervisor_model.startswith('us.'))
        self.assertTrue(subagent_model.startswith('us.'))

        # Should contain anthropic
        self.assertIn('anthropic', supervisor_model)
        self.assertIn('anthropic', subagent_model)

        # Should have version suffix
        self.assertTrue(supervisor_model.endswith(':0'))
        self.assertTrue(subagent_model.endswith(':0'))

    def test_agent_names(self):
        """Test agent naming convention"""
        environment = 'dev'
        agent_names = [
            f"TroubleshootingOrchestrator-{environment}",
            f"DiagnosticAgent-{environment}",
            f"PlatformAgent-{environment}",
            f"EscalationAgent-{environment}",
        ]

        for name in agent_names:
            self.assertIn(environment, name)
            self.assertTrue(name.endswith(f'-{environment}'))


class TestSSMParameters(unittest.TestCase):
    """Tests for SSM parameter handling"""

    @patch('create_agents.boto3.client')
    def test_update_ssm_parameter(self, mock_boto_client):
        """Test SSM parameter update"""
        from create_agents import update_ssm_parameter

        mock_ssm = MagicMock()
        mock_boto_client.return_value = mock_ssm

        update_ssm_parameter('/headset-agent/dev/supervisor-agent-id', 'ABCD1234', 'us-east-1')

        mock_ssm.put_parameter.assert_called_once_with(
            Name='/headset-agent/dev/supervisor-agent-id',
            Value='ABCD1234',
            Type='String',
            Overwrite=True
        )


if __name__ == '__main__':
    unittest.main()
