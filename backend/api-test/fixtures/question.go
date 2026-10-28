package fixtures

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rudradcruze/go-react-template/backend/internal/exam/app"
	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
)

var questionCounter atomic.Uint64

type questionValues struct {
	text      string
	options   []domain.OptionInput
	createdAt time.Time
}

// QuestionOption changes a value of the Question that Question builds.
type QuestionOption func(*questionValues)

// WithQuestionText sets the text of the Question.
func WithQuestionText(text string) QuestionOption {
	return func(v *questionValues) { v.text = text }
}

// WithOptions sets the Options of the Question.
func WithOptions(options ...domain.OptionInput) QuestionOption {
	return func(v *questionValues) { v.options = options }
}

// WithQuestionCreatedAt sets the time that the Question was created.
func WithQuestionCreatedAt(createdAt time.Time) QuestionOption {
	return func(v *questionValues) { v.createdAt = createdAt }
}

// Question builds a valid Question with 4 Options. The first Option is correct.
// It panics when an option gives a value that the domain does not accept, because that is a mistake in the test.
func Question(options ...QuestionOption) domain.Question {
	n := questionCounter.Add(1)
	values := questionValues{
		text: fmt.Sprintf("Question %d?", n),
		options: []domain.OptionInput{
			{Text: fmt.Sprintf("Right answer %d", n), Correct: true},
			{Text: "Wrong answer 1"}, {Text: "Wrong answer 2"}, {Text: "Wrong answer 3"},
		},
		createdAt: time.Now(),
	}
	for _, option := range options {
		option(&values)
	}
	question, err := domain.NewQuestion(values.text, values.options, values.createdAt)
	if err != nil {
		panic(fmt.Sprintf("The fixture Question is not valid. Cause: %v", err))
	}
	return question
}

// SaveQuestion builds a Question and saves it through the repository.
func SaveQuestion(t *testing.T, questions app.QuestionRepository, options ...QuestionOption) domain.Question {
	t.Helper()
	question := Question(options...)
	if err := questions.Create(t.Context(), question); err != nil {
		t.Fatalf("The fixture Question did not save. Cause: %v", err)
	}
	return question
}
