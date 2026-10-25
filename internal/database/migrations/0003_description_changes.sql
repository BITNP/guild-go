CREATE TABLE node_description_changes (
    id              BIGSERIAL PRIMARY KEY,
    node_id         BIGINT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    author_subject  TEXT NOT NULL REFERENCES users(subject),
    old_description TEXT NOT NULL,
    new_description TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    CHECK (old_description <> new_description)
);

CREATE INDEX node_description_changes_page_idx
    ON node_description_changes (node_id, created_at DESC, id DESC);
CREATE INDEX comments_timeline_idx ON comments (node_id, created_at DESC, id DESC);
