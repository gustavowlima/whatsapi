# Webhook Event Contract Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Preserve Evolution API webhook configuration identifiers while accepting legacy Manager UI aliases and keeping emitted payload event values unchanged.

**Architecture:** Introduce a package-private configuration event type and a single normalizer that builds typed subscription maps. Keep `Wook` constants dedicated to payload serialization, route every subscription path through the normalizer, and document the two namespaces explicitly.

**Tech Stack:** Go 1.25, standard library `strings` and `testing`, `xsync/v4`, Git, GitHub CLI

---

## File Structure

- Create `lib/whatsmiau/webhook_events.go` to own configuration event identifiers and normalization.
- Create `lib/whatsmiau/webhook_events_test.go` to own normalization, compatibility, connection-path, and payload-contract tests.
- Modify `lib/whatsmiau/event_emitter.go` to remove its local normalizer, use typed configuration keys, and normalize direct connection updates.
- Modify `lib/whatsmiau/call_events.go` to use the typed `CALL` subscription key.
- Modify `lib/whatsmiau/call_events_test.go` to retain call-specific tests only.
- Modify `README.md` to distinguish configured event values from emitted payload values.

### Task 1: Separate Configuration Events From Payload Events

**Files:**

- Create: `lib/whatsmiau/webhook_events.go`
- Create: `lib/whatsmiau/webhook_events_test.go`
- Modify: `lib/whatsmiau/event_emitter.go:280-795`
- Modify: `lib/whatsmiau/call_events.go:14-74`
- Modify: `lib/whatsmiau/call_events_test.go:13-20`

- [ ] **Step 1: Write failing normalization tests**

Create `lib/whatsmiau/webhook_events_test.go` with the following content:

```go
package whatsmiau

import (
	"encoding/json"
	"testing"
)

func TestWebhookEventMapNormalizesConfigurationAliases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  webhookConfigEvent
	}{
		{name: "canonical message", input: "MESSAGES_UPSERT", want: webhookConfigMessagesUpsert},
		{name: "payload message", input: "messages.upsert", want: webhookConfigMessagesUpsert},
		{name: "trimmed mixed case", input: "  Messages.Update  ", want: webhookConfigMessagesUpdate},
		{name: "message delete", input: "messages.delete", want: webhookConfigMessagesDelete},
		{name: "message history", input: "messages.set", want: webhookConfigMessagesSet},
		{name: "contact", input: "contacts.upsert", want: webhookConfigContactsUpsert},
		{name: "hyphenated group event", input: "group-participants.update", want: webhookConfigGroupParticipantsUpdate},
		{name: "canonical connection", input: "CONNECTION_UPDATE", want: webhookConfigConnectionUpdate},
		{name: "payload connection", input: "connection.update", want: webhookConfigConnectionUpdate},
		{name: "canonical call", input: "CALL", want: webhookConfigCall},
		{name: "payload call", input: "call", want: webhookConfigCall},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			eventMap := webhookEventMap([]string{test.input})
			if !eventMap[test.want] {
				t.Fatalf("expected %q to enable %q in %#v", test.input, test.want, eventMap)
			}
		})
	}
}

func TestWebhookEventMapDoesNotEnableSupportedEventsForEmptyOrUnknownInput(t *testing.T) {
	eventMap := webhookEventMap([]string{"", "   ", "unknown.event"})
	supported := []webhookConfigEvent{
		webhookConfigMessagesUpsert,
		webhookConfigMessagesUpdate,
		webhookConfigMessagesDelete,
		webhookConfigMessagesSet,
		webhookConfigContactsUpsert,
		webhookConfigGroupParticipantsUpdate,
		webhookConfigConnectionUpdate,
		webhookConfigCall,
	}

	for _, event := range supported {
		if eventMap[event] {
			t.Fatalf("unexpected supported event %q in %#v", event, eventMap)
		}
	}
}

func TestWebhookPayloadEventIdentifiersRemainLowercase(t *testing.T) {
	tests := []struct {
		name  string
		event Wook
		want  string
	}{
		{name: "message", event: WookMessagesUpsert, want: "messages.upsert"},
		{name: "group", event: WookGroupParticipantsUpdate, want: "group-participants.update"},
		{name: "connection", event: WookConnectionUpdate, want: "connection.update"},
		{name: "call", event: WookCall, want: "call"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := json.Marshal(WookEvent[struct{}]{Event: test.event})
			if err != nil {
				t.Fatalf("marshal webhook payload: %v", err)
			}

			var payload struct {
				Event string `json:"event"`
			}
			if err := json.Unmarshal(encoded, &payload); err != nil {
				t.Fatalf("unmarshal webhook payload: %v", err)
			}
			if payload.Event != test.want {
				t.Fatalf("expected payload event %q, got %q", test.want, payload.Event)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:

```bash
go test ./lib/whatsmiau -run 'TestWebhookEventMap' -count=1
```

Expected: FAIL to compile because `webhookConfigEvent` and its constants do not exist.

- [ ] **Step 3: Add the typed configuration event model and normalizer**

Create `lib/whatsmiau/webhook_events.go`:

```go
package whatsmiau

