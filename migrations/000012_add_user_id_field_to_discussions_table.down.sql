ALTER TABLE IF EXISTS discussions
DROP COLUMN user_id;

ALTER TABLE IF EXISTS discussions
DROP CONSTRAINT fk_user;

DROP INDEX idx_discussion_user_id ON discussions(user_id):

