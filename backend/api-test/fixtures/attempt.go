package fixtures

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rudradcruze/go-react-template/backend/internal/exam/app"
	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
)

var attemptCounter atomic.Uint64

type attemptValues struct {
	name         string
	emailAddress string
	startedAt    time.Time
	submittedAt  *time.Time
}

// AttemptOption changes a value of the Attempt that Attempt builds.
type AttemptOption func(*attemptValues)

// WithCandidateName sets the name of the Candidate.
func WithCandidateName(name string) AttemptOption {
	return func(v *attemptValues) { v.name = name }
}

// WithCandidateEmail sets the email address of the Candidate. Use lowercase, because the database checks it.
func WithCandidateEmail(emailAddress string) AttemptOption {
	return func(v *attemptValues) { v.emailAddress = emailAddress }
}

// WithStartedAt sets the start time. The expiry is the start time plus the Time limit of the Exam.
// Use a time in the past to make an Attempt that is past its expiry.
func WithStartedAt(startedAt time.Time) AttemptOption {
	return func(v *attemptValues) { v.startedAt = startedAt }
}

// WithSubmittedAt sets the submit time.
func WithSubmittedAt(submittedAt time.Time) AttemptOption {
	return func(v *attemptValues) { v.submittedAt = &submittedAt }
}

// Attempt builds an Attempt of the Exam. By default, it started now and it is open.
// It does not check the Exam window, so a test can make an Attempt that the API would not start.
func Attempt(exam domain.Exam, options ...AttemptOption) domain.Attempt {
	n := attemptCounter.Add(1)
	values := attemptValues{
		name:         fmt.Sprintf("Candidate %d", n),
		emailAddress: fmt.Sprintf("candidate%d@example.com", n),
		startedAt:    time.Now(),
	}
	for _, option := range options {
		option(&values)
	}
	startedAt := values.startedAt.UTC().Truncate(time.Microsecond)
	attempt := domain.Attempt{
		ID:            uuid.New(),
		ExamID:        exam.ID,
		CandidateName: values.name,
		EmailAddress:  domain.EmailAddress(values.emailAddress),
		StartedAt:     startedAt,
		ExpiresAt:     startedAt.Add(time.Duration(exam.TimeLimitMinutes) * time.Minute),
	}
	if values.submittedAt != nil {
		submittedAt := values.submittedAt.UTC().Truncate(time.Microsecond)
		attempt.SubmittedAt = &submittedAt
	}
	return attempt
}

// SaveAttempt builds an Attempt of a saved Exam and saves it through the repository.
func SaveAttempt(t *testing.T, attempts app.AttemptRepository, exam domain.Exam, options ...AttemptOption) domain.Attempt {
	t.Helper()
	attempt := Attempt(exam, options...)
	if err := attempts.Create(t.Context(), attempt); err != nil {
		t.Fatalf("The fixture Attempt did not save. Cause: %v", err)
	}
	return attempt
}