import "strings"

type webhookConfigEvent string

const (
	webhookConfigMessagesUpsert          webhookConfigEvent = "MESSAGES_UPSERT"
	webhookConfigMessagesUpdate          webhookConfigEvent = "MESSAGES_UPDATE"
	webhookConfigMessagesDelete          webhookConfigEvent = "MESSAGES_DELETE"
	webhookConfigMessagesSet             webhookConfigEvent = "MESSAGES_SET"
	webhookConfigContactsUpsert          webhookConfigEvent = "CONTACTS_UPSERT"
	webhookConfigGroupParticipantsUpdate webhookConfigEvent = "GROUP_PARTICIPANTS_UPDATE"
	webhookConfigConnectionUpdate        webhookConfigEvent = "CONNECTION_UPDATE"
	webhookConfigCall                    webhookConfigEvent = "CALL"
)

var webhookConfigEventReplacer = strings.NewReplacer(".", "_", "-", "_")

func normalizeWebhookConfigEvent(event string) webhookConfigEvent {
	normalized := strings.ToUpper(strings.TrimSpace(event))
	return webhookConfigEvent(webhookConfigEventReplacer.Replace(normalized))
}

func webhookEventMap(events []string) map[webhookConfigEvent]bool {
	eventMap := make(map[webhookConfigEvent]bool, len(events))
	for _, event := range events {
		normalized := normalizeWebhookConfigEvent(event)
		if normalized != "" {
			eventMap[normalized] = true
		}
	}
	return eventMap
}
```

Delete the existing `webhookEventMap` function from `lib/whatsmiau/event_emitter.go`. Keep its `strings` import because the file uses `strings` elsewhere.

- [ ] **Step 4: Convert handler signatures and subscription checks**

Change the `eventMap` parameter from `map[string]bool` to `map[webhookConfigEvent]bool` in these functions:

```text
lib/whatsmiau/event_emitter.go
handleMessageEvent
handleMessageDeleteEvent
handleReceiptEvent
handleBusinessNameEvent
handleContactEvent
handlePictureEvent
handleHistorySyncEvent
handleGroupInfoEvent
handleGroupParticipantsUpdateEvent
handleJoinedGroupEvent
handlePushNameEvent
handleConnectionUpdateEvent

