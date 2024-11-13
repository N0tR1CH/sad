CREATE TYPE token_type AS ENUM ('activation', 'authentication');
CREATE TABLE IF NOT EXISTS tokens (
  hash BYTEA PRIMARY KEY,
  user_id INTEGER NOT NULL,
  expired_at TIMESTAMPTZ,
  token_type TOKEN_TYPE NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
