# ADR 0006: Device protocol security

- Status: Accepted
- Date: 2026-09-12

## Context

Access nodes can trigger physical actions and include constrained ESP8266 and
ESP32 hardware. Sharing server implementation code is neither practical nor a
security boundary.

## Decision

Share a versioned wire protocol, not a single implementation. Every device has
an immutable ID and individual key. Commands are authenticated and carry an ID,
creation time, expiry, and idempotency key. Expired, duplicate, or unverifiable
commands are rejected. Loss of server contact must preserve a safe state.

## Consequences

Pairing, rotation, replay protection, clock assumptions, and recovery require a
threat model and cross-platform protocol tests before physical actions are
enabled. Firmware updates will be signed and are deferred beyond the first
protocol slice.
