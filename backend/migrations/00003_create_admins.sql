-- +goose Up
CREATE TABLE admins (
    id            uuid        PRIMARY KEY,
    email_address text        NOT NULL,
    password_hash text        NOT NULL,
    created_at    timestamptz NOT NULL,

    -- The check applies the rule of the domain in the database too. The values of the application already follow it.
    CONSTRAINT admins_email_address_check CHECK (
        email_address = lower(email_address) AND char_length(email_address) BETWEEN 1 AND 254
    )
);

CREATE UNIQUE INDEX admins_email_address_key ON admins (email_address);

-- +goose Down
DROP TABLE admins;
