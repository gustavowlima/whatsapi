package whatsmiau

import (
	"testing"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func TestBuildSendTextMessageWithoutQuote(t *testing.T) {
	msg := buildSendTextMessage(&SendText{Text: "hello"})

	if msg.Conversation == nil || msg.GetConversation() != "hello" {
		t.Fatalf("expected Conversation=%q, got %v", "hello", msg.Conversation)
	}
	if msg.ExtendedTextMessage != nil {
		t.Fatalf("expected no ExtendedTextMessage, got %v", msg.ExtendedTextMessage)
	}
}

func TestBuildSendTextMessageQuoteIDOnly(t *testing.T) {
	msg := buildSendTextMessage(&SendText{Text: "reply", Quote: &Quote{MessageID: "ABC123"}})

	ext := msg.ExtendedTextMessage
	if ext == nil {
		t.Fatal("expected ExtendedTextMessage for quoted message")
	}
	if ext.GetText() != "reply" {
		t.Fatalf("expected text=%q, got %q", "reply", ext.GetText())
	}
	if ext.ContextInfo == nil {
		t.Fatal("expected ContextInfo")
	}
	if ext.ContextInfo.GetStanzaID() != "ABC123" {
		t.Fatalf("expected stanzaID=ABC123, got %q", ext.ContextInfo.GetStanzaID())
	}
	if ext.ContextInfo.Participant != nil {
		t.Fatalf("expected no participant, got %q", ext.ContextInfo.GetParticipant())
	}
	if ext.ContextInfo.QuotedMessage != nil {
		t.Fatalf("expected no quotedMessage, got %v", ext.ContextInfo.QuotedMessage)
	}
}

func TestBuildSendTextMessageQuoteWithConversation(t *testing.T) {
	msg := buildSendTextMessage(&SendText{
		Text:  "reply",
		Quote: &Quote{MessageID: "ABC123", Message: "original text"},
	})

	if msg.ExtendedTextMessage == nil {
		t.Fatal("expected ExtendedTextMessage for quoted message")
	}
	ci := msg.ExtendedTextMessage.ContextInfo
	if ci.GetStanzaID() != "ABC123" {
		t.Fatalf("expected stanzaID=ABC123, got %q", ci.GetStanzaID())
	}
	if ci.QuotedMessage == nil || ci.QuotedMessage.GetConversation() != "original text" {
		t.Fatalf("expected quotedMessage conversation=%q, got %v", "original text", ci.QuotedMessage)
	}
}

func TestBuildSendTextMessageQuoteWithParticipant(t *testing.T) {
	participant, err := types.ParseJID("5511999999999@s.whatsapp.net")
	if err != nil {
		t.Fatalf("failed to parse participant JID: %v", err)
	}

	msg := buildSendTextMessage(&SendText{
		Text:  "reply",
		Quote: &Quote{MessageID: "ABC123", Message: "original text", Participant: &participant},
	})

	if msg.ExtendedTextMessage == nil {
		t.Fatal("expected ExtendedTextMessage for quoted message")
	}
	ci := msg.ExtendedTextMessage.ContextInfo
	if ci.GetStanzaID() != "ABC123" {
		t.Fatalf("expected stanzaID=ABC123, got %q", ci.GetStanzaID())
	}
	if ci.GetParticipant() != participant.String() {
		t.Fatalf("expected participant=%q, got %q", participant.String(), ci.GetParticipant())
	}
	if ci.QuotedMessage == nil {
		t.Fatal("expected quotedMessage")
	}
}

func TestBuildSendTextMessageQuoteTextPreserved(t *testing.T) {
	// The reply text must go into ExtendedTextMessage.Text, not Conversation.
	msg := buildSendTextMessage(&SendText{Text: "reply", Quote: &Quote{MessageID: "ABC123"}})

	if msg.GetConversation() != "" {
		t.Fatalf("expected no Conversation field, got %q", msg.GetConversation())
	}
	if msg.ExtendedTextMessage.GetText() != "reply" {
		t.Fatalf("expected text=%q, got %q", "reply", msg.ExtendedTextMessage.GetText())
	}
	if !proto.Equal(msg, buildSendTextMessage(&SendText{Text: "reply", Quote: &Quote{MessageID: "ABC123"}})) {
		t.Fatal("expected deterministic message construction")
	}
}

func TestBuildContextInfoNoQuote(t *testing.T) {
	if ci := buildContextInfo(nil); ci != nil {
		t.Fatalf("expected nil ContextInfo, got %v", ci)
	}
	if ci := buildContextInfo(&Quote{}); ci != nil {
		t.Fatalf("expected nil ContextInfo for empty quote, got %v", ci)
	}
}

func TestBuildContextInfoQuoteIDOnly(t *testing.T) {
	ci := buildContextInfo(&Quote{MessageID: "ABC123"})

	if ci.GetStanzaID() != "ABC123" {
		t.Fatalf("expected stanzaID=ABC123, got %q", ci.GetStanzaID())
	}
	if ci.Participant != nil {
		t.Fatalf("expected no participant, got %q", ci.GetParticipant())
	}
	if ci.QuotedMessage != nil {
		t.Fatalf("expected no quotedMessage, got %v", ci.QuotedMessage)
	}
}

func TestBuildContextInfoWithContentAndParticipant(t *testing.T) {
	participant, err := types.ParseJID("5511999999999@s.whatsapp.net")
	if err != nil {
		t.Fatalf("failed to parse participant JID: %v", err)
	}

	ci := buildContextInfo(&Quote{
		MessageID:   "ABC123",
		Message:     "original text",
		Participant: &participant,
	})

	if ci.GetStanzaID() != "ABC123" {
		t.Fatalf("expected stanzaID=ABC123, got %q", ci.GetStanzaID())
	}
	if ci.GetParticipant() != participant.String() {
		t.Fatalf("expected participant=%q, got %q", participant.String(), ci.GetParticipant())
	}
	if ci.QuotedMessage == nil || ci.QuotedMessage.GetConversation() != "original text" {
		t.Fatalf("expected quotedMessage conversation=%q, got %v", "original text", ci.QuotedMessage)
	}
}

func TestBuildContactMessageWithQuote(t *testing.T) {
	// Mirrors SendContact's single-contact branch (send_more.go): the ContextInfo
	// must be attached to the ContactMessage and survive proto marshaling.
	contact := contactToProto(SendContactItem{FullName: "Ivo Silva", PhoneNumber: "5511999997777"})
	contact.ContextInfo = buildContextInfo(&Quote{MessageID: "ABC123", Message: "h"})
	msg := &waE2E.Message{ContactMessage: contact}

	ci := msg.ContactMessage.GetContextInfo()
	if ci == nil {
		t.Fatal("expected ContextInfo on ContactMessage")
	}
	if ci.GetStanzaID() != "ABC123" {
		t.Fatalf("expected stanzaID=ABC123, got %q", ci.GetStanzaID())
	}
	if ci.QuotedMessage == nil || ci.QuotedMessage.GetConversation() != "h" {
		t.Fatalf("expected quotedMessage conversation=%q, got %v", "h", ci.QuotedMessage)
	}

	if _, err := proto.Marshal(msg); err != nil {
		t.Fatalf("expected message to marshal, got %v", err)
	}
}
