CREATE TABLE swf_audit_events (
    id text PRIMARY KEY,
    occurred_at timestamptz NOT NULL,
    actor text NOT NULL,
    action text NOT NULL,
    resource text NOT NULL,
    result text NOT NULL CHECK (result IN ('success', 'failure', 'denied')),
    correlation_id text NOT NULL DEFAULT '',
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX swf_audit_events_occurred_at_idx
    ON swf_audit_events (occurred_at DESC);

CREATE INDEX swf_audit_events_resource_idx
    ON swf_audit_events (resource, occurred_at DESC);
