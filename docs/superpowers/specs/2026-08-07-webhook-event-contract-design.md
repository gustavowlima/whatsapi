# Webhook Event Contract Design

## Status

Approved for implementation planning on 2026-08-07.

## Context

WhatsMiau uses two event identifiers for different parts of its Evolution API compatibility contract:

- Webhook configuration uses canonical uppercase identifiers such as `MESSAGES_UPSERT`.
- Emitted webhook payloads use lowercase identifiers such as `messages.upsert` in the `event` field.

Pull request #79 changed subscription checks to use the payload identifiers. That fixes configurations written by the former Manager UI, but it breaks existing Evolution API configurations that contain `MESSAGES_UPSERT`. The `develop` branch already normalizes most configured values before subscription checks, but the code still represents configuration identifiers as string literals and bypasses normalization in `emitConnectionUpdate`.

## Goals

- Preserve uppercase Evolution API identifiers as the canonical webhook configuration contract.
- Preserve lowercase dot-separated identifiers in emitted webhook payloads.
- Accept payload-style identifiers in configuration as a compatibility alias for values previously written by the Manager UI.
- Make the distinction between configuration and payload identifiers explicit in code and documentation.
- Apply one normalization path to every webhook subscription check.
- Add focused regression coverage for both identifier formats.

## Non-goals

- Rewriting stored webhook configurations.
- Changing the JSON shape or the `event` value of emitted webhook payloads.
- Restoring or changing the Manager UI removed from `develop`.
- Rejecting unknown configuration identifiers or adding request validation.
- Refactoring webhook delivery, retry, or persistence behavior.

## Contract

The public contract keeps separate configuration and payload identifiers:

| Configuration value | Payload `event` value |
| --- | --- |
| `MESSAGES_UPSERT` | `messages.upsert` |
| `MESSAGES_UPDATE` | `messages.update` |
| `MESSAGES_DELETE` | `messages.delete` |
| `MESSAGES_SET` | `messages.set` |
| `CONTACTS_UPSERT` | `contacts.upsert` |
| `GROUP_PARTICIPANTS_UPDATE` | `group-participants.update` |
| `CONNECTION_UPDATE` | `connection.update` |
| `CALL` | `call` |

API examples and documentation must present the uppercase values as the preferred configuration format. Subscription matching also accepts the corresponding payload values for backward compatibility.

## Design

### Configuration identifiers

Add a dedicated, package-private `webhookConfigEvent` type and constants for the eight supported configuration values. Keep these definitions separate from `Wook` and the `Wook*` constants, which represent payload values. Package-private names avoid expanding the Go API because the public contract consists of the JSON configuration values.

All handlers receive a subscription map keyed by `webhookConfigEvent` and compare against named configuration constants. This removes uppercase string literals from handlers and prevents payload constants from being used accidentally for subscription checks.

### Normalization

Keep one helper that converts `[]string` from `instance.Webhook.Events` into the typed subscription map. For each value, the helper:

1. Trims surrounding whitespace.
2. Converts letters to uppercase.
3. Replaces dots and hyphens with underscores.
4. Ignores empty results.

This makes `MESSAGES_UPSERT`, `messages.upsert`, and harmless case or whitespace variations resolve to the same configuration identifier. Unsupported values remain inert because no handler checks their normalized key. The implementation does not mutate or persist normalized values.

### Payload identifiers

Keep `Wook` and all `Wook*` constants unchanged. Event builders continue assigning these constants to `WookEvent.Event`, so emitted payloads retain values such as `messages.upsert` and `connection.update`.

### Event flow

The main WhatsMeow event handler continues to normalize the configured event list once and pass the resulting map to individual handlers. The direct `emitConnectionUpdate` path must call the same helper instead of building a raw string map. No handler may construct its own subscription map.

The resulting flow is:

```text
stored configuration strings
        |
        v
webhook configuration normalizer
        |
        v
typed webhookConfigEvent subscription map
        |
        v
handler subscription check
        |
        v
WookEvent with an unchanged payload event identifier
```

## Components

- `lib/whatsmiau/webhook_events.go`: owns configuration event type, constants, and normalization helper.
- `lib/whatsmiau/event_emitter.go`: uses typed configuration constants and routes `emitConnectionUpdate` through the shared normalizer.
- `lib/whatsmiau/call_events.go`: checks the typed `CALL` configuration constant.
- `lib/whatsmiau/models.go`: retains the payload-only `Wook` type and constants without behavioral changes.
- `lib/whatsmiau/webhook_events_test.go`: covers normalization and configuration/payload separation.
- `README.md`: documents both columns of the webhook event contract and marks uppercase values as canonical configuration values.

Existing tests may move from `call_events_test.go` when they describe general webhook subscription behavior rather than call handling.

## Error Handling and Compatibility

The change introduces no new runtime errors. Empty and unknown values remain ignored during matching. Existing uppercase configurations continue to work, and payload-style configurations written by older Manager UI versions remain compatible.

The API continues returning stored event strings as written. Avoiding automatic rewriting prevents unexpected response changes and removes the need for a data migration.

## Testing

Focused unit tests must prove that:

- Canonical values such as `MESSAGES_UPSERT` enable their handlers.
- Compatibility values such as `messages.upsert` enable the same handlers.
- Dots, hyphens, case, and surrounding whitespace normalize consistently.
- Empty and unsupported values do not enable a supported handler.
- `emitConnectionUpdate` honors both `CONNECTION_UPDATE` and `connection.update`.
- Emitted message and connection payloads still contain lowercase `Wook` event values.
- Call subscriptions still accept `CALL` and `call`.

Run the focused `lib/whatsmiau` tests, then run the repository's complete Go test suite to detect type-change regressions across packages.

## Acceptance Criteria

- Every webhook handler checks a named `webhookConfigEvent` constant.
- Every subscription map is created by the shared normalization helper.
- Uppercase configuration values remain the documented canonical format.
- Payload-style configuration aliases remain accepted.
- Emitted payload event values do not change.
- The direct connection update path uses the same matching behavior as the main event handler.
- Focused and complete Go tests pass.
- The replacement pull request targets `develop` and explains that it supersedes #79.
