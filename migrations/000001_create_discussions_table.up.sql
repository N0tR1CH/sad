CREATE TABLE IF NOT EXISTS discussions (
    id bigserial PRIMARY KEY,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    url text NOT NULL,
    title text NOT NULL,
    description text NOT NULL
);