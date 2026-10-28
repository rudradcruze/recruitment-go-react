package domain

import (
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

// MaxCandidateNameLength is the maximum length of the name of a Candidate, in characters.
const MaxCandidateNameLength = 100

// AttemptStatus tells what an Attempt is now. The values are the same as in the API spec.
type AttemptStatus string

// The statuses of an Attempt.
const (
	StatusOpen      AttemptStatus = "open"
	StatusSubmitted AttemptStatus = "submitted"
	StatusTimedOut  AttemptStatus = "timed_out"
)

// Candidate is the name and the email address that a person gives to start an Attempt.
type Candidate struct {
	Name         string
	EmailAddress EmailAddress
}

// NewCandidate checks and normalizes the name and the email address.
func NewCandidate(name, emailAddress string) (Candidate, error) {
	cleanName, err := cleanText(name, MaxCandidateNameLength, false, ErrEmptyCandidateName, ErrCandidateNameTooLong)
	if err != nil {
		return Candidate{}, err
	}
	email, err := NewEmailAddress(emailAddress)
	if err != nil {
		return Candidate{}, err
	}
	return Candidate{Name: cleanName, EmailAddress: email}, nil
}

// Attempt is one try of one Candidate at one Exam.
type Attempt struct {
	ID            uuid.UUID // A random UUID of version 4. It is the secret of the Candidate.
	ExamID        uuid.UUID
	CandidateName string
	EmailAddress  EmailAddress
	StartedAt     time.Time
	ExpiresAt     time.Time  // StartedAt plus the Time limit.
	SubmittedAt   *time.Time // Nil until the Candidate submits.
}

// Answer is the Option that a Candidate selected for one Question of an Attempt.
type Answer struct {
	QuestionID uuid.UUID
	OptionID   uuid.UUID
}

// NewAttempt starts an Attempt of the Candidate at the given time. It returns ErrExamNotOpen before the open time
// of the Exam, and ErrExamClosed at or after the close time. The expiry is not limited by the close time:
// an Attempt that starts before the close time can use its full Time limit.
func NewAttempt(exam Exam, candidate Candidate, now time.Time) (Attempt, error) {
	if now.Before(exam.OpensAt) {
		return Attempt{}, ErrExamNotOpen
	}
	if !now.Before(exam.ClosesAt) {
		return Attempt{}, ErrExamClosed
	}
	// The ID is a secret, so it is a random UUID. A UUID of version 7 has a time and is easier to guess.
	id, err := uuid.NewRandom()
	if err != nil {
		return Attempt{}, fmt.Errorf("The ID of the Attempt is not available. Cause: %w", err)
	}
	startedAt := now.UTC().Truncate(time.Microsecond) // Postgres keeps microseconds.
	return Attempt{
		ID:            id,
		ExamID:        exam.ID,
		CandidateName: candidate.Name,
		EmailAddress:  candidate.EmailAddress,
		StartedAt:     startedAt,
		ExpiresAt:     startedAt.Add(time.Duration(exam.TimeLimitMinutes) * time.Minute),
	}, nil
}

// Status tells what the Attempt is at the given time. It is submitted when the Candidate submitted it
// at or before the expiry. It is timed out when the Candidate submitted it after the expiry, or did not submit it
// and the expiry is past. No job sets the status: this method works it out when the API reads the Attempt.
func (a Attempt) Status(now time.Time) AttemptStatus {
	switch {
	case a.SubmittedAt != nil && !a.SubmittedAt.After(a.ExpiresAt):
		return StatusSubmitted
	case a.SubmittedAt != nil, now.After(a.ExpiresAt):
		return StatusTimedOut
	default:
		return StatusOpen
	}
}

// AcceptsAnswers returns ErrAttemptClosed when the Candidate submitted the Attempt, or when the expiry is past.
func (a Attempt) AcceptsAnswers(now time.Time) error {
	if a.SubmittedAt != nil || now.After(a.ExpiresAt) {
		return ErrAttemptClosed
	}
	return nil
}

// Submit gives a copy of the Attempt with the submit time. It works after the expiry too, because the Answers
// cannot change after the expiry. It returns ErrAttemptAlreadySubmitted when the Attempt has a submit time.
func (a Attempt) Submit(now time.Time) (Attempt, error) {
	if a.SubmittedAt != nil {
		return Attempt{}, ErrAttemptAlreadySubmitted
	}
	submittedAt := now.UTC().Truncate(time.Microsecond) // Postgres keeps microseconds.
	a.SubmittedAt = &submittedAt
	return a, nil
}

// ValidateAnswer returns ErrInvalidAnswer when the Question is not a Question of the Exam,
// or when the Option is not an Option of the Question. questions are the Questions of the Exam.
func ValidateAnswer(exam Exam, questions []Question, answer Answer) error {
	if !slices.Contains(exam.QuestionIDs, answer.QuestionID) {
		return ErrInvalidAnswer
	}
	for _, question := range questions {
		if question.ID != answer.QuestionID {
			continue
		}
		if slices.ContainsFunc(question.Options, func(option Option) bool { return option.ID == answer.OptionID }) {
			return nil
		}
	}
	return ErrInvalidAnswer
}

// Score is the result of an Attempt: the number of correct Answers out of the number of Questions.
type Score struct {
	Correct int
	Total   int
}

// ScoreOf counts 1 point for each Question that the Candidate answered with the correct Option.
// A wrong Answer and a Question with no Answer count 0 points.
func ScoreOf(questions []Question, answers []Answer) Score {
	chosen := make(map[uuid.UUID]uuid.UUID, len(answers))
	for _, answer := range answers {
		chosen[answer.QuestionID] = answer.OptionID
	}
	score := Score{Total: len(questions)}
	for _, question := range questions {
		optionID, answered := chosen[question.ID]
		if !answered {
			continue
		}
		for _, option := range question.Options {
			if option.ID == optionID && option.Correct {
				score.Correct++
			}
		}
	}
	return score
}
