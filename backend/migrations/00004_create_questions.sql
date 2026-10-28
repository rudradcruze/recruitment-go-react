-- +goose Up
CREATE TABLE questions (
    id         uuid        PRIMARY KEY,
    text       text        NOT NULL,
    created_at timestamptz NOT NULL,

    -- The checks apply the rules of the domain in the database too. The values of the application already follow them.
    CONSTRAINT questions_text_check CHECK (char_length(text) BETWEEN 1 AND 2000)
);

CREATE INDEX questions_created_at_idx ON questions (created_at DESC, id DESC);

CREATE TABLE options (
    id          uuid    PRIMARY KEY,
    question_id uuid    NOT NULL REFERENCES questions (id) ON DELETE CASCADE,
    position    integer NOT NULL,
    text        text    NOT NULL,
    correct     boolean NOT NULL,

    CONSTRAINT options_text_check CHECK (char_length(text) BETWEEN 1 AND 500),
    CONSTRAINT options_position_check CHECK (position BETWEEN 0 AND 5),
    CONSTRAINT options_question_position_key UNIQUE (question_id, position)
);

-- A Question has exactly one correct Option. The database stops a second one.
CREATE UNIQUE INDEX options_one_correct_key ON options (question_id) WHERE correct;

-- +goose Down
DROP TABLE options;
DROP TABLE questions;
