CREATE TABLE IF NOT EXISTS discussions (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    created_at TIMESTAMPTZ DEFAULT current_timestamp,
    updated_at TIMESTAMPTZ DEFAULT current_timestamp,
    url TEXT NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL
);
