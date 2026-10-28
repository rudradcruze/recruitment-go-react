package exam_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rudradcruze/go-react-template/backend/api-test/fixtures"
	"github.com/rudradcruze/go-react-template/backend/internal/exam/app"
	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
)

func (m attemptMocks) saveAnswer(now time.Time) *app.SaveAnswer {
	return app.NewSaveAnswer(m.exams, m.questions, m.attempts, func() time.Time { return now })
}

func (m attemptMocks) submitAttempt(now time.Time) *app.SubmitAttempt {
	return app.NewSubmitAttempt(m.exams, m.questions, m.attempts, func() time.Time { return now })
}

func TestSaveAnswer_Execute_SavesTheAnswerOfAnOpenAttempt(t *testing.T) {
	t.Parallel()
	m := newAttemptMocks(t)
	question := fixtures.Question()
	exam := fixtures.Exam(fixtures.WithQuestionIDs(question.ID))
	attempt := fixtures.Attempt(exam, fixtures.WithStartedAt(attemptNow.Add(-time.Minute)))
	answer := domain.Answer{QuestionID: question.ID, OptionID: question.Options[2].ID}
	m.attempts.EXPECT().FindByID(t.Context(), attempt.ID).Return(attempt, nil).Once()
	m.exams.EXPECT().FindByID(t.Context(), exam.ID).Return(exam, nil).Once()
	m.questions.EXPECT().FindByIDs(t.Context(), exam.QuestionIDs).Return([]domain.Question{question}, nil).Once()
	m.attempts.EXPECT().SaveAnswer(t.Context(), attempt.ID, answer, attemptNow).Return(nil).Once()

	err := m.saveAnswer(attemptNow).Execute(t.Context(), attempt.ID, answer)

	require.NoError(t, err)
}

func TestSaveAnswer_Execute_ReturnsClosedAfterTheExpiryOrTheSubmitAndLoadsNothingElse(t *testing.T) {
	t.Parallel()
	exam := openExam()
	expired := fixtures.Attempt(exam, fixtures.WithStartedAt(attemptNow.Add(-2*time.Hour)))
	submitted := fixtures.Attempt(exam, fixtures.WithStartedAt(attemptNow.Add(-10*time.Minute)), fixtures.WithSubmittedAt(attemptNow.Add(-time.Minute)))
	for name, attempt := range map[string]domain.Attempt{"expired": expired, "submitted": submitted} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m := newAttemptMocks(t) // The other mocks have no expected call, so a call fails the test.
			m.attempts.EXPECT().FindByID(t.Context(), attempt.ID).Return(attempt, nil).Once()

			err := m.saveAnswer(attemptNow).Execute(t.Context(), attempt.ID, domain.Answer{QuestionID: uuid.New(), OptionID: uuid.New()})

			require.ErrorIs(t, err, domain.ErrAttemptClosed)
		})
	}
}

func TestSaveAnswer_Execute_ReturnsNotFoundForAnUnknownAttemptAndInvalidForAnOptionOfAnotherQuestion(t *testing.T) {
	t.Parallel()
	t.Run("unknown Attempt", func(t *testing.T) {
		t.Parallel()
		m := newAttemptMocks(t)
		id := uuid.New()
		m.attempts.EXPECT().FindByID(t.Context(), id).Return(domain.Attempt{}, domain.ErrAttemptNotFound).Once()

		err := m.saveAnswer(attemptNow).Execute(t.Context(), id, domain.Answer{})

		require.ErrorIs(t, err, domain.ErrAttemptNotFound)
	})
	t.Run("an Option of another Question", func(t *testing.T) {
		t.Parallel()
		m := newAttemptMocks(t) // SaveAnswer has no expected call, so a call fails the test.
		first, second := fixtures.Question(), fixtures.Question()
		exam := fixtures.Exam(fixtures.WithQuestionIDs(first.ID, second.ID))
		attempt := fixtures.Attempt(exam, fixtures.WithStartedAt(attemptNow.Add(-time.Minute)))
		m.attempts.EXPECT().FindByID(t.Context(), attempt.ID).Return(attempt, nil).Once()
		m.exams.EXPECT().FindByID(t.Context(), exam.ID).Return(exam, nil).Once()
		m.questions.EXPECT().FindByIDs(t.Context(), exam.QuestionIDs).Return([]domain.Question{first, second}, nil).Once()

		err := m.saveAnswer(attemptNow).Execute(t.Context(), attempt.ID, domain.Answer{QuestionID: first.ID, OptionID: second.Options[0].ID})

		require.ErrorIs(t, err, domain.ErrInvalidAnswer)
	})
}

func TestSaveAnswer_Execute_ReturnsClosedWhenTheRepositoryFindsTheAttemptClosed(t *testing.T) {
	t.Parallel()
	// A submit or the expiry can come between the check of the use case and the save. The repository checks again.
	m := newAttemptMocks(t)
	question := fixtures.Question()
	exam := fixtures.Exam(fixtures.WithQuestionIDs(question.ID))
	attempt := fixtures.Attempt(exam, fixtures.WithStartedAt(attemptNow.Add(-time.Minute)))
	m.attempts.EXPECT().FindByID(t.Context(), attempt.ID).Return(attempt, nil).Once()
	m.exams.EXPECT().FindByID(t.Context(), exam.ID).Return(exam, nil).Once()
	m.questions.EXPECT().FindByIDs(t.Context(), exam.QuestionIDs).Return([]domain.Question{question}, nil).Once()
	m.attempts.EXPECT().SaveAnswer(t.Context(), attempt.ID, mock.Anything, attemptNow).Return(domain.ErrAttemptClosed).Once()

	err := m.saveAnswer(attemptNow).Execute(t.Context(), attempt.ID, domain.Answer{QuestionID: question.ID, OptionID: question.Options[0].ID})

	require.ErrorIs(t, err, domain.ErrAttemptClosed)
}

