package exam_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudradcruze/go-react-template/backend/api-test/fixtures"
	"github.com/rudradcruze/go-react-template/backend/internal/exam/app"
	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
)

func TestListExamAttempts_Execute_GivesTheStatusTheScoreAndTheFinishTimeOfEachAttempt(t *testing.T) {
	t.Parallel()
	m := newAttemptMocks(t)
	q1, q2 := fixtures.Question(), fixtures.Question()
	exam := fixtures.Exam(fixtures.WithQuestionIDs(q1.ID, q2.ID), fixtures.WithTimeLimit(30))
	started := attemptNow.Add(-time.Hour)
	submittedAt := started.Add(10 * time.Minute)
	lateSubmitAt := started.Add(40 * time.Minute)
	open := fixtures.Attempt(exam, fixtures.WithStartedAt(attemptNow.Add(-5*time.Minute)))
	submitted := fixtures.Attempt(exam, fixtures.WithStartedAt(started), fixtures.WithSubmittedAt(submittedAt))
	abandoned := fixtures.Attempt(exam, fixtures.WithStartedAt(started))
	late := fixtures.Attempt(exam, fixtures.WithStartedAt(started), fixtures.WithSubmittedAt(lateSubmitAt))
	correct := func(q domain.Question) domain.Answer {
		return domain.Answer{QuestionID: q.ID, OptionID: q.Options[0].ID}
	}
	m.exams.EXPECT().FindByID(t.Context(), exam.ID).Return(exam, nil).Once()
	m.questions.EXPECT().FindByIDs(t.Context(), exam.QuestionIDs).Return([]domain.Question{q1, q2}, nil).Once()
	m.attempts.EXPECT().ListByExam(t.Context(), exam.ID).Return([]domain.Attempt{open, submitted, abandoned, late}, nil).Once()
	m.attempts.EXPECT().AnswersOfExam(t.Context(), exam.ID).Return(map[uuid.UUID][]domain.Answer{
		open.ID:      {correct(q1)},
		submitted.ID: {correct(q1), correct(q2)},
		abandoned.ID: {correct(q2)},
	}, nil).Once()

	got, err := app.NewListExamAttempts(m.exams, m.questions, m.attempts, func() time.Time { return attemptNow }).Execute(t.Context(), exam.ID)

	require.NoError(t, err)
	require.Len(t, got, 4)
	assert.Equal(t, open, got[0].Attempt)
	assert.Equal(t, domain.StatusOpen, got[0].Status)
	assert.Nil(t, got[0].Score, "The Score of an open Attempt is null.")
	assert.Nil(t, got[0].FinishedAt)

	assert.Equal(t, domain.StatusSubmitted, got[1].Status)
	assert.Equal(t, &domain.Score{Correct: 2, Total: 2}, got[1].Score)
	assert.Equal(t, submittedAt.UTC().Truncate(time.Microsecond), *got[1].FinishedAt)

	assert.Equal(t, domain.StatusTimedOut, got[2].Status)
	assert.Equal(t, &domain.Score{Correct: 1, Total: 2}, got[2].Score, "A timed-out Attempt counts its saved Answers.")
	assert.Equal(t, abandoned.ExpiresAt, *got[2].FinishedAt, "Without a submit, the finish time is the expiry.")

	assert.Equal(t, domain.StatusTimedOut, got[3].Status)
	assert.Equal(t, &domain.Score{Correct: 0, Total: 2}, got[3].Score)
	assert.Equal(t, lateSubmitAt.UTC().Truncate(time.Microsecond), *got[3].FinishedAt, "With a late submit, the finish time is the submit time.")
}

func TestListExamAttempts_Execute_GivesAnEmptyListAndReturnsNotFoundForAnUnknownExam(t *testing.T) {
	t.Parallel()
	m := newAttemptMocks(t)
	exam := fixtures.Exam()
	m.exams.EXPECT().FindByID(t.Context(), exam.ID).Return(exam, nil).Once()
	m.questions.EXPECT().FindByIDs(t.Context(), exam.QuestionIDs).Return(nil, nil).Once()
	m.attempts.EXPECT().ListByExam(t.Context(), exam.ID).Return(nil, nil).Once()
	m.attempts.EXPECT().AnswersOfExam(t.Context(), exam.ID).Return(nil, nil).Once()
	missing := uuid.New()
	m.exams.EXPECT().FindByID(t.Context(), missing).Return(domain.Exam{}, domain.ErrExamNotFound).Once()
	useCase := app.NewListExamAttempts(m.exams, m.questions, m.attempts, time.Now)

	got, err := useCase.Execute(t.Context(), exam.ID)
	require.NoError(t, err)
	assert.Empty(t, got)

	_, err = useCase.Execute(t.Context(), missing)
	require.ErrorIs(t, err, domain.ErrExamNotFound)
}
