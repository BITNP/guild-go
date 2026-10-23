ALTER TABLE comment_attachments
    ALTER COLUMN comment_id DROP NOT NULL,
    ADD COLUMN pending_node_id BIGINT REFERENCES nodes(id) ON DELETE CASCADE,
    ADD COLUMN uploader_subject TEXT REFERENCES users(subject),
    ADD COLUMN position INTEGER NOT NULL DEFAULT 0,
    ADD CONSTRAINT attachment_destination CHECK (
        (comment_id IS NOT NULL AND pending_node_id IS NULL AND uploader_subject IS NULL)
        OR (comment_id IS NULL AND pending_node_id IS NOT NULL AND uploader_subject IS NOT NULL)
    );

CREATE INDEX idx_pending_comment_images_created_at
    ON comment_attachments(created_at) WHERE comment_id IS NULL;
