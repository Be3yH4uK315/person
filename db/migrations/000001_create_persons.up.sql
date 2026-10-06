CREATE TABLE persons (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL CHECK (
        btrim(email) <> ''
        AND char_length(btrim(email)) <= 254
    ),
    phone TEXT NOT NULL CHECK (
        btrim(phone) <> ''
        AND char_length(btrim(phone)) <= 32
        AND btrim(phone) ~ '^\+?[0-9]{7,15}$'
    ),
    first_name TEXT NOT NULL CHECK (
        btrim(first_name) <> ''
        AND char_length(btrim(first_name)) <= 100
    ),
    last_name TEXT NOT NULL CHECK (
        btrim(last_name) <> ''
        AND char_length(btrim(last_name)) <= 100
    ),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    version BIGINT NOT NULL CHECK (version > 0)
);

CREATE UNIQUE INDEX uq_persons_email_normalized
    ON persons (lower(btrim(email)));

CREATE INDEX idx_persons_created_at_id ON persons (created_at DESC, id ASC);