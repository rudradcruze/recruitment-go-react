-- +goose Up
CREATE TABLE attempts (
    id             uuid        PRIMARY KEY,
    exam_id        uuid        NOT NULL REFERENCES exams (id),
    candidate_name text        NOT NULL,
    email_address  text        NOT NULL,
    started_at     timestamptz NOT NULL,
    expires_at     timestamptz NOT NULL,
    submitted_at   timestamptz,

    -- The checks apply the rules of the domain in the database too. The values of the application already follow them.
    CONSTRAINT attempts_candidate_name_check CHECK (char_length(candidate_name) BETWEEN 1 AND 100),
    CONSTRAINT attempts_email_address_check CHECK (
        email_address = lower(email_address) AND char_length(email_address) BETWEEN 1 AND 254
    ),
    CONSTRAINT attempts_expiry_check CHECK (expires_at > started_at)
);

-- One Attempt for each email address for each Exam.
CREATE UNIQUE INDEX attempts_exam_email_key ON attempts (exam_id, email_address);

-- An Answer refers to an Option of its Question. A Candidate has one Answer for each Question of an Attempt.
CREATE UNIQUE INDEX options_question_option_key ON options (question_id, id);

CREATE TABLE answers (
    attempt_id  uuid NOT NULL REFERENCES attempts (id) ON DELETE CASCADE,
    question_id uuid NOT NULL,
    option_id   uuid NOT NULL,
    saved_at    timestamptz NOT NULL DEFAULT now(),

    PRIMARY KEY (attempt_id, question_id),
    FOREIGN KEY (question_id, option_id) REFERENCES options (question_id, id)
);

-- +goose Down
DROP TABLE answers;
DROP INDEX options_question_option_key;
DROP TABLE attempts;
