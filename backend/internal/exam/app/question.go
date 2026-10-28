package app

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
)

// QuestionInput is the data of a Question, as the caller gives it.
type QuestionInput struct {
	Text    string
	Options []domain.OptionInput
}

// CreateQuestion is the use case that adds a Question to the Question bank.
type CreateQuestion struct {
	questions QuestionRepository
	now       func() time.Time
}

// NewCreateQuestion builds the use case. now gives the time of the creation.
func NewCreateQuestion(questions QuestionRepository, now func() time.Time) *CreateQuestion {
	return &CreateQuestion{questions: questions, now: now}
}

// Execute creates a Question. It returns an error of the domain when the input breaks a rule.
func (c *CreateQuestion) Execute(ctx context.Context, input QuestionInput) (domain.Question, error) {
	question, err := domain.NewQuestion(input.Text, input.Options, c.now())
	if err != nil {
		return domain.Question{}, err
	}
	if err := c.questions.Create(ctx, question); err != nil {
		return domain.Question{}, fmt.Errorf("The application did not save the Question. Cause: %w", err)
	}
	return question, nil
}

// GetQuestion is the use case that reads one Question.
type GetQuestion struct {
	questions QuestionRepository
}

// NewGetQuestion builds the use case.
func NewGetQuestion(questions QuestionRepository) *GetQuestion {
	return &GetQuestion{questions: questions}
}

// Execute gives the Question with this ID. It returns domain.ErrQuestionNotFound when there is none.
func (g *GetQuestion) Execute(ctx context.Context, id uuid.UUID) (domain.Question, error) {
	return g.questions.FindByID(ctx, id)
}

// ListQuestions is the use case that reads all Questions.
type ListQuestions struct {
	questions QuestionRepository
}

// NewListQuestions builds the use case.
func NewListQuestions(questions QuestionRepository) *ListQuestions {
	return &ListQuestions{questions: questions}
}

// Execute gives all Questions, newest first.
func (l *ListQuestions) Execute(ctx context.Context) ([]domain.Question, error) {
	return l.questions.List(ctx)
}

// UpdateQuestion is the use case that replaces the text and the Options of a Question.
type UpdateQuestion struct {
	questions QuestionRepository
}

// NewUpdateQuestion builds the use case.
func NewUpdateQuestion(questions QuestionRepository) *UpdateQuestion {
	return &UpdateQuestion{questions: questions}
}

// Execute changes the Question. It returns domain.ErrQuestionNotFound when there is none,
// domain.ErrQuestionLocked when an Attempt used the Question, and an error of the domain when the input breaks a rule.
func (u *UpdateQuestion) Execute(ctx context.Context, id uuid.UUID, input QuestionInput) (domain.Question, error) {
	question, err := u.questions.FindByID(ctx, id)
	if err != nil {
		return domain.Question{}, err
	}
	used, err := u.questions.UsedByAttempt(ctx, id)
	if err != nil {
		return domain.Question{}, fmt.Errorf("The application did not read the Attempts of the Question. Cause: %w", err)
	}
	if used {
		return domain.Question{}, domain.ErrQuestionLocked
	}
	changed, err := question.Change(input.Text, input.Options)
	if err != nil {
		return domain.Question{}, err
	}
	if err := u.questions.Update(ctx, changed); err != nil {
		return domain.Question{}, err
	}
	return changed, nil
}

// DeleteQuestion is the use case that removes a Question.
type DeleteQuestion struct {
	questions QuestionRepository
}

// NewDeleteQuestion builds the use case.
func NewDeleteQuestion(questions QuestionRepository) *DeleteQuestion {
	return &DeleteQuestion{questions: questions}
}

// Execute deletes the Question. It returns domain.ErrQuestionNotFound when there is none.
func (d *DeleteQuestion) Execute(ctx context.Context, id uuid.UUID) error {
	return d.questions.Delete(ctx, id)
}
