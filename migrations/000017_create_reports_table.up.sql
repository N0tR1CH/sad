CREATE TABLE IF NOT EXISTS reports (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    created_at TIMESTAMPTZ DEFAULT current_timestamp,
    updated_at TIMESTAMPTZ DEFAULT current_timestamp,
    user_id INTEGER references users(id) NOT NULL,
    reported_user_id INTEGER references users(id) NOT NULL,
    discussion_id INTEGER references discussions(id),
    comment_id INTEGER references comments(id),
    reason TEXT NOT NULL,
    UNIQUE (user_id, reported_user_id, comment_id),
    UNIQUE (user_id, reported_user_id, discussion_id)
)
