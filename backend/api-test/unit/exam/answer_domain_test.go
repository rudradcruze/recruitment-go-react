package exam_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudradcruze/go-react-template/backend/api-test/fixtures"
	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
)

func TestAttempt_AcceptsAnswers_OnlyBeforeTheExpiryAndBeforeTheSubmit(t *testing.T) {
	t.Parallel()
	started := attemptNow
	expires := started.Add(30 * time.Minute)
	submitted := expires.Add(-time.Minute)
	tests := map[string]struct {
		submittedAt *time.Time
		now         time.Time
		want        error
	}{
		"before the expiry":         {nil, expires.Add(-time.Second), nil},
		"at the expiry":             {nil, expires, nil},
		"1 second after the expiry": {nil, expires.Add(time.Second), domain.ErrAttemptClosed},
		"after the submit":          {&submitted, expires.Add(-30 * time.Second), domain.ErrAttemptClosed},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			attempt := domain.Attempt{StartedAt: started, ExpiresAt: expires, SubmittedAt: tt.submittedAt}

			err := attempt.AcceptsAnswers(tt.now)

			if tt.want == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tt.want)
			}
		})
	}
}

func TestAttempt_Submit_StoresTheSubmitTimeAndRejectsASecondSubmit(t *testing.T) {
	t.Parallel()
	attempt := domain.Attempt{StartedAt: attemptNow, ExpiresAt: attemptNow.Add(time.Hour)}
	at := attemptNow.Add(10*time.Minute + 123*time.Nanosecond)

	submitted, err := attempt.Submit(at)

	require.NoError(t, err)
	require.NotNil(t, submitted.SubmittedAt)
	assert.Equal(t, at.UTC().Truncate(time.Microsecond), *submitted.SubmittedAt)
	assert.Nil(t, attempt.SubmittedAt, "Submit does not change the Attempt that it is called on.")
	_, err = submitted.Submit(at.Add(time.Second))
	require.ErrorIs(t, err, domain.ErrAttemptAlreadySubmitted)
}

func TestAttempt_Submit_WorksAfterTheExpiry(t *testing.T) {
	t.Parallel()
	attempt := domain.Attempt{StartedAt: attemptNow, ExpiresAt: attemptNow.Add(time.Hour)}

	submitted, err := attempt.Submit(attemptNow.Add(2 * time.Hour))

	require.NoError(t, err)
	assert.Equal(t, domain.StatusTimedOut, submitted.Status(attemptNow.Add(3*time.Hour)))
}

func TestAnswer_Validate_AcceptsAnOptionOfAQuestionOfTheExamAndRejectsOthers(t *testing.T) {
	t.Parallel()
	first, second, outside := fixtures.Question(), fixtures.Question(), fixtures.Question()
	exam := fixtures.Exam(fixtures.WithQuestionIDs(first.ID, second.ID))
	questions := []domain.Question{first, second}
	tests := map[string]struct {
		answer domain.Answer
		want   error
	}{
		"an Option of the first Question":    {domain.Answer{QuestionID: first.ID, OptionID: first.Options[3].ID}, nil},
		"an Option of the second Question":   {domain.Answer{QuestionID: second.ID, OptionID: second.Options[0].ID}, nil},
		"a Question that is not in the Exam": {domain.Answer{QuestionID: outside.ID, OptionID: outside.Options[0].ID}, domain.ErrInvalidAnswer},
		"an Option of another Question":      {domain.Answer{QuestionID: first.ID, OptionID: second.Options[0].ID}, domain.ErrInvalidAnswer},
		"an Option that does not exist":      {domain.Answer{QuestionID: first.ID, OptionID: uuid.New()}, domain.ErrInvalidAnswer},
		"no Option":                          {domain.Answer{QuestionID: first.ID}, domain.ErrInvalidAnswer},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := domain.ValidateAnswer(exam, questions, tt.answer)

			if tt.want == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tt.want)
			}
		})
	}
}

func TestScore_Of_CountsOnePointForEachCorrectAnswerAndZeroForAWrongOrSkippedQuestion(t *testing.T) {
	t.Parallel()
	q1, q2, q3, q4 := fixtures.Question(), fixtures.Question(), fixtures.Question(), fixtures.Question()
	questions := []domain.Question{q1, q2, q3, q4}
	correct := func(q domain.Question) domain.Answer {
		return domain.Answer{QuestionID: q.ID, OptionID: q.Options[0].ID}
	}
	wrong := func(q domain.Question) domain.Answer {
		return domain.Answer{QuestionID: q.ID, OptionID: q.Options[1].ID}
	}

	got := domain.ScoreOf(questions, []domain.Answer{correct(q1), wrong(q2), correct(q4)})

	assert.Equal(t, domain.Score{Correct: 2, Total: 4}, got)
}

func TestScore_Of_IsZeroOfTheTotalWithoutAnswersAndIgnoresAnAnswerOfAQuestionThatIsNotInTheList(t *testing.T) {
	t.Parallel()
	q1, other := fixtures.Question(), fixtures.Question()

	assert.Equal(t, domain.Score{Correct: 0, Total: 1}, domain.ScoreOf([]domain.Question{q1}, nil))
	assert.Equal(t, domain.Score{Correct: 0, Total: 1},
		domain.ScoreOf([]domain.Question{q1}, []domain.Answer{{QuestionID: other.ID, OptionID: other.Options[0].ID}}))
}
