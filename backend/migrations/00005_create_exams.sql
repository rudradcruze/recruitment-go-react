-- +goose Up
CREATE TABLE exams (
    id                 uuid        PRIMARY KEY,
    title              text        NOT NULL,
    opens_at           timestamptz NOT NULL,
    closes_at          timestamptz NOT NULL,
    time_limit_minutes integer     NOT NULL,
    link_token         text        NOT NULL,
    created_at         timestamptz NOT NULL,

    -- The checks apply the rules of the domain in the database too. The values of the application already follow them.
    CONSTRAINT exams_title_check CHECK (char_length(title) BETWEEN 1 AND 200),
    CONSTRAINT exams_time_limit_check CHECK (time_limit_minutes BETWEEN 1 AND 480),
    CONSTRAINT exams_window_check CHECK (opens_at < closes_at),
    CONSTRAINT exams_link_token_check CHECK (char_length(link_token) = 22)
);

CREATE UNIQUE INDEX exams_link_token_key ON exams (link_token);
CREATE INDEX exams_created_at_idx ON exams (created_at DESC, id DESC);

-- A Question that an Exam uses cannot be deleted. The default action NO ACTION fails with the error code 23503,
-- and GORM changes that code to ErrForeignKeyViolated. RESTRICT would fail with the code 23001, which GORM does not change.
-- Deleting an Exam removes its rows here.
CREATE TABLE exam_questions (
    exam_id     uuid    NOT NULL REFERENCES exams (id) ON DELETE CASCADE,
    question_id uuid    NOT NULL REFERENCES questions (id),
    position    integer NOT NULL,

    PRIMARY KEY (exam_id, position),
    CONSTRAINT exam_questions_question_key UNIQUE (exam_id, question_id),
    CONSTRAINT exam_questions_position_check CHECK (position BETWEEN 0 AND 199)
);

CREATE INDEX exam_questions_question_id_idx ON exam_questions (question_id);

-- +goose Down
DROP TABLE exam_questions;
DROP TABLE exams;
