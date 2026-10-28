package exam_test

import (
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

func TestCreateExam_Execute_SavesTheExamAndReturnsIt(t *testing.T) {
	t.Parallel()
	repository := mocks.NewMockExamRepository(t)
	input := examInput(2)
	repository.EXPECT().
		Create(t.Context(), mock.MatchedBy(func(e domain.Exam) bool {
			return e.Title == "Go basics" && len(e.QuestionIDs) == 2 && e.CreatedAt.Equal(questionTime) && e.LinkToken != ""
		})).
		Return(nil).
		Once()

	got, err := app.NewCreateExam(repository, func() time.Time { return questionTime }).Execute(t.Context(), input)

	require.NoError(t, err)
	assert.Equal(t, "Go basics", got.Title)
}

func TestCreateExam_Execute_ReturnsTheErrorOfTheRepositoryForAnUnknownQuestion(t *testing.T) {
	t.Parallel()
	repository := mocks.NewMockExamRepository(t)
	repository.EXPECT().Create(t.Context(), mock.Anything).Return(domain.ErrUnknownQuestion).Once()

	_, err := app.NewCreateExam(repository, time.Now).Execute(t.Context(), examInput(1))

	require.ErrorIs(t, err, domain.ErrUnknownQuestion)
}

func TestCreateExam_Execute_DoesNotCallTheRepositoryWhenTheInputBreaksARule(t *testing.T) {
	t.Parallel()
	repository := mocks.NewMockExamRepository(t) // It has no expected call, so a call fails the test.

	_, err := app.NewCreateExam(repository, time.Now).Execute(t.Context(), examInput(0))

	require.ErrorIs(t, err, domain.ErrTooFewQuestions)
}

func TestGetExamAndListExams_Execute_GiveWhatTheRepositoryGives(t *testing.T) {
	t.Parallel()
	exam := fixtures.Exam()
	repository := mocks.NewMockExamRepository(t)
	repository.EXPECT().FindByID(t.Context(), exam.ID).Return(exam, nil).Once()
	missing := uuid.New()
	repository.EXPECT().FindByID(t.Context(), missing).Return(domain.Exam{}, domain.ErrExamNotFound).Once()
	repository.EXPECT().List(t.Context()).Return([]domain.Exam{exam}, nil).Once()

	got, err := app.NewGetExam(repository).Execute(t.Context(), exam.ID)
	require.NoError(t, err)
	assert.Equal(t, exam, got)
	_, err = app.NewGetExam(repository).Execute(t.Context(), missing)
	require.ErrorIs(t, err, domain.ErrExamNotFound)
	list, err := app.NewListExams(repository).Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []domain.Exam{exam}, list)
}

func TestUpdateExam_Execute_SavesTheChangedExam(t *testing.T) {
	t.Parallel()
	original := fixtures.Exam()
	repository := mocks.NewMockExamRepository(t)
	attempts := mocks.NewMockAttemptRepository(t)
	repository.EXPECT().FindByID(t.Context(), original.ID).Return(original, nil).Once()
	attempts.EXPECT().ExistsForExam(t.Context(), original.ID).Return(false, nil).Once()
	repository.EXPECT().
		Update(t.Context(), mock.MatchedBy(func(e domain.Exam) bool {
			return e.ID == original.ID && e.Title == "New title" && e.LinkToken == original.LinkToken
		})).
		Return(nil).
		Once()
	input := examInput(1)
	input.Title = "New title"

	got, err := app.NewUpdateExam(repository, attempts).Execute(t.Context(), original.ID, input)

	require.NoError(t, err)
	assert.Equal(t, "New title", got.Title)
}

func TestUpdateExam_Execute_DoesNotSaveForAnUnknownExamOrAnInputThatBreaksARule(t *testing.T) {
	t.Parallel()
	original := fixtures.Exam()
	repository := mocks.NewMockExamRepository(t)
	attempts := mocks.NewMockAttemptRepository(t)
	repository.EXPECT().FindByID(t.Context(), original.ID).Return(original, nil).Once()
	attempts.EXPECT().ExistsForExam(t.Context(), original.ID).Return(false, nil).Once()
	missing := uuid.New()
	repository.EXPECT().FindByID(t.Context(), missing).Return(domain.Exam{}, domain.ErrExamNotFound).Once()
	useCase := app.NewUpdateExam(repository, attempts)

	_, err := useCase.Execute(t.Context(), original.ID, examInput(0))
	require.ErrorIs(t, err, domain.ErrTooFewQuestions)
	_, err = useCase.Execute(t.Context(), missing, examInput(1))
	require.ErrorIs(t, err, domain.ErrExamNotFound)
}

func TestRegenerateExamLink_Execute_SavesTheExamWithANewToken(t *testing.T) {
	t.Parallel()
	original := fixtures.Exam()
	repository := mocks.NewMockExamRepository(t)
	repository.EXPECT().FindByID(t.Context(), original.ID).Return(original, nil).Once()
	repository.EXPECT().
		Update(t.Context(), mock.MatchedBy(func(e domain.Exam) bool {
			return e.ID == original.ID && e.LinkToken != original.LinkToken
		})).
		Return(nil).
		Once()

	got, err := app.NewRegenerateExamLink(repository).Execute(t.Context(), original.ID)

	require.NoError(t, err)
	assert.NotEqual(t, original.LinkToken, got.LinkToken)
}

func TestDeleteExam_Execute_DeletesTheExam(t *testing.T) {
	t.Parallel()
	repository := mocks.NewMockExamRepository(t)
	id := uuid.New()
	repository.EXPECT().Delete(t.Context(), id).Return(nil).Once()

	require.NoError(t, app.NewDeleteExam(repository).Execute(t.Context(), id))
}

func TestUpdateExam_Execute_AfterTheFirstAttemptChangesOnlyTheCloseTime(t *testing.T) {
	t.Parallel()
	original := fixtures.Exam()
	repository := mocks.NewMockExamRepository(t)
	attempts := mocks.NewMockAttemptRepository(t)
	repository.EXPECT().FindByID(t.Context(), original.ID).Return(original, nil).Twice()
	attempts.EXPECT().ExistsForExam(t.Context(), original.ID).Return(true, nil).Twice()
	repository.EXPECT().
		Update(t.Context(), mock.MatchedBy(func(e domain.Exam) bool { return e.ClosesAt.After(original.ClosesAt) })).
		Return(nil).
		Once()
	useCase := app.NewUpdateExam(repository, attempts)
	sameExceptClose := domain.ExamInput{
		Title: original.Title, QuestionIDs: original.QuestionIDs, OpensAt: original.OpensAt,
		ClosesAt: original.ClosesAt.Add(time.Hour), TimeLimitMinutes: original.TimeLimitMinutes,
	}

	got, err := useCase.Execute(t.Context(), original.ID, sameExceptClose)
	require.NoError(t, err)
	assert.Equal(t, original.ClosesAt.Add(time.Hour).UTC().Truncate(time.Microsecond), got.ClosesAt)

	changedTitle := sameExceptClose
	changedTitle.Title = "Another title"
	_, err = useCase.Execute(t.Context(), original.ID, changedTitle)
	require.ErrorIs(t, err, domain.ErrExamLocked)
}
