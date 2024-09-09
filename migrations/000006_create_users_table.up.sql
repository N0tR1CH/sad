CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  created_at TIMESTAMPTZ DEFAULT current_timestamp,
  updated_at TIMESTAMPTZ DEFAULT current_timestamp,
  name TEXT NOT NULL,
  email TEXT NOT NULL COLLATE case_insensitive,
  password_hash BYTEA NOT NULL,
  activated BOOLEAN NOT NULL DEFAULT FALSE,
  version INTEGER NOT NULL DEFAULT 1,
  UNIQUE(email)
);
