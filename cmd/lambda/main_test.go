package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestGetIntAttr(t *testing.T) {
	tests := []struct {
		name     string
		attrs    map[string]string
		key      string
		expected int
	}{
		{
			name:     "existing key with valid int",
			attrs:    map[string]string{"count": "5"},
			key:      "count",
			expected: 5,
		},
		{
			name:     "existing key with zero",
			attrs:    map[string]string{"count": "0"},
			key:      "count",
			expected: 0,
		},
		{
			name:     "missing key",
			attrs:    map[string]string{"other": "1"},
			key:      "count",
			expected: 0,
		},
		{
			name:     "nil map",
			attrs:    nil,
			key:      "count",
			expected: 0,
		},
		{
			name:     "invalid int value",
			attrs:    map[string]string{"count": "abc"},
			key:      "count",
			expected: 0,
		},
		{
			name:     "negative value",
			attrs:    map[string]string{"count": "-3"},
			key:      "count",
			expected: -3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIntAttr(tt.attrs, tt.key)
			if result != tt.expected {
				t.Errorf("getIntAttr() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLexV2Event_Parsing(t *testing.T) {
	eventJSON := `{
		"sessionId": "test-session-123",
		"inputTranscript": "My headset is not working",
		"sessionState": {
			"sessionAttributes": {
				"persona_id": "tangerine",
				"frustration_count": "1"
			}
		}
	}`

	var event LexV2Event
	err := json.Unmarshal([]byte(eventJSON), &event)
	if err != nil {
		t.Fatalf("Failed to unmarshal LexV2Event: %v", err)
	}

	if event.SessionID != "test-session-123" {
		t.Errorf("SessionID = %v, want test-session-123", event.SessionID)
	}
	if event.InputTranscript != "My headset is not working" {
		t.Errorf("InputTranscript = %v, want 'My headset is not working'", event.InputTranscript)
	}
	if event.SessionState.SessionAttributes["persona_id"] != "tangerine" {
		t.Errorf("persona_id = %v, want tangerine", event.SessionState.SessionAttributes["persona_id"])
	}
}

func TestChatRequest_Parsing(t *testing.T) {
	requestJSON := `{
		"sessionId": "web-session-456",
		"inputTranscript": "Hello, I need help",
		"sessionState": {
			"sessionAttributes": {
				"persona_id": "joseph"
			}
		}
	}`

	var req ChatRequest
	err := json.Unmarshal([]byte(requestJSON), &req)
	if err != nil {
		t.Fatalf("Failed to unmarshal ChatRequest: %v", err)
	}

	if req.SessionID != "web-session-456" {
		t.Errorf("SessionID = %v, want web-session-456", req.SessionID)
	}
	if req.InputTranscript != "Hello, I need help" {
		t.Errorf("InputTranscript = %v, want 'Hello, I need help'", req.InputTranscript)
	}
}

func TestChatResponse_Serialization(t *testing.T) {
	response := ChatResponse{
		Messages: []ChatMessage{
			{Content: "Hello! How can I help you today?"},
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal ChatResponse: %v", err)
	}

	// Verify the JSON structure
	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	messages, ok := parsed["messages"].([]interface{})
	if !ok {
		t.Fatal("Response should have messages array")
	}
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	msg := messages[0].(map[string]interface{})
	if msg["content"] != "Hello! How can I help you today?" {
		t.Errorf("Message content mismatch")
	}
}

func TestAPIGatewayEventDetection(t *testing.T) {
	// API Gateway V2 event with HTTP method
	apiEventJSON := `{
		"requestContext": {
			"http": {
				"method": "POST",
				"path": "/chat"
			}
		},
		"body": "{\"sessionId\":\"test\",\"inputTranscript\":\"hello\"}"
	}`

	var apiEvent events.APIGatewayV2HTTPRequest
	err := json.Unmarshal([]byte(apiEventJSON), &apiEvent)
	if err != nil {
		t.Fatalf("Failed to unmarshal API event: %v", err)
	}

	// Verify the HTTP method is detected
	if apiEvent.RequestContext.HTTP.Method != "POST" {
		t.Errorf("HTTP Method = %v, want POST", apiEvent.RequestContext.HTTP.Method)
	}
}

func TestLexEventDetection(t *testing.T) {
	// Lex V2 event (no HTTP context)
	lexEventJSON := `{
		"sessionId": "lex-session",
		"inputTranscript": "test"
	}`

	var lexEvent LexV2Event
	err := json.Unmarshal([]byte(lexEventJSON), &lexEvent)
	if err != nil {
		t.Fatalf("Failed to unmarshal Lex event: %v", err)
	}

	if lexEvent.SessionID != "lex-session" {
		t.Errorf("SessionID = %v, want lex-session", lexEvent.SessionID)
	}
}

func TestSessionState_NilHandling(t *testing.T) {
	// Event with no session state
	eventJSON := `{
		"sessionId": "test-session",
		"inputTranscript": "hello"
	}`

	var event LexV2Event
	err := json.Unmarshal([]byte(eventJSON), &event)
	if err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	// SessionAttributes should be nil
	if event.SessionState.SessionAttributes != nil {
		t.Log("SessionAttributes was initialized, testing empty map handling")
	}

	// getIntAttr should handle nil gracefully
	result := getIntAttr(event.SessionState.SessionAttributes, "any_key")
	if result != 0 {
		t.Errorf("getIntAttr with nil map should return 0, got %d", result)
	}
}

func TestTestInvocation(t *testing.T) {
	// Test invocation event
	eventJSON := `{
		"sessionId": "test-check",
		"inputTranscript": "",
		"sessionState": {
			"sessionAttributes": {
				"test": "true"
			}
		}
	}`

	var event LexV2Event
	err := json.Unmarshal([]byte(eventJSON), &event)
	if err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	// Verify test conditions
	if event.InputTranscript != "" {
		t.Error("Test invocation should have empty transcript")
	}
	if event.SessionState.SessionAttributes["test"] != "true" {
		t.Error("Test invocation should have test=true attribute")
	}
}

// Integration test placeholder - requires AWS credentials
func TestHandleRequest_Integration(t *testing.T) {
	t.Skip("Integration test requires AWS credentials and deployed infrastructure")

	// This would test the full request flow
	ctx := context.Background()
	event := json.RawMessage(`{
		"sessionId": "integration-test",
		"inputTranscript": "hello",
		"sessionState": {
			"sessionAttributes": {}
		}
	}`)

	_, err := handleRequest(ctx, event)
	if err != nil {
		t.Errorf("handleRequest() error = %v", err)
	}
}
