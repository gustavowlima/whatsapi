package whatsmiau

import (
	"encoding/json"
	"testing"
)

func TestWebhookEventMapNormalizesConfigurationAliases(t *testing.T) {
	tests := []struct {
		input    string
		expected webhookConfigEvent
	}{
		{input: "MESSAGES_UPSERT", expected: webhookConfigMessagesUpsert},
		{input: "messages.upsert", expected: webhookConfigMessagesUpsert},
		{input: "  Messages.Update  ", expected: webhookConfigMessagesUpdate},
		{input: "messages.delete", expected: webhookConfigMessagesDelete},
		{input: "messages.set", expected: webhookConfigMessagesSet},
		{input: "contacts.upsert", expected: webhookConfigContactsUpsert},
		{input: "group-participants.update", expected: webhookConfigGroupParticipantsUpdate},
		{input: "CONNECTION_UPDATE", expected: webhookConfigConnectionUpdate},
		{input: "connection.update", expected: webhookConfigConnectionUpdate},
		{input: "CALL", expected: webhookConfigCall},
		{input: "call", expected: webhookConfigCall},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			eventMap := webhookEventMap([]string{tt.input})
			if !eventMap[tt.expected] {
				t.Fatalf("expected normalized event %q in %#v", tt.expected, eventMap)
			}
		})
	}
}

func TestWebhookEventMapDoesNotEnableSupportedEventsForEmptyOrUnknownInput(t *testing.T) {
	supportedEvents := []webhookConfigEvent{
		webhookConfigMessagesUpsert,
		webhookConfigMessagesUpdate,
		webhookConfigMessagesDelete,
		webhookConfigMessagesSet,
		webhookConfigContactsUpsert,
		webhookConfigGroupParticipantsUpdate,
		webhookConfigConnectionUpdate,
		webhookConfigCall,
	}

	for _, input := range []string{"", "   ", "unknown.event"} {
		t.Run(input, func(t *testing.T) {
			eventMap := webhookEventMap([]string{input})
			for _, event := range supportedEvents {
				if eventMap[event] {
					t.Fatalf("input %q unexpectedly enabled supported event %q", input, event)
				}
			}
		})
	}
}

func TestWebhookPayloadEventIdentifiersRemainLowercase(t *testing.T) {
	tests := []struct {
		event    Wook
		expected string
	}{
		{event: WookMessagesUpsert, expected: "messages.upsert"},
		{event: WookGroupParticipantsUpdate, expected: "group-participants.update"},
		{event: WookConnectionUpdate, expected: "connection.update"},
		{event: WookCall, expected: "call"},
	}

	for _, tt := range tests {
		t.Run(string(tt.event), func(t *testing.T) {
			payload, err := json.Marshal(WookEvent[struct{}]{Event: tt.event})
			if err != nil {
				t.Fatalf("marshal webhook event: %v", err)
			}

			var envelope struct {
				Event string `json:"event"`
			}
			if err := json.Unmarshal(payload, &envelope); err != nil {
				t.Fatalf("unmarshal webhook event: %v", err)
			}
			if envelope.Event != tt.expected {
				t.Fatalf("expected event %q, got %q", tt.expected, envelope.Event)
			}
		})
	}
}
