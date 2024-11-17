DROP TABLE IF EXISTS categories;
ALTER TABLE IF EXISTS discussions (
    DROP COLUMN category_id
);

