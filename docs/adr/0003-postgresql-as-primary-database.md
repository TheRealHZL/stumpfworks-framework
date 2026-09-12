# ADR 0003: PostgreSQL as the primary database

- Status: Accepted
- Date: 2026-09-12

## Decision

Provide PostgreSQL pooling, migrations, and readiness integration without a
universal ORM. Applications retain their own queries and repositories behind
consumer-owned interfaces.
