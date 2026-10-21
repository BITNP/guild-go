-- Baseline schema for guild-go.
-- Identity, display names, permissions and positions live in Keycloak; the
-- local database only keeps the known-subject index and guild-owned content.

CREATE TABLE users (
    subject    TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE nodes (
    id           BIGSERIAL PRIMARY KEY,
    parent_id    BIGINT REFERENCES nodes(id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    due_at       TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    tags         TEXT[] NOT NULL DEFAULT '{}',
    sort_order   BIGINT NOT NULL DEFAULT 0,
    created_by   TEXT NOT NULL REFERENCES users(subject),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT nodes_title_length CHECK (char_length(title) BETWEEN 1 AND 200),
    CONSTRAINT nodes_description_length CHECK (char_length(description) <= 20000),
    CONSTRAINT nodes_tags_count CHECK (cardinality(tags) <= 10)
);

CREATE INDEX nodes_parent_order_idx ON nodes (parent_id, sort_order, id);
CREATE INDEX nodes_created_by_idx ON nodes (created_by);
CREATE INDEX nodes_tags_idx ON nodes USING GIN (tags);
CREATE INDEX nodes_open_due_idx ON nodes (due_at) WHERE completed_at IS NULL;

CREATE TABLE node_members (
    node_id    BIGINT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    subject    TEXT NOT NULL REFERENCES users(subject),
    role       TEXT NOT NULL CHECK (role IN ('leader', 'member')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (node_id, subject)
);

CREATE INDEX node_members_subject_idx ON node_members (subject);
CREATE INDEX node_members_node_role_idx ON node_members (node_id, role);

CREATE TABLE comments (
    id             BIGSERIAL PRIMARY KEY,
    node_id        BIGINT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    author_subject TEXT NOT NULL REFERENCES users(subject),
    body           TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT comments_body_length CHECK (char_length(body) <= 5000)
);

CREATE INDEX comments_node_page_idx ON comments (node_id, id DESC);
CREATE INDEX comments_recent_idx ON comments (created_at DESC, id DESC);
CREATE INDEX comments_author_idx ON comments (author_subject);

CREATE TABLE comment_attachments (
    id           BIGSERIAL PRIMARY KEY,
    comment_id   BIGINT NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    object_key   TEXT NOT NULL UNIQUE,
    content_type TEXT NOT NULL,
    size_bytes   BIGINT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT comment_attachments_size_positive CHECK (size_bytes > 0)
);

CREATE INDEX comment_attachments_comment_idx ON comment_attachments (comment_id);

CREATE TABLE comment_reactions (
    comment_id BIGINT NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    subject    TEXT NOT NULL REFERENCES users(subject),
    emoji      TEXT NOT NULL CHECK (emoji IN (
        'thumbs_up', 'thumbs_down', 'laugh', 'hooray',
        'confused', 'heart', 'rocket', 'eyes'
    )),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (comment_id, subject, emoji)
);

CREATE INDEX comment_reactions_comment_idx ON comment_reactions (comment_id);