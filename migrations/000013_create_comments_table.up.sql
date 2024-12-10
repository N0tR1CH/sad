CREATE TABLE IF NOT EXISTS comments (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    created_at TIMESTAMPTZ DEFAULT current_timestamp,
    updated_at TIMESTAMPTZ DEFAULT current_timestamp,
    user_id INTEGER references users(id) NOT NULL,
    discussion_id INTEGER references discussions(id) NOT NULL,
    content TEXT NOT NULL
);
