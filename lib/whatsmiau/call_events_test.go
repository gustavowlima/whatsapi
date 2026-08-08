package whatsmiau

import (
	"testing"
	"time"

	"github.com/verbeux-ai/whatsmiau/models"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func TestCallOfferWebhookIsSanitizedAndDescribesVideo(t *testing.T) {
	for _, configuredEvent := range []string{"CALL", "call"} {
		t.Run(configuredEvent, func(t *testing.T) {
			s := &Whatsmiau{emitter: make(chan emitter, 1)}
			instance := &models.Instance{
				ID: "instance-1",
				Webhook: models.InstanceWebhook{
					Url:    "https://webhook.example/calls",
					Events: []string{configuredEvent},
				},
			}
			timestamp := time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC)
			event := &events.CallOffer{
				BasicCallMeta: types.BasicCallMeta{
					CallID:         "call-1",
					From:           types.NewJID("5511999999999", types.DefaultUserServer),
					CallCreator:    types.NewJID("123", types.HiddenUserServer),
					CallCreatorAlt: types.NewJID("5511999999999", types.DefaultUserServer),
					Timestamp:      timestamp,
				},
				CallRemoteMeta: types.CallRemoteMeta{RemotePlatform: "android", RemoteVersion: "2.26"},
				Data: &waBinary.Node{
					Tag:     "offer",
					Content: []waBinary.Node{{Tag: "video"}},
				},
			}

			s.handleCallOfferEvent("instance-1", instance, event, webhookEventMap(instance.Webhook.Events))

			select {
			case emitted := <-s.emitter:
				payload, ok := emitted.data.(*WookEvent[WookCallData])
				if !ok {
					t.Fatalf("unexpected emitted data type %T", emitted.data)
				}
				if payload.Event != WookCall || !payload.DateTime.Equal(timestamp) {
					t.Fatalf("unexpected webhook envelope: %+v", payload)
				}
				data := payload.Data
				if data.CallID != "call-1" || data.Status != "offer" || data.Media != "video" {
					t.Fatalf("unexpected call data: %+v", data)
				}
				if data.IsVideo == nil || !*data.IsVideo || data.RTCMediaAvailable {
					t.Fatalf("unexpected media status: %+v", data)
				}
				if data.From != "5511999999999@s.whatsapp.net" || data.CallCreatorAlt != "5511999999999@s.whatsapp.net" {
					t.Fatalf("unexpected caller identity: %+v", data)
				}
			case <-time.After(time.Second):
				t.Fatalf("expected call webhook for configuration %q", configuredEvent)
			}
		})
	}
}

func TestCallEventsAreNotEmittedWithoutCallSubscription(t *testing.T) {
	s := &Whatsmiau{emitter: make(chan emitter, 1)}
	instance := &models.Instance{ID: "instance-1", Webhook: models.InstanceWebhook{Url: "https://webhook.example/calls"}}
	event := &events.CallTerminate{BasicCallMeta: types.BasicCallMeta{CallID: "call-1", Timestamp: time.Now()}, Reason: "timeout"}

	s.handleCallTerminateEvent("instance-1", instance, event, webhookEventMap(instance.Webhook.Events))

	select {
	case emitted := <-s.emitter:
		t.Fatalf("unexpected emitted call event: %#v", emitted)
	default:
	}
}
