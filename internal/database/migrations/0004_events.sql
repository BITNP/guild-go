-- Append-only domain event log consumed by internal services via cursor polling and SSE.

CREATE TABLE events (
    id            BIGSERIAL PRIMARY KEY,
    kind          TEXT NOT NULL,
    node_id       BIGINT REFERENCES nodes(id) ON DELETE SET NULL,
    actor_subject TEXT NOT NULL REFERENCES users(subject),
    payload       JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX events_created_idx ON events (created_at, id);
CREATE INDEX events_node_idx ON events (node_id, id) WHERE node_id IS NOT NULL;
