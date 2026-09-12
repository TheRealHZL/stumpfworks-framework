# Core and HTTP Threat Model

- Status: Initial review
- Date: 2026-09-12
- Scope: configuration, logs, HTTP lifecycle, health, PostgreSQL connection,
  migrations, and audit foundation

## Assets

- service availability and bounded resource consumption;
- configuration secrets and PostgreSQL credentials;
- integrity and ordering of database migrations;
- confidentiality and integrity of audit events;
- trustworthy correlation and operational logs.

## Trust boundaries

1. Untrusted network clients cross into the HTTP server.
2. Local operators and service managers provide files and environment values.
3. The application crosses a TLS-authenticated boundary into PostgreSQL.
4. CI obtains Go modules, actions, and container images from external registries.
5. A reverse proxy may terminate public TLS but is not implicitly trusted through
   client-controlled forwarded headers.

## Threats and current controls

| Threat | Current control | Residual work |
|---|---|---|
| Slow headers, oversized headers, or excessive concurrent requests exhaust the service | read/header/write/idle timeouts, one-MiB header cap, concurrency semaphore, bounded queue | load-test limits per application |
| Oversized or ambiguous JSON consumes memory or bypasses validation | per-route size cap, one-value rule, unknown-field rejection, media-type check | add endpoint-specific semantic validators |
| Panic discloses internals or kills the process | recovery middleware, generic Problem Details, correlation-only panic log | define policy for panics after partial streaming output |
| Request URL or query leaks personal data or tokens into logs | log matched route patterns rather than raw paths or queries | review application-added attributes |
| Credentials leak through structured fields | centralized case-insensitive sensitive-key redaction | add static review/lint guidance for secrets embedded in free-form error strings |
| Attacker forges or injects log correlation values | strict request-ID character and length validation; random UUID fallback | document trusted upstream ID propagation if introduced |
| Database traffic silently falls back to plaintext | reject any non-TLS connection or fallback unless explicitly enabled | verify certificate policy in deployment tests |
| Database protocol message causes memory exhaustion | 16-MiB default pgx protocol-message limit | tune only with measured consumer requirements |
| Concurrent instances race migrations | transaction-scoped PostgreSQL advisory lock and concurrent-runner integration test | monitor migration duration |
| Applied migration is silently edited | stored SHA-256 checksum and mismatch failure | sign release artifacts before 1.0 |
| Audit metadata stores obvious credentials | recursive sensitive-key rejection and JSON validation | define per-event schemas and retention policy |
| Shutdown closes dependencies while requests are still running | stop listener, wait up to deadline, force-close connections, then release owned resources | test application-specific long-running handlers |
| Known dependency vulnerability reaches production | pinned Go/tool dependencies, `go mod verify`, Dependabot, `govulncheck` CI | pin GitHub Actions by immutable commit before public release |

## Explicit assumptions

- Operators protect configuration-file and environment access at the OS level.
- Applications do not log secrets inside free-form messages or misleading field
  names; redaction is defense in depth, not a substitute for data classification.
- PostgreSQL credentials have least privilege and production never enables the
  local-development plaintext override.
- Reverse proxies enforce public TLS and HSTS when TLS terminates before SWF.

## Out of scope for this revision

Identity tokens, sessions, RBAC, device pairing, update signatures, tenant
isolation, and application domain logic require separate threat models when the
corresponding modules are designed.

## Required follow-ups

1. Define and test TLS certificate modes for deployment profiles.
2. Add immutable action pins and SBOM generation to release workflows.
3. Review the model with the first Identity consumer.
