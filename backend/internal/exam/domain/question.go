// Package domain holds the rules of the Exam context. It imports no framework, database or cache.
package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

// The limits of a Question. The lengths are in characters.
const (
	MaxQuestionTextLength = 2000
	MaxOptionTextLength   = 500
	MinOptions            = 2
	MaxOptions            = 6
)

// OptionInput is an Option as the caller gives it, before the domain checks it.
type OptionInput struct {
	Text    string
	Correct bool
}

// Option is one possible Answer to a Question.
type Option struct {
	ID      uuid.UUID
	Text    string
	Correct bool
}

// Question is a multiple-choice question in the Question bank.
type Question struct {
	ID        uuid.UUID
	Text      string
	Options   []Option // In the order that the Question shows them.
	CreatedAt time.Time
}

// NewQuestion makes a Question at the given time. It makes the IDs, UUIDs of version 7.
func NewQuestion(text string, options []OptionInput, now time.Time) (Question, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return Question{}, fmt.Errorf("The ID of the Question is not available. Cause: %w", err)
	}
	question := Question{ID: id, CreatedAt: now.UTC().Truncate(time.Microsecond)} // Postgres keeps microseconds.
	return question.Change(text, options)
}

// Change gives a copy of the Question with a new text and new Options. The ID and the creation time stay.
// The Options get new IDs.
func (q Question) Change(text string, options []OptionInput) (Question, error) {
	questionText, err := cleanText(text, MaxQuestionTextLength, true, ErrEmptyQuestionText, ErrQuestionTextTooLong)
	if err != nil {
		return Question{}, err
	}
	if len(options) < MinOptions {
		return Question{}, ErrTooFewOptions
	}
	if len(options) > MaxOptions {
		return Question{}, ErrTooManyOptions
	}
	changed := Question{ID: q.ID, Text: questionText, CreatedAt: q.CreatedAt, Options: make([]Option, len(options))}
	correct := 0
	for i, input := range options {
		optionText, err := cleanText(input.Text, MaxOptionTextLength, false, ErrEmptyOptionText, ErrOptionTextTooLong)
		if err != nil {
			return Question{}, err
		}
		optionID, err := uuid.NewV7()
		if err != nil {
			return Question{}, fmt.Errorf("The ID of an Option is not available. Cause: %w", err)
		}
		changed.Options[i] = Option{ID: optionID, Text: optionText, Correct: input.Correct}
		if input.Correct {
			correct++
		}
	}
	if correct != 1 {
		return Question{}, ErrCorrectOptionCount
	}
	return changed, nil
}

// cleanText removes the spaces around a text, and checks its length in characters and its characters.
// Control characters are not allowed, and Postgres cannot keep a NUL character. A text with several lines
// (multiline) can have a new line and a tab.
func cleanText(raw string, maxLength int, multiline bool, errEmpty, errTooLong error) (string, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return "", errEmpty
	}
	if utf8.RuneCountInString(text) > maxLength {
		return "", errTooLong
	}
	for _, r := range text {
		lineBreak := r == '\n' || r == '\r' || r == '\t'
		if unicode.IsControl(r) && (!multiline || !lineBreak) {
			return "", ErrTextNotAllowed
		}
	}
	return text, nil
}
