# ADR 0001: Go as the primary language

- Status: Accepted
- Date: 2026-09-12

## Decision

Use Go 1.26 or newer for SWF server libraries and tools. The baseline was raised
from the initial Go 1.24 assumption after vulnerability scanning found reachable
standard-library issues without fixes on that line. Device implementations
may use platform-appropriate languages while sharing the protocol.