lib/whatsmiau/call_events.go
handleCallOfferEvent
handleCallOfferNoticeEvent
handleCallPreAcceptEvent
handleCallAcceptEvent
handleCallTransportEvent
handleCallRelayLatencyEvent
handleCallRejectEvent
handleCallTerminateEvent
emitCallEvent
```

Replace every string-key check with its configuration constant:

```text
eventMap["MESSAGES_UPSERT"]          -> eventMap[webhookConfigMessagesUpsert]
eventMap["MESSAGES_UPDATE"]          -> eventMap[webhookConfigMessagesUpdate]
eventMap["MESSAGES_DELETE"]          -> eventMap[webhookConfigMessagesDelete]
eventMap["MESSAGES_SET"]             -> eventMap[webhookConfigMessagesSet]
eventMap["CONTACTS_UPSERT"]          -> eventMap[webhookConfigContactsUpsert]
eventMap["GROUP_PARTICIPANTS_UPDATE"] -> eventMap[webhookConfigGroupParticipantsUpdate]
eventMap["CONNECTION_UPDATE"]        -> eventMap[webhookConfigConnectionUpdate]
eventMap["CALL"]                     -> eventMap[webhookConfigCall]
```

Do not change `WookEvent.Event` assignments or any `Wook*` constant.

Keep the direct connection path behavior unchanged during this type-only refactor, but make its raw map compile with the new handler signature:

```go
eventMap := make(map[webhookConfigEvent]bool)
for _, evt := range instance.Webhook.Events {
	eventMap[webhookConfigEvent(evt)] = true
}
```

Task 2 replaces this temporary raw construction with the shared normalizer after proving the compatibility failure.

- [ ] **Step 5: Remove the old general-purpose test from the call test file and format**

Delete `TestWebhookEventMapNormalizesManagerAndCanonicalValues` from `lib/whatsmiau/call_events_test.go`. The new test file now owns this behavior.

Run:

```bash
gofmt -w lib/whatsmiau/webhook_events.go lib/whatsmiau/webhook_events_test.go lib/whatsmiau/event_emitter.go lib/whatsmiau/call_events.go lib/whatsmiau/call_events_test.go
```

Expected: no output.

- [ ] **Step 6: Run focused tests and structural checks**

Run:

```bash
go test ./lib/whatsmiau -run 'TestWebhookEventMap|TestWebhookPayloadEventIdentifiersRemainLowercase|TestCall' -count=1
```

Expected: PASS.

Run:

```bash
rg -n 'eventMap map\[string\]bool|eventMap\["(MESSAGES|CONTACTS|GROUP_PARTICIPANTS|CONNECTION|CALL)' lib/whatsmiau/event_emitter.go lib/whatsmiau/call_events.go
```

Expected: no output.

- [ ] **Step 7: Commit the configuration event separation**

```bash
git add lib/whatsmiau/webhook_events.go lib/whatsmiau/webhook_events_test.go lib/whatsmiau/event_emitter.go lib/whatsmiau/call_events.go lib/whatsmiau/call_events_test.go
git commit -m "refactor(webhook): separate config event identifiers" -m "- Type webhook subscription keys independently from payload events
- Normalize canonical and legacy Manager UI values in one helper"
```

### Task 2: Normalize Direct Connection Updates

**Files:**

- Modify: `lib/whatsmiau/webhook_events_test.go`
- Modify: `lib/whatsmiau/event_emitter.go:785-797`

- [ ] **Step 1: Write the failing direct-path regression test**

Extend the import block in `lib/whatsmiau/webhook_events_test.go` and add the test below:

```go
import (
	"encoding/json"
	"testing"
	"time"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/verbeux-ai/whatsmiau/models"
)

