CREATE TABLE IF NOT EXISTS roles (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    created_at TIMESTAMPTZ DEFAULT current_timestamp,
    updated_at TIMESTAMPTZ DEFAULT current_timestamp,
    name TEXT NOT NULL,
    permissions JSONB,
    UNIQUE(name)
);

INSERT INTO roles(name) VALUES('user');
INSERT INTO roles(name) VALUES('admin');
INSERT INTO roles(name) VALUES('guest');

ALTER TABLE IF EXISTS users
ADD COLUMN role_id INTEGER REFERENCES roles(id) ON DELETE cascade;

UPDATE users SET role_id=(
    SELECT id
    FROM roles
    WHERE name='user'
    LIMIT 1
) WHERE role_id IS NULL;

ALTER TABLE IF EXISTS users ALTER COLUMN role_id SET DEFAULT 1;

