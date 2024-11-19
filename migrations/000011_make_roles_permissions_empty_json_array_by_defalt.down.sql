ALTER TABLE roles
ALTER COLUMN permissions
DROP DEFAULT;

UPDATE roles SET permissions=NULL WHERE permissions='[]'::JSONB;
