package exam_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rudradcruze/go-react-template/backend/api-test/fixtures"
	"github.com/rudradcruze/go-react-template/backend/api-test/mocks"
	"github.com/rudradcruze/go-react-template/backend/internal/exam/app"
	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
)

var questionTime = time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)

func TestCreateQuestion_Execute_SavesTheNormalizedQuestionAndReturnsIt(t *testing.T) {
	t.Parallel()
	repository := mocks.NewMockQuestionRepository(t)
	var saved domain.Question
	repository.EXPECT().
		Create(t.Context(), mock.MatchedBy(func(q domain.Question) bool {
			return q.Text == "What?" && len(q.Options) == 2 && q.CreatedAt.Equal(questionTime)
		})).
		Run(func(_ context.Context, q domain.Question) { saved = q }).
		Return(nil).
		Once()

	got, err := app.NewCreateQuestion(repository, func() time.Time { return questionTime }).Execute(t.Context(), app.QuestionInput{
		Text:    " What? ",
		Options: []domain.OptionInput{{Text: " A ", Correct: true}, {Text: "B"}},
	})

	require.NoError(t, err)
	assert.Equal(t, saved, got)
}

func TestCreateQuestion_Execute_DoesNotCallTheRepositoryWhenTheInputBreaksARule(t *testing.T) {
	t.Parallel()
	repository := mocks.NewMockQuestionRepository(t) // It has no expected call, so a call fails the test.

	_, err := app.NewCreateQuestion(repository, time.Now).Execute(t.Context(), app.QuestionInput{Text: "", Options: nil})

	require.ErrorIs(t, err, domain.ErrEmptyQuestionText)
}

func TestGetQuestion_Execute_GivesTheQuestionOrTheNotFoundError(t *testing.T) {
	t.Parallel()
	question := fixtures.Question()
	repository := mocks.NewMockQuestionRepository(t)
	repository.EXPECT().FindByID(t.Context(), question.ID).Return(question, nil).Once()
	missing := uuid.New()
	repository.EXPECT().FindByID(t.Context(), missing).Return(domain.Question{}, domain.ErrQuestionNotFound).Once()
	useCase := app.NewGetQuestion(repository)

	got, err := useCase.Execute(t.Context(), question.ID)
	require.NoError(t, err)
	assert.Equal(t, question, got)

	_, err = useCase.Execute(t.Context(), missing)
	require.ErrorIs(t, err, domain.ErrQuestionNotFound)
}

func TestListQuestions_Execute_GivesTheQuestionsOfTheRepository(t *testing.T) {
	t.Parallel()
	questions := []domain.Question{fixtures.Question(), fixtures.Question()}
	repository := mocks.NewMockQuestionRepository(t)
	repository.EXPECT().List(t.Context()).Return(questions, nil).Once()

	got, err := app.NewListQuestions(repository).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, questions, got)
}

func TestUpdateQuestion_Execute_SavesTheChangedQuestion(t *testing.T) {
	t.Parallel()
	original := fixtures.Question()
	repository := mocks.NewMockQuestionRepository(t)
	repository.EXPECT().FindByID(t.Context(), original.ID).Return(original, nil).Once()
	repository.EXPECT().UsedByAttempt(t.Context(), original.ID).Return(false, nil).Once()
	repository.EXPECT().
		Update(t.Context(), mock.MatchedBy(func(q domain.Question) bool {
			return q.ID == original.ID && q.Text == "New text" && q.CreatedAt.Equal(original.CreatedAt) && len(q.Options) == 2
		})).
		Return(nil).
		Once()

	got, err := app.NewUpdateQuestion(repository).Execute(t.Context(), original.ID, app.QuestionInput{
		Text:    "New text",
		Options: []domain.OptionInput{{Text: "A", Correct: true}, {Text: "B"}},
	})

	require.NoError(t, err)
	assert.Equal(t, "New text", got.Text)
}

func TestUpdateQuestion_Execute_ReturnsTheNotFoundErrorAndDoesNotSave(t *testing.T) {
	t.Parallel()
	repository := mocks.NewMockQuestionRepository(t)
	id := uuid.New()
	repository.EXPECT().FindByID(t.Context(), id).Return(domain.Question{}, domain.ErrQuestionNotFound).Once()

	_, err := app.NewUpdateQuestion(repository).Execute(t.Context(), id, app.QuestionInput{
		Text: "T", Options: []domain.OptionInput{{Text: "A", Correct: true}, {Text: "B"}},
	})

	require.ErrorIs(t, err, domain.ErrQuestionNotFound)
}

func TestUpdateQuestion_Execute_DoesNotSaveWhenTheInputBreaksARule(t *testing.T) {
	t.Parallel()
	original := fixtures.Question()
	repository := mocks.NewMockQuestionRepository(t)
	repository.EXPECT().FindByID(t.Context(), original.ID).Return(original, nil).Once()
	repository.EXPECT().UsedByAttempt(t.Context(), original.ID).Return(false, nil).Once()

	_, err := app.NewUpdateQuestion(repository).Execute(t.Context(), original.ID, app.QuestionInput{Text: "", Options: nil})

	require.ErrorIs(t, err, domain.ErrEmptyQuestionText)
}

func TestUpdateQuestion_Execute_ReturnsLockedWhenAnAttemptUsedTheQuestion(t *testing.T) {
	t.Parallel()
	original := fixtures.Question()
	repository := mocks.NewMockQuestionRepository(t) // It has no expected Update call, so a call fails the test.
	repository.EXPECT().FindByID(t.Context(), original.ID).Return(original, nil).Once()
	repository.EXPECT().UsedByAttempt(t.Context(), original.ID).Return(true, nil).Once()

	_, err := app.NewUpdateQuestion(repository).Execute(t.Context(), original.ID, app.QuestionInput{
		Text: "T", Options: []domain.OptionInput{{Text: "A", Correct: true}, {Text: "B"}},
	})

	require.ErrorIs(t, err, domain.ErrQuestionLocked)
}

func TestDeleteQuestion_Execute_DeletesTheQuestionAndReturnsTheErrorOfTheRepository(t *testing.T) {
	t.Parallel()
	repository := mocks.NewMockQuestionRepository(t)
	id := uuid.New()
	cause := errors.New("The database is down.")
	repository.EXPECT().Delete(t.Context(), id).Return(nil).Once()
	other := uuid.New()
	repository.EXPECT().Delete(t.Context(), other).Return(cause).Once()
	useCase := app.NewDeleteQuestion(repository)

	require.NoError(t, useCase.Execute(t.Context(), id))
	require.ErrorIs(t, useCase.Execute(t.Context(), other), cause)
}
