-- Creating upvotes table
CREATE TABLE IF NOT EXISTS upvotes (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id INTEGER REFERENCES users(id),
    comment_id INTEGER REFERENCES comments(id),
    UNIQUE(user_id, comment_id)
)