func TestEmitConnectionUpdateAcceptsCanonicalAndPayloadConfiguration(t *testing.T) {
	for _, configuredEvent := range []string{"CONNECTION_UPDATE", "connection.update"} {
		t.Run(configuredEvent, func(t *testing.T) {
			enabled := true
			service := &Whatsmiau{
				instanceCache: xsync.NewMap[string, models.Instance](),
				emitter:       make(chan emitter, 1),
			}
			service.instanceCache.Store("instance-1", models.Instance{
				ID: "instance-1",
				Webhook: models.InstanceWebhook{
					Enabled: &enabled,
					Url:     "https://webhook.example/connection",
					Events:  []string{configuredEvent},
				},
			})

			service.emitConnectionUpdate("instance-1", "connecting", 0)

			select {
			case emitted := <-service.emitter:
				payload, ok := emitted.data.(*WookEvent[WookConnectionUpdateData])
				if !ok {
					t.Fatalf("unexpected emitted data type %T", emitted.data)
				}
				if emitted.url != "https://webhook.example/connection" {
					t.Fatalf("unexpected webhook URL %q", emitted.url)
				}
				if payload.Event != WookConnectionUpdate {
					t.Fatalf("unexpected payload event %q", payload.Event)
				}
			case <-time.After(time.Second):
				t.Fatalf("expected connection webhook for configuration %q", configuredEvent)
			}
		})
	}
}
```

- [ ] **Step 2: Run the regression test to verify it fails**

Run:

```bash
go test ./lib/whatsmiau -run TestEmitConnectionUpdateAcceptsCanonicalAndPayloadConfiguration -count=1
```

Expected: FAIL for the `connection.update` case because `emitConnectionUpdate` still builds a raw map.

- [ ] **Step 3: Route the direct path through the shared normalizer**

Replace the raw map construction in `emitConnectionUpdate` with:

```go
eventMap := webhookEventMap(instance.Webhook.Events)
s.handleConnectionUpdateEvent(id, instance, state, statusReason, eventMap)
```

The complete function becomes:

```go
func (s *Whatsmiau) emitConnectionUpdate(id string, state string, statusReason int) {
	instance := s.getInstanceCached(id)
	if instance == nil || instance.Webhook.Enabled == nil || !*instance.Webhook.Enabled {
		return
	}

	eventMap := webhookEventMap(instance.Webhook.Events)
	s.handleConnectionUpdateEvent(id, instance, state, statusReason, eventMap)
}
```

- [ ] **Step 4: Format and rerun the regression test**

Run:

```bash
gofmt -w lib/whatsmiau/webhook_events_test.go lib/whatsmiau/event_emitter.go
go test ./lib/whatsmiau -run TestEmitConnectionUpdateAcceptsCanonicalAndPayloadConfiguration -count=1
```

Expected: PASS for both `CONNECTION_UPDATE` and `connection.update`. The assertions also prove that the payload still emits `connection.update` through `WookConnectionUpdate`.

- [ ] **Step 5: Commit the direct-path fix**

```bash
git add lib/whatsmiau/webhook_events_test.go lib/whatsmiau/event_emitter.go
git commit -m "fix(webhook): normalize direct connection updates" -m "- Use the shared subscription map for lifecycle emissions
- Preserve the lowercase connection.update payload event"
```

### Task 3: Document the Two Event Namespaces

**Files:**

- Modify: `README.md:181-197`

- [ ] **Step 1: Replace the supported-events section**

Replace the existing `Supported Events` introduction and table with:

```markdown
## Supported Events

Webhook configuration and webhook payloads use different event identifiers to preserve Evolution API compatibility. Configure subscriptions with the uppercase value. The emitted payload uses the lowercase value in its `event` field.

| Configuration value | Payload `event` value | Description |
|---------------------|-----------------------|-------------|
| `MESSAGES_UPSERT` | `messages.upsert` | Triggered when a new message is received. |
| `MESSAGES_UPDATE` | `messages.update` | Triggered when a message status changes, such as a read receipt. |
| `MESSAGES_DELETE` | `messages.delete` | Triggered when a message is deleted for everyone. |
| `MESSAGES_SET` | `messages.set` | Triggered when message history is synchronized. |
| `CONTACTS_UPSERT` | `contacts.upsert` | Triggered when a contact is created or updated. |
| `GROUP_PARTICIPANTS_UPDATE` | `group-participants.update` | Triggered when group membership changes. |
| `CONNECTION_UPDATE` | `connection.update` | Triggered when connection state changes. |
| `CALL` | `call` | Triggered for call signaling; no media, SDP, or call key is included. |

