package exam_test

import (
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

type attemptMocks struct {
	exams     *mocks.MockExamRepository
	questions *mocks.MockQuestionRepository
	attempts  *mocks.MockAttemptRepository
}

func newAttemptMocks(t *testing.T) attemptMocks {
	t.Helper()
	return attemptMocks{
		exams:     mocks.NewMockExamRepository(t),
		questions: mocks.NewMockQuestionRepository(t),
		attempts:  mocks.NewMockAttemptRepository(t),
	}
}

func (m attemptMocks) startAttempt() *app.StartAttempt {
	return app.NewStartAttempt(m.exams, m.questions, m.attempts, func() time.Time { return attemptNow })
}

// expectView sets the calls that build the view of an open Attempt.
func (m attemptMocks) expectView(t *testing.T, exam domain.Exam, questions []domain.Question, answers []domain.Answer) {
	t.Helper()
	m.exams.EXPECT().FindByID(t.Context(), exam.ID).Return(exam, nil).Once()
	m.questions.EXPECT().FindByIDs(t.Context(), exam.QuestionIDs).Return(questions, nil).Once()
	m.attempts.EXPECT().Answers(t.Context(), mock.Anything).Return(answers, nil).Once()
}

var startInput = app.StartAttemptInput{Token: "token", Name: " Ann ", EmailAddress: " Ann@Example.COM "}

func TestStartAttempt_Execute_StartsANewAttemptAndGivesItsQuestions(t *testing.T) {
	t.Parallel()
	m := newAttemptMocks(t)
	question := fixtures.Question()
	exam := fixtures.Exam(fixtures.WithQuestionIDs(question.ID), fixtures.WithExamWindow(attemptNow.Add(-time.Hour), attemptNow.Add(time.Hour)), fixtures.WithTimeLimit(30))
	m.exams.EXPECT().FindByToken(t.Context(), "token").Return(exam, nil).Once()
	m.attempts.EXPECT().FindByExamAndEmail(t.Context(), exam.ID, domain.EmailAddress("ann@example.com")).Return(domain.Attempt{}, domain.ErrAttemptNotFound).Once()
	m.attempts.EXPECT().
		Create(t.Context(), mock.MatchedBy(func(a domain.Attempt) bool {
			return a.ExamID == exam.ID && a.CandidateName == "Ann" && a.EmailAddress == "ann@example.com" &&
				a.ExpiresAt.Sub(a.StartedAt) == 30*time.Minute && a.SubmittedAt == nil
		})).
		Return(nil).
		Once()
	m.expectView(t, exam, []domain.Question{question}, nil)

	view, created, err := m.startAttempt().Execute(t.Context(), startInput)

	require.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, domain.StatusOpen, view.Status)
	assert.Equal(t, exam, view.Exam)
	assert.Equal(t, []domain.Question{question}, view.Questions)
	assert.Equal(t, "ann@example.com", string(view.Attempt.EmailAddress))
}

func TestStartAttempt_Execute_GivesTheOpenAttemptOfTheEmailAddressWithoutANewOne(t *testing.T) {
	t.Parallel()
	m := newAttemptMocks(t) // It has no expected Create call, so a call fails the test.
	exam := openExam()
	open := fixtures.Attempt(exam, fixtures.WithCandidateEmail("ann@example.com"), fixtures.WithStartedAt(attemptNow.Add(-5*time.Minute)))
	answers := []domain.Answer{{QuestionID: exam.QuestionIDs[0], OptionID: uuid.New()}}
	m.exams.EXPECT().FindByToken(t.Context(), "token").Return(exam, nil).Once()
	m.attempts.EXPECT().FindByExamAndEmail(t.Context(), exam.ID, domain.EmailAddress("ann@example.com")).Return(open, nil).Once()
	m.expectView(t, exam, nil, answers)

	view, created, err := m.startAttempt().Execute(t.Context(), startInput)

	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, open, view.Attempt)
	assert.Equal(t, answers, view.Answers)
}

func TestStartAttempt_Execute_GivesTheOpenAttemptAlsoAfterTheCloseTimeOfTheExam(t *testing.T) {
	t.Parallel()
	m := newAttemptMocks(t)
	exam := fixtures.Exam(fixtures.WithExamWindow(attemptNow.Add(-2*time.Hour), attemptNow.Add(-time.Minute)), fixtures.WithTimeLimit(30))
	open := fixtures.Attempt(exam, fixtures.WithCandidateEmail("ann@example.com"), fixtures.WithStartedAt(attemptNow.Add(-10*time.Minute)))
	m.exams.EXPECT().FindByToken(t.Context(), "token").Return(exam, nil).Once()
	m.attempts.EXPECT().FindByExamAndEmail(t.Context(), exam.ID, mock.Anything).Return(open, nil).Once()
	m.expectView(t, exam, nil, nil)

	view, created, err := m.startAttempt().Execute(t.Context(), startInput)

	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, domain.StatusOpen, view.Status)
}

func TestStartAttempt_Execute_ReturnsFinishedForASubmittedOrATimedOutAttemptAndLoadsNothingElse(t *testing.T) {
	t.Parallel()
	exam := openExam()
	submitted := fixtures.Attempt(exam, fixtures.WithStartedAt(attemptNow.Add(-10*time.Minute)), fixtures.WithSubmittedAt(attemptNow.Add(-time.Minute)))
	timedOut := fixtures.Attempt(exam, fixtures.WithStartedAt(attemptNow.Add(-2*time.Hour)))
	for name, attempt := range map[string]domain.Attempt{"submitted": submitted, "timed out": timedOut} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m := newAttemptMocks(t) // The questions mock and Answers have no expected call.
			m.exams.EXPECT().FindByToken(t.Context(), "token").Return(exam, nil).Once()
			m.attempts.EXPECT().FindByExamAndEmail(t.Context(), exam.ID, mock.Anything).Return(attempt, nil).Once()

			_, created, err := m.startAttempt().Execute(t.Context(), startInput)

			require.ErrorIs(t, err, domain.ErrAttemptFinished)
			assert.False(t, created)
		})
	}
}

func TestStartAttempt_Execute_ReturnsNotOpenOrClosedForANewAttemptOutsideTheExamWindow(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		opens, closes time.Time
		want          error
	}{
		"before the open time": {attemptNow.Add(time.Hour), attemptNow.Add(2 * time.Hour), domain.ErrExamNotOpen},
		"after the close time": {attemptNow.Add(-2 * time.Hour), attemptNow.Add(-time.Hour), domain.ErrExamClosed},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m := newAttemptMocks(t) // It has no expected Create call, so a call fails the test.
			exam := fixtures.Exam(fixtures.WithExamWindow(tt.opens, tt.closes))
			m.exams.EXPECT().FindByToken(t.Context(), "token").Return(exam, nil).Once()
			m.attempts.EXPECT().FindByExamAndEmail(t.Context(), exam.ID, mock.Anything).Return(domain.Attempt{}, domain.ErrAttemptNotFound).Once()

			_, _, err := m.startAttempt().Execute(t.Context(), startInput)

			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestStartAttempt_Execute_ReturnsTheLinkErrorForAnUnknownTokenAndTheRuleErrorForABadName(t *testing.T) {
	t.Parallel()
	t.Run("unknown token", func(t *testing.T) {
		t.Parallel()
		m := newAttemptMocks(t)
		m.exams.EXPECT().FindByToken(t.Context(), "token").Return(domain.Exam{}, domain.ErrExamLinkNotFound).Once()

		_, _, err := m.startAttempt().Execute(t.Context(), startInput)

		require.ErrorIs(t, err, domain.ErrExamLinkNotFound)
	})
	t.Run("name is empty", func(t *testing.T) {
		t.Parallel()
		m := newAttemptMocks(t) // The Attempt repository has no expected call.
		m.exams.EXPECT().FindByToken(t.Context(), "token").Return(openExam(), nil).Once()

		_, _, err := m.startAttempt().Execute(t.Context(), app.StartAttemptInput{Token: "token", Name: " ", EmailAddress: "ann@example.com"})

		require.ErrorIs(t, err, domain.ErrEmptyCandidateName)
	})
}

func TestStartAttempt_Execute_GivesTheAttemptThatAnotherRequestSavedFirst(t *testing.T) {
	t.Parallel()
	m := newAttemptMocks(t)
	exam := openExam()
	other := fixtures.Attempt(exam, fixtures.WithCandidateEmail("ann@example.com"), fixtures.WithStartedAt(attemptNow))
	m.exams.EXPECT().FindByToken(t.Context(), "token").Return(exam, nil).Once()
	m.attempts.EXPECT().FindByExamAndEmail(t.Context(), exam.ID, mock.Anything).Return(domain.Attempt{}, domain.ErrAttemptNotFound).Once()
	m.attempts.EXPECT().Create(t.Context(), mock.Anything).Return(domain.ErrAttemptExists).Once()
	m.attempts.EXPECT().FindByExamAndEmail(t.Context(), exam.ID, mock.Anything).Return(other, nil).Once()
	m.expectView(t, exam, nil, nil)

	view, created, err := m.startAttempt().Execute(t.Context(), startInput)

	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, other.ID, view.Attempt.ID)
}