func TestSubmitAttempt_Execute_StoresTheSubmitTimeAndGivesTheScore(t *testing.T) {
	t.Parallel()
	m := newAttemptMocks(t)
	q1, q2 := fixtures.Question(), fixtures.Question()
	exam := fixtures.Exam(fixtures.WithQuestionIDs(q1.ID, q2.ID))
	attempt := fixtures.Attempt(exam, fixtures.WithStartedAt(attemptNow.Add(-time.Minute)))
	m.attempts.EXPECT().FindByID(t.Context(), attempt.ID).Return(attempt, nil).Once()
	m.attempts.EXPECT().MarkSubmitted(t.Context(), attempt.ID, attemptNow.Truncate(time.Microsecond)).Return(nil).Once()
	m.exams.EXPECT().FindByID(t.Context(), exam.ID).Return(exam, nil).Once()
	m.questions.EXPECT().FindByIDs(t.Context(), exam.QuestionIDs).Return([]domain.Question{q1, q2}, nil).Once()
	m.attempts.EXPECT().Answers(t.Context(), attempt.ID).Return([]domain.Answer{
		{QuestionID: q1.ID, OptionID: q1.Options[0].ID}, {QuestionID: q2.ID, OptionID: q2.Options[1].ID},
	}, nil).Once()

	got, err := m.submitAttempt(attemptNow).Execute(t.Context(), attempt.ID)

	require.NoError(t, err)
	assert.Equal(t, domain.Score{Correct: 1, Total: 2}, got)
}

func TestSubmitAttempt_Execute_WorksAfterTheExpiryWithTheSavedAnswers(t *testing.T) {
	t.Parallel()
	m := newAttemptMocks(t)
	question := fixtures.Question()
	exam := fixtures.Exam(fixtures.WithQuestionIDs(question.ID))
	attempt := fixtures.Attempt(exam, fixtures.WithStartedAt(attemptNow.Add(-3*time.Hour)))
	m.attempts.EXPECT().FindByID(t.Context(), attempt.ID).Return(attempt, nil).Once()
	m.attempts.EXPECT().MarkSubmitted(t.Context(), attempt.ID, attemptNow.Truncate(time.Microsecond)).Return(nil).Once()
	m.exams.EXPECT().FindByID(t.Context(), exam.ID).Return(exam, nil).Once()
	m.questions.EXPECT().FindByIDs(t.Context(), exam.QuestionIDs).Return([]domain.Question{question}, nil).Once()
	m.attempts.EXPECT().Answers(t.Context(), attempt.ID).Return([]domain.Answer{{QuestionID: question.ID, OptionID: question.Options[0].ID}}, nil).Once()

	got, err := m.submitAttempt(attemptNow).Execute(t.Context(), attempt.ID)

	require.NoError(t, err)
	assert.Equal(t, domain.Score{Correct: 1, Total: 1}, got)
}

func TestSubmitAttempt_Execute_ReturnsTheErrorForASecondSubmitAndForAnUnknownAttempt(t *testing.T) {
	t.Parallel()
	exam := openExam()
	submitted := fixtures.Attempt(exam, fixtures.WithStartedAt(attemptNow.Add(-10*time.Minute)), fixtures.WithSubmittedAt(attemptNow.Add(-time.Minute)))
	t.Run("second submit", func(t *testing.T) {
		t.Parallel()
		m := newAttemptMocks(t) // MarkSubmitted has no expected call, so a call fails the test.
		m.attempts.EXPECT().FindByID(t.Context(), submitted.ID).Return(submitted, nil).Once()

		_, err := m.submitAttempt(attemptNow).Execute(t.Context(), submitted.ID)

		require.ErrorIs(t, err, domain.ErrAttemptAlreadySubmitted)
	})
	t.Run("a second submit that wins the race", func(t *testing.T) {
		t.Parallel()
		m := newAttemptMocks(t)
		open := fixtures.Attempt(exam, fixtures.WithStartedAt(attemptNow.Add(-time.Minute)))
		m.attempts.EXPECT().FindByID(t.Context(), open.ID).Return(open, nil).Once()
		m.attempts.EXPECT().MarkSubmitted(t.Context(), open.ID, attemptNow.Truncate(time.Microsecond)).Return(domain.ErrAttemptAlreadySubmitted).Once()

		_, err := m.submitAttempt(attemptNow).Execute(t.Context(), open.ID)

		require.ErrorIs(t, err, domain.ErrAttemptAlreadySubmitted)
	})
	t.Run("unknown Attempt", func(t *testing.T) {
		t.Parallel()
		m := newAttemptMocks(t)
		id := uuid.New()
		m.attempts.EXPECT().FindByID(t.Context(), id).Return(domain.Attempt{}, domain.ErrAttemptNotFound).Once()

		_, err := m.submitAttempt(attemptNow).Execute(t.Context(), id)

		require.ErrorIs(t, err, domain.ErrAttemptNotFound)
	})
}
