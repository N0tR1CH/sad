ALTER TABLE IF EXISTS roles
ALTER COLUMN permissions
SET DEFAULT '[]'::JSONB;

UPDATE roles SET permissions='[]'::JSONB WHERE permissions IS NULL;
