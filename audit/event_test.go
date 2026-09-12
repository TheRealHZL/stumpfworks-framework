package audit

import (
	"testing"
	"time"
)

func TestEventValidate(t *testing.T) {
	event := Event{ID: "event-1", Timestamp: time.Now(), Actor: "user:1", Action: "door.open", Resource: "door:1", Result: ResultSuccess}
	if err := event.Validate(); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}
	event.Result = "maybe"
	if err := event.Validate(); err == nil {
		t.Fatal("invalid result accepted")
	}
}

func TestEventRejectsSensitiveNestedMetadata(t *testing.T) {
	event := Event{ID: "event-1", Timestamp: time.Now(), Actor: "user:1", Action: "login", Resource: "session:1", Result: ResultSuccess, Metadata: map[string]any{"details": map[string]string{"access_token": "private"}}}
	if err := event.Validate(); err == nil {
		t.Fatal("sensitive metadata accepted")
	}
}
