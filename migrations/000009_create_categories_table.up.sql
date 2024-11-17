CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    created_at TIMESTAMPTZ DEFAULT current_timestamp,
    updated_at TIMESTAMPTZ DEFAULT current_timestamp,
    name VARCHAR(50) NOT NULL,
    UNIQUE(name)
);

INSERT INTO categories(name) VALUES ('random');

ALTER TABLE IF EXISTS discussions
ADD COLUMN category_id INTEGER REFERENCES categories(id) ON DELETE cascade;

UPDATE discussions SET category_id=(
    SELECT id
    FROM categories
    WHERE name='random'
    LIMIT 1
) WHERE category_id IS NULL;
