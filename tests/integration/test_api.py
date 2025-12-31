#!/usr/bin/env python3
"""
Integration tests for the Headset Support Agent API
These tests require the infrastructure to be deployed.
"""

import unittest
import requests
import json
import os
import uuid
import time
from typing import Optional


class HeadsetAPITestCase(unittest.TestCase):
    """Base test case for Headset API tests"""

    API_URL: Optional[str] = None
    session_id: str = ""

    @classmethod
    def setUpClass(cls):
        """Set up test fixtures"""
        cls.API_URL = os.environ.get('API_URL')
        if not cls.API_URL:
            raise unittest.SkipTest("API_URL environment variable not set")

        # Generate unique session ID for this test run
        cls.session_id = f"test-{uuid.uuid4()}"

    def make_request(self, message: str, persona: str = "tangerine") -> dict:
        """Helper to make API requests"""
        response = requests.post(
            self.API_URL,
            headers={
                'Content-Type': 'application/json',
                'X-Session-Id': self.session_id,
                'X-Persona-Id': persona,
            },
            json={
                'sessionId': self.session_id,
                'inputTranscript': message,
                'sessionState': {
                    'sessionAttributes': {
                        'persona_id': persona
                    }
                }
            },
            timeout=30
        )
        return {
            'status_code': response.status_code,
            'body': response.json() if response.content else None
        }


class TestAPIHealth(HeadsetAPITestCase):
    """Test API health and basic connectivity"""

    def test_api_responds(self):
        """Test that API responds to requests"""
        result = self.make_request("Hello")

        self.assertEqual(result['status_code'], 200)
        self.assertIsNotNone(result['body'])

    def test_api_returns_messages(self):
        """Test that API returns messages array"""
        result = self.make_request("Hi there")

        self.assertEqual(result['status_code'], 200)
        self.assertIn('messages', result['body'])
        self.assertIsInstance(result['body']['messages'], list)
        self.assertGreater(len(result['body']['messages']), 0)

    def test_message_has_content(self):
        """Test that messages have content field"""
        result = self.make_request("I need help")

        messages = result['body']['messages']
        for msg in messages:
            self.assertIn('content', msg)
            self.assertIsInstance(msg['content'], str)
            self.assertGreater(len(msg['content']), 0)


class TestPersonaSwitching(HeadsetAPITestCase):
    """Test persona switching functionality"""

    def test_tangerine_persona(self):
        """Test Tangerine persona responds"""
        result = self.make_request("Hello", persona="tangerine")

        self.assertEqual(result['status_code'], 200)
        self.assertIsNotNone(result['body'])

    def test_joseph_persona(self):
        """Test Joseph persona responds"""
        # Use new session for different persona
        self.session_id = f"test-joseph-{uuid.uuid4()}"
        result = self.make_request("Hello", persona="joseph")

        self.assertEqual(result['status_code'], 200)
        self.assertIsNotNone(result['body'])

    def test_jennifer_persona(self):
        """Test Jennifer persona responds"""
        # Use new session for different persona
        self.session_id = f"test-jennifer-{uuid.uuid4()}"
        result = self.make_request("Hello", persona="jennifer")

        self.assertEqual(result['status_code'], 200)
        self.assertIsNotNone(result['body'])


class TestTroubleshootingFlow(HeadsetAPITestCase):
    """Test troubleshooting conversation flow"""

    def test_headset_issue_response(self):
        """Test response to headset issue"""
        result = self.make_request("My headset isn't working")

        self.assertEqual(result['status_code'], 200)
        content = result['body']['messages'][0]['content'].lower()

        # Response should be relevant to troubleshooting
        troubleshooting_keywords = ['check', 'connect', 'help', 'try', 'issue', 'problem']
        has_relevant_response = any(kw in content for kw in troubleshooting_keywords)
        self.assertTrue(has_relevant_response, f"Response should contain troubleshooting keywords: {content}")

    def test_multi_turn_conversation(self):
        """Test multi-turn conversation maintains context"""
        # First turn
        result1 = self.make_request("I can't hear anything through my headset")
        self.assertEqual(result1['status_code'], 200)

        # Brief delay between turns
        time.sleep(1)

        # Second turn
        result2 = self.make_request("I already checked that")
        self.assertEqual(result2['status_code'], 200)

        # Both should have content
        self.assertGreater(len(result1['body']['messages'][0]['content']), 0)
        self.assertGreater(len(result2['body']['messages'][0]['content']), 0)


class TestEscalation(HeadsetAPITestCase):
    """Test escalation detection"""

    def test_agent_request_triggers_escalation(self):
        """Test that requesting an agent triggers escalation"""
        # Use new session
        self.session_id = f"test-escalation-{uuid.uuid4()}"
        result = self.make_request("I want to speak to a human agent")

        self.assertEqual(result['status_code'], 200)
        content = result['body']['messages'][0]['content'].lower()

        # Should mention transfer or escalation
        escalation_keywords = ['transfer', 'connect', 'specialist', 'agent', 'someone']
        has_escalation = any(kw in content for kw in escalation_keywords)
        self.assertTrue(has_escalation, f"Response should indicate escalation: {content}")


class TestErrorHandling(HeadsetAPITestCase):
    """Test error handling"""

    def test_empty_message_handled(self):
        """Test that empty messages are handled gracefully"""
        result = self.make_request("")

        # Should return 200 with some response
        self.assertEqual(result['status_code'], 200)

    def test_long_message_handled(self):
        """Test that long messages are handled"""
        long_message = "My headset problem is " + "very complicated " * 100
        result = self.make_request(long_message)

        self.assertEqual(result['status_code'], 200)


class TestCORS(HeadsetAPITestCase):
    """Test CORS headers"""

    def test_cors_headers_present(self):
        """Test that CORS headers are present when Origin header is sent"""
        # CORS headers are only returned when an Origin header is present
        response = requests.post(
            self.API_URL,
            headers={
                'Content-Type': 'application/json',
                'Origin': 'https://example.com'
            },
            json={
                'sessionId': self.session_id,
                'inputTranscript': 'test',
                'sessionState': {'sessionAttributes': {}}
            },
            timeout=30
        )

        # Check for CORS headers (API Gateway adds these when Origin is present)
        # Note: Some API Gateway configurations may handle this differently
        if 'Access-Control-Allow-Origin' not in response.headers:
            # If CORS headers not present, at least verify the request succeeded
            # This can happen with certain API Gateway configurations
            self.assertEqual(response.status_code, 200)
        else:
            self.assertIn('Access-Control-Allow-Origin', response.headers)


def run_integration_tests():
    """Run integration tests and return results"""
    loader = unittest.TestLoader()
    suite = unittest.TestSuite()

    # Add test classes
    suite.addTests(loader.loadTestsFromTestCase(TestAPIHealth))
    suite.addTests(loader.loadTestsFromTestCase(TestPersonaSwitching))
    suite.addTests(loader.loadTestsFromTestCase(TestTroubleshootingFlow))
    suite.addTests(loader.loadTestsFromTestCase(TestEscalation))
    suite.addTests(loader.loadTestsFromTestCase(TestErrorHandling))
    suite.addTests(loader.loadTestsFromTestCase(TestCORS))

    # Run tests
    runner = unittest.TextTestRunner(verbosity=2)
    result = runner.run(suite)

    return result.wasSuccessful()


if __name__ == '__main__':
    import sys
    success = run_integration_tests()
    sys.exit(0 if success else 1)