func TestStartAttempt_Execute_ReturnsTheErrorWhenTheRepositoryFails(t *testing.T) {
	t.Parallel()
	m := newAttemptMocks(t)
	cause := errors.New("The database is down.")
	m.exams.EXPECT().FindByToken(t.Context(), "token").Return(openExam(), nil).Once()
	m.attempts.EXPECT().FindByExamAndEmail(t.Context(), mock.Anything, mock.Anything).Return(domain.Attempt{}, cause).Once()

	_, _, err := m.startAttempt().Execute(t.Context(), startInput)

	require.ErrorIs(t, err, cause)
}

func TestGetAttempt_Execute_GivesAnOpenAttemptWithQuestionsAndOnlyTheStatusForAnotherAttempt(t *testing.T) {
	t.Parallel()
	exam := openExam()
	open := fixtures.Attempt(exam, fixtures.WithStartedAt(attemptNow.Add(-time.Minute)))
	timedOut := fixtures.Attempt(exam, fixtures.WithStartedAt(attemptNow.Add(-3*time.Hour)))
	m := newAttemptMocks(t)
	m.attempts.EXPECT().FindByID(t.Context(), open.ID).Return(open, nil).Once()
	m.attempts.EXPECT().FindByID(t.Context(), timedOut.ID).Return(timedOut, nil).Once()
	missing := uuid.New()
	m.attempts.EXPECT().FindByID(t.Context(), missing).Return(domain.Attempt{}, domain.ErrAttemptNotFound).Once()
	m.expectView(t, exam, nil, nil)
	useCase := app.NewGetAttempt(m.exams, m.questions, m.attempts, func() time.Time { return attemptNow })

	openView, err := useCase.Execute(t.Context(), open.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusOpen, openView.Status)
	assert.Equal(t, exam, openView.Exam)

	finished, err := useCase.Execute(t.Context(), timedOut.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusTimedOut, finished.Status)
	assert.Empty(t, finished.Questions)

	_, err = useCase.Execute(t.Context(), missing)
	require.ErrorIs(t, err, domain.ErrAttemptNotFound)
}

func TestGetExamLink_Execute_GivesTheExamOrTheLinkError(t *testing.T) {
	t.Parallel()
	exam := fixtures.Exam()
	exams := mocks.NewMockExamRepository(t)
	exams.EXPECT().FindByToken(t.Context(), "good").Return(exam, nil).Once()
	exams.EXPECT().FindByToken(t.Context(), "bad").Return(domain.Exam{}, domain.ErrExamLinkNotFound).Once()
	useCase := app.NewGetExamLink(exams)

	got, err := useCase.Execute(t.Context(), "good")
	require.NoError(t, err)
	assert.Equal(t, exam, got)
	_, err = useCase.Execute(t.Context(), "bad")
	require.ErrorIs(t, err, domain.ErrExamLinkNotFound)
}
