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

var examCounter atomic.Uint64

type examValues struct {
	input          domain.ExamInput
	questionsGiven bool
}

// ExamOption changes a value of the Exam that Exam builds.
type ExamOption func(*examValues)

// WithExamTitle sets the title.
func WithExamTitle(title string) ExamOption {
	return func(v *examValues) { v.input.Title = title }
}

// WithQuestionIDs sets the Questions of the Exam. SaveExam needs the IDs of saved Questions.
func WithQuestionIDs(ids ...uuid.UUID) ExamOption {
	return func(v *examValues) { v.input.QuestionIDs, v.questionsGiven = ids, true }
}

// WithExamWindow sets the open time and the close time.
func WithExamWindow(opensAt, closesAt time.Time) ExamOption {
	return func(v *examValues) { v.input.OpensAt, v.input.ClosesAt = opensAt, closesAt }
}

// WithTimeLimit sets the Time limit in minutes.
func WithTimeLimit(minutes int) ExamOption {
	return func(v *examValues) { v.input.TimeLimitMinutes = minutes }
}

func newExamValues(options []ExamOption) examValues {
	n := examCounter.Add(1)
	now := time.Now()
	values := examValues{input: domain.ExamInput{
		Title:            fmt.Sprintf("Exam %d", n),
		QuestionIDs:      []uuid.UUID{uuid.New()},
		OpensAt:          now.Add(-time.Hour),
		ClosesAt:         now.Add(time.Hour),
		TimeLimitMinutes: 30,
	}}
	for _, option := range options {
		option(&values)
	}
	return values
}

// Exam builds a valid Exam. By default, its Exam window is open, and it has 1 Question with a random ID.
// It panics when an option gives a value that the domain does not accept, because that is a mistake in the test.
func Exam(options ...ExamOption) domain.Exam {
	exam, err := domain.NewExam(newExamValues(options).input, time.Now())
	if err != nil {
		panic(fmt.Sprintf("The fixture Exam is not valid. Cause: %v", err))
	}
	return exam
}

// SaveExam builds an Exam and saves it through the repository. When the options give no Questions,
// it saves one new Question first, because the database needs a Question that exists.
func SaveExam(t *testing.T, exams app.ExamRepository, questions app.QuestionRepository, options ...ExamOption) domain.Exam {
	t.Helper()
	values := newExamValues(options)
	if !values.questionsGiven {
		values.input.QuestionIDs = []uuid.UUID{SaveQuestion(t, questions).ID}
	}
	exam, err := domain.NewExam(values.input, time.Now())
	if err != nil {
		t.Fatalf("The fixture Exam is not valid. Cause: %v", err)
	}
	if err := exams.Create(t.Context(), exam); err != nil {
		t.Fatalf("The fixture Exam did not save. Cause: %v", err)
	}
	return exam
}