Payload-style values such as `messages.upsert` remain accepted in configuration for compatibility with older Manager UI versions. WhatsMiau stores and returns configured values as provided; it normalizes them only when matching subscriptions.
```

- [ ] **Step 2: Verify the documented mapping and run focused tests**

Run:

```bash
rg -n 'Configuration value|MESSAGES_UPSERT|messages\.upsert|GROUP_PARTICIPANTS_UPDATE|group-participants\.update|CONNECTION_UPDATE|connection\.update' README.md
```

Expected: the heading and both forms of each sampled event appear in the supported-events section.

Run:

```bash
go test ./lib/whatsmiau -count=1
```

Expected: PASS.

- [ ] **Step 3: Commit the contract documentation**

```bash
git add README.md
git commit -m "docs: clarify webhook event namespaces" -m "- Document canonical configuration values beside payload values
- Explain compatibility aliases and unchanged persistence"
```

### Task 4: Verify the Complete Change

**Files:**

- Verify: `lib/whatsmiau/webhook_events.go`
- Verify: `lib/whatsmiau/webhook_events_test.go`
- Verify: `lib/whatsmiau/event_emitter.go`
- Verify: `lib/whatsmiau/call_events.go`
- Verify: `README.md`

- [ ] **Step 1: Check formatting**

Run:

```bash
gofmt -l lib/whatsmiau/webhook_events.go lib/whatsmiau/webhook_events_test.go lib/whatsmiau/event_emitter.go lib/whatsmiau/call_events.go lib/whatsmiau/call_events_test.go
```

Expected: no output.

- [ ] **Step 2: Run the package regression suite**

Run:

```bash
go test ./lib/whatsmiau -count=1
```

Expected: PASS with zero failures.

- [ ] **Step 3: Run the complete default Go test suite**

Run:

```bash
go test ./... -count=1
```

Expected: PASS. Integration tests tagged with `integration` remain excluded from the default suite.

- [ ] **Step 4: Check the final diff and branch state**

Run:

```bash
git diff origin/develop...HEAD --check
git diff origin/develop...HEAD --stat
git status --short --branch
```

Expected: no whitespace errors, only the planned files differ from `origin/develop`, and the working tree is clean.

Run:

```bash
rg -n 'make\(map\[(string|webhookConfigEvent)\]bool' lib/whatsmiau/event_emitter.go lib/whatsmiau/call_events.go lib/whatsmiau/webhook_events.go
```

Expected: the only subscription-map construction is inside `webhookEventMap` in `webhook_events.go`.

### Task 5: Publish the Replacement Pull Request

**Files:** None

- [ ] **Step 1: Confirm branch ancestry and commit scope**

Run:

```bash
git merge-base --is-ancestor origin/develop HEAD
git log --oneline origin/develop..HEAD
```

Expected: the ancestry command exits with status 0, and the log contains only the approved specification, plan, implementation, tests, and documentation commits.

- [ ] **Step 2: Push the feature branch**

Run:

```bash
git push -u origin fix/webhook-event-contract
```

Expected: the branch is published and tracks `origin/fix/webhook-event-contract`.

- [ ] **Step 3: Open the replacement PR against `develop`**

Run:

```bash
gh pr create --repo verbeux-ai/whatsmiau --base develop --head fix/webhook-event-contract --title "fix: preserve webhook event configuration contract" --body '## Summary
- separate canonical webhook configuration identifiers from payload event values
- accept legacy Manager UI aliases through centralized normalization
- normalize direct connection updates and document both namespaces

## Testing
- go test ./lib/whatsmiau -count=1
- go test ./... -count=1

Fixes #78.
Supersedes #79.'
```

Expected: GitHub creates an open PR whose base is `develop` and whose head is `fix/webhook-event-contract`.

- [ ] **Step 4: Verify the replacement PR before closing the old PR**

Run:

```bash
gh pr view --repo verbeux-ai/whatsmiau --json number,url,state,baseRefName,headRefName,statusCheckRollup
```

Expected: `state` is `OPEN`, `baseRefName` is `develop`, and `headRefName` is `fix/webhook-event-contract`. Wait for required checks to finish before closing #79.

- [ ] **Step 5: Close #79 as superseded**

Resolve the replacement PR number and close the old PR:

```bash
replacement_pr_number="$(gh pr view --repo verbeux-ai/whatsmiau --json number --jq .number)"
gh pr close 79 --repo verbeux-ai/whatsmiau --comment "Superseded by #${replacement_pr_number}, rebuilt from develop with separate configuration and payload event contracts."
```

Expected: PR #79 is closed with a link to the open replacement PR.
