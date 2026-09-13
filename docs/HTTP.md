# HTTP Conventions

## Middleware order

`core/app` applies HTTP middleware in registration order: the first registered
middleware is outermost. The minimal app installs the standard chain first and
the concurrency limiter second, so rejected requests still receive correlation,
security, logging, and Problem Details behaviour.

## Request identity and logging

An incoming `X-Request-ID` is accepted only when it is 1–128 ASCII letters,
digits, hyphens, or underscores. Otherwise SWF creates a random UUID v4. Request
logs contain the matched Go route pattern, never the raw path or query string.

## Security headers

The standard chain emits content-type sniffing, framing, referrer, content
security, and browser capability restrictions. HSTS is emitted only for direct
TLS requests. When TLS terminates at a trusted reverse proxy such as Zoraxy, HSTS
must be configured at that proxy; SWF does not trust arbitrary forwarded headers.

## Errors

Framework-generated API errors use `application/problem+json`. Public titles and
codes are separate from wrapped internal causes. Problem responses and health
responses are marked `Cache-Control: no-store`.

## Input limits

The server bounds header size and all relevant timeouts. JSON endpoints should
use `web/validation.DecodeJSON` with a route-specific maximum body size. The
minimal app also limits simultaneous requests and applies a bounded queue wait.

## CORS

CORS is disabled unless an application explicitly installs `middleware.CORS`.
The middleware validates exact HTTP(S) origins, methods, and headers. Credentialed
requests cannot use a wildcard origin. CORS does not replace CSRF protection.

## Metrics

`web/metrics.New` provides a Prometheus-compatible request counter and duration
histogram. Install `registry.Middleware` outermost to observe the final status
after panic recovery and overload handling. Register `registry.Handler()` only
on a separate, protected management listener; the minimal app intentionally
does not publish `/metrics`. Labels use the registered route pattern (or
`unmatched`), a bounded method set, and status. Raw paths, queries, IDs, and
headers are never used as labels. Each application owns its own registry.
