-- +goose Up
DROP TABLE users;

-- +goose Down
CREATE TABLE users (
    id            uuid        PRIMARY KEY,
    email_address text        NOT NULL,
    display_name  text        NOT NULL,
    created_at    timestamptz NOT NULL,

    CONSTRAINT users_email_address_check CHECK (
        email_address = lower(email_address) AND char_length(email_address) BETWEEN 1 AND 254
    ),
    CONSTRAINT users_display_name_check CHECK (char_length(display_name) BETWEEN 1 AND 100)
);

CREATE UNIQUE INDEX users_email_address_key ON users (email_address);
