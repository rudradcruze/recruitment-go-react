package domain

import (
	"crypto/rand"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

// The limits of an Exam. The lengths are in characters.
const (
	MaxTitleLength      = 200
	MinQuestions        = 1
	MaxQuestions        = 200
	MinTimeLimitMinutes = 1
	MaxTimeLimitMinutes = 480
	LinkTokenLength     = 22
)

const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// ExamInput is the data of an Exam, as the caller gives it, before the domain checks it.
type ExamInput struct {
	Title            string
	QuestionIDs      []uuid.UUID // In the order that the Exam shows the Questions.
	OpensAt          time.Time
	ClosesAt         time.Time
	TimeLimitMinutes int
}

// Exam is a named set of Questions with an Exam window and a Time limit.
type Exam struct {
	ID               uuid.UUID
	Title            string
	QuestionIDs      []uuid.UUID
	OpensAt          time.Time
	ClosesAt         time.Time
	TimeLimitMinutes int
	LinkToken        string // The token in the Exam link.
	CreatedAt        time.Time
}

// NewExam makes an Exam at the given time. It makes the ID, a UUID of version 7, and the Exam link token.
func NewExam(input ExamInput, now time.Time) (Exam, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return Exam{}, fmt.Errorf("The ID of the Exam is not available. Cause: %w", err)
	}
	token, err := newLinkToken()
	if err != nil {
		return Exam{}, err
	}
	// Postgres keeps microseconds. Truncate here, so the Exam is the same after it is saved.
	exam := Exam{ID: id, LinkToken: token, CreatedAt: now.UTC().Truncate(time.Microsecond)}
	return exam.Change(input)
}

// Change gives a copy of the Exam with new values. The ID, the Exam link token and the creation time stay.
func (e Exam) Change(input ExamInput) (Exam, error) {
	title, err := cleanText(input.Title, MaxTitleLength, false, ErrEmptyTitle, ErrTitleTooLong)
	if err != nil {
		return Exam{}, err
	}
	if len(input.QuestionIDs) < MinQuestions {
		return Exam{}, ErrTooFewQuestions
	}
	if len(input.QuestionIDs) > MaxQuestions {
		return Exam{}, ErrTooManyQuestions
	}
	seen := make(map[uuid.UUID]bool, len(input.QuestionIDs))
	for _, id := range input.QuestionIDs {
		if seen[id] {
			return Exam{}, ErrDuplicateQuestion
		}
		seen[id] = true
	}
	if input.TimeLimitMinutes < MinTimeLimitMinutes || input.TimeLimitMinutes > MaxTimeLimitMinutes {
		return Exam{}, ErrInvalidTimeLimit
	}
	if input.OpensAt.IsZero() || input.ClosesAt.IsZero() || !input.OpensAt.Before(input.ClosesAt) {
		return Exam{}, ErrInvalidWindow
	}
	e.Title = title
	e.QuestionIDs = append([]uuid.UUID(nil), input.QuestionIDs...)
	e.OpensAt = input.OpensAt.UTC().Truncate(time.Microsecond)
	e.ClosesAt = input.ClosesAt.UTC().Truncate(time.Microsecond)
	e.TimeLimitMinutes = input.TimeLimitMinutes
	return e, nil
}

// ChangeAfterAttempts is like Change, for an Exam that has Attempts. Only the close time can change.
// It returns ErrExamLocked when the input changes another value. It checks the rules of Change first.
func (e Exam) ChangeAfterAttempts(input ExamInput) (Exam, error) {
	changed, err := e.Change(input)
	if err != nil {
		return Exam{}, err
	}
	if changed.Title != e.Title || !slices.Equal(changed.QuestionIDs, e.QuestionIDs) ||
		!changed.OpensAt.Equal(e.OpensAt) || changed.TimeLimitMinutes != e.TimeLimitMinutes {
		return Exam{}, ErrExamLocked
	}
	return changed, nil
}

// RegenerateLink gives a copy of the Exam with a new Exam link token. The old token stops working.
func (e Exam) RegenerateLink() (Exam, error) {
	token, err := newLinkToken()
	if err != nil {
		return Exam{}, err
	}
	e.LinkToken = token
	return e, nil
}

// newLinkToken makes a token of 22 random letters and digits, about 131 bits. Nobody can guess it.
func newLinkToken() (string, error) {
	token := make([]byte, 0, LinkTokenLength)
	buffer := make([]byte, 32)
	for len(token) < LinkTokenLength {
		if _, err := rand.Read(buffer); err != nil {
			return "", fmt.Errorf("The Exam link token is not available. Cause: %w", err)
		}
		for _, b := range buffer {
			// Use only the values below 248, so that each of the 62 characters has the same chance.
			if b < 248 && len(token) < LinkTokenLength {
				token = append(token, base62[b%62])
			}
		}
	}
	return string(token), nil
}
