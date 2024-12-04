ALTER TABLE IF EXISTS discussions
ADD COLUMN user_id INTEGER;

ALTER TABLE IF EXISTS discussions
ADD CONSTRAINT fk_user
    FOREIGN KEY(user_id)
    REFERENCES users(id)
    ON DELETE SET NULL;

CREATE INDEX idx_discussions_user_id
ON discussions(user_id);

