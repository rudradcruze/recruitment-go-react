package exam_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudradcruze/go-react-template/backend/api-test/fixtures"
	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
)

func TestCandidate_New_RemovesTheSpacesAndChangesTheEmailAddressToLowercase(t *testing.T) {
	t.Parallel()

	got, err := domain.NewCandidate("  Ann Smith ", " Ann@Example.COM ")

	require.NoError(t, err)
	assert.Equal(t, "Ann Smith", got.Name)
	assert.Equal(t, domain.EmailAddress("ann@example.com"), got.EmailAddress)
}

func TestCandidate_New_RejectsTheInputThatBreaksARule(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		name, emailAddress string
		want               error
	}{
		"name is empty":              {"  ", "ann@example.com", domain.ErrEmptyCandidateName},
		"name of 101 characters":     {strings.Repeat("a", 101), "ann@example.com", domain.ErrCandidateNameTooLong},
		"name has a new line":        {"Ann\nSmith", "ann@example.com", domain.ErrTextNotAllowed},
		"email address is not valid": {"Ann", "ann", domain.ErrInvalidEmailAddress},
		"email address is empty":     {"Ann", "", domain.ErrInvalidEmailAddress},
		"email address of 255 chars": {"Ann", strings.Repeat("a", 243) + "@example.com", domain.ErrInvalidEmailAddress},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.NewCandidate(tt.name, tt.emailAddress)

			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestCandidate_New_AcceptsANameOf100CharactersAndAnEmailAddressOf254(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", 254-len("@example.com")) + "@example.com"

	_, err := domain.NewCandidate(strings.Repeat("é", 100), long)

	require.NoError(t, err)
}

var attemptNow = time.Date(2026, 11, 1, 10, 0, 0, 123456789, time.UTC)

func openExam() domain.Exam {
	return fixtures.Exam(fixtures.WithExamWindow(attemptNow.Add(-time.Hour), attemptNow.Add(time.Hour)), fixtures.WithTimeLimit(30))
}

func TestAttempt_New_StartsAnOpenAttemptThatExpiresAfterTheTimeLimit(t *testing.T) {
	t.Parallel()
	exam := openExam()
	candidate, err := domain.NewCandidate("Ann", "ann@example.com")
	require.NoError(t, err)

	got, err := domain.NewAttempt(exam, candidate, attemptNow)

	require.NoError(t, err)
	assert.Equal(t, uuid.Version(4), got.ID.Version(), "The Attempt ID is a secret, so it is random.")
	assert.Equal(t, exam.ID, got.ExamID)
	assert.Equal(t, "Ann", got.CandidateName)
	assert.Equal(t, domain.EmailAddress("ann@example.com"), got.EmailAddress)
	assert.Equal(t, time.Date(2026, 11, 1, 10, 0, 0, 123456000, time.UTC), got.StartedAt)
	assert.Equal(t, got.StartedAt.Add(30*time.Minute), got.ExpiresAt)
	assert.Nil(t, got.SubmittedAt)
	assert.Equal(t, domain.StatusOpen, got.Status(attemptNow))
}

func TestAttempt_New_DoesNotCapTheExpiryAtTheCloseTimeOfTheExam(t *testing.T) {
	t.Parallel()
	exam := fixtures.Exam(fixtures.WithExamWindow(attemptNow.Add(-time.Hour), attemptNow.Add(time.Minute)), fixtures.WithTimeLimit(30))
	candidate, _ := domain.NewCandidate("Ann", "ann@example.com")

	got, err := domain.NewAttempt(exam, candidate, attemptNow)

	require.NoError(t, err)
	assert.True(t, got.ExpiresAt.After(exam.ClosesAt))
}

func TestAttempt_New_ChecksTheExamWindow(t *testing.T) {
	t.Parallel()
	exam := openExam()
	candidate, _ := domain.NewCandidate("Ann", "ann@example.com")
	tests := map[string]struct {
		now  time.Time
		want error
	}{
		"1 second before the open time":  {exam.OpensAt.Add(-time.Second), domain.ErrExamNotOpen},
		"at the open time":               {exam.OpensAt, nil},
		"1 second before the close time": {exam.ClosesAt.Add(-time.Second), nil},
		"at the close time":              {exam.ClosesAt, domain.ErrExamClosed},
		"after the close time":           {exam.ClosesAt.Add(time.Hour), domain.ErrExamClosed},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.NewAttempt(exam, candidate, tt.now)

			if tt.want == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tt.want)
			}
		})
	}
}

func TestAttempt_Status_FollowsTheRulesOfTheSpec(t *testing.T) {
	t.Parallel()
	started := attemptNow
	expires := started.Add(30 * time.Minute)
	at := func(d time.Duration) *time.Time { v := expires.Add(d); return &v }
	tests := map[string]struct {
		submittedAt *time.Time
		now         time.Time
		want        domain.AttemptStatus
	}{
		"not submitted, before the expiry":        {nil, expires.Add(-time.Second), domain.StatusOpen},
		"not submitted, at the expiry":            {nil, expires, domain.StatusOpen},
		"not submitted, after the expiry":         {nil, expires.Add(time.Second), domain.StatusTimedOut},
		"submitted before the expiry":             {at(-time.Minute), expires.Add(time.Hour), domain.StatusSubmitted},
		"submitted at the expiry":                 {at(0), expires.Add(time.Hour), domain.StatusSubmitted},
		"submitted 1 second after the expiry":     {at(time.Second), expires.Add(time.Hour), domain.StatusTimedOut},
		"submitted, and the check is before time": {at(-time.Minute), started, domain.StatusSubmitted},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			attempt := domain.Attempt{StartedAt: started, ExpiresAt: expires, SubmittedAt: tt.submittedAt}

			assert.Equal(t, tt.want, attempt.Status(tt.now))
		})
	}
}

func TestAttemptStatus_HasTheNamesOfTheSpec(t *testing.T) {
	t.Parallel()

	assert.EqualValues(t, "open", domain.StatusOpen)
	assert.EqualValues(t, "submitted", domain.StatusSubmitted)
	assert.EqualValues(t, "timed_out", domain.StatusTimedOut)
}

func TestExam_ChangeAfterAttempts_ChangesOnlyTheCloseTime(t *testing.T) {
	t.Parallel()
	exam := fixtures.Exam()
	input := domain.ExamInput{
		Title: exam.Title, QuestionIDs: exam.QuestionIDs, OpensAt: exam.OpensAt,
		ClosesAt: exam.ClosesAt.Add(2 * time.Hour), TimeLimitMinutes: exam.TimeLimitMinutes,
	}

	got, err := exam.ChangeAfterAttempts(input)

	require.NoError(t, err)
	assert.Equal(t, exam.ClosesAt.Add(2*time.Hour).UTC().Truncate(time.Microsecond), got.ClosesAt)
	got.ClosesAt = exam.ClosesAt
	assert.Equal(t, exam, got)
}

func TestExam_ChangeAfterAttempts_ReturnsLockedForAChangeOfAnotherField(t *testing.T) {
	t.Parallel()
	exam := fixtures.Exam()
	base := func() domain.ExamInput {
		return domain.ExamInput{
			Title: exam.Title, QuestionIDs: exam.QuestionIDs, OpensAt: exam.OpensAt,
			ClosesAt: exam.ClosesAt, TimeLimitMinutes: exam.TimeLimitMinutes,
		}
	}
	tests := map[string]func(*domain.ExamInput){
		"title":      func(i *domain.ExamInput) { i.Title = "Another title" },
		"Questions":  func(i *domain.ExamInput) { i.QuestionIDs = []uuid.UUID{uuid.New()} },
		"open time":  func(i *domain.ExamInput) { i.OpensAt = i.OpensAt.Add(-time.Minute) },
		"Time limit": func(i *domain.ExamInput) { i.TimeLimitMinutes++ },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			input := base()
			change(&input)

			_, err := exam.ChangeAfterAttempts(input)

			require.ErrorIs(t, err, domain.ErrExamLocked)
		})
	}
}

func TestExam_ChangeAfterAttempts_AcceptsTheSameOpenTimeInAnotherTimezone(t *testing.T) {
	t.Parallel()
	exam := fixtures.Exam()
	input := domain.ExamInput{
		Title: exam.Title, QuestionIDs: exam.QuestionIDs, OpensAt: exam.OpensAt.In(time.FixedZone("UTC+6", 6*3600)),
		ClosesAt: exam.ClosesAt, TimeLimitMinutes: exam.TimeLimitMinutes,
	}

	_, err := exam.ChangeAfterAttempts(input)

	require.NoError(t, err)
}

func TestExam_ChangeAfterAttempts_ReturnsTheErrorOfARuleBeforeLocked(t *testing.T) {
	t.Parallel()
	exam := fixtures.Exam()

	_, err := exam.ChangeAfterAttempts(domain.ExamInput{Title: "", QuestionIDs: exam.QuestionIDs, OpensAt: exam.OpensAt, ClosesAt: exam.ClosesAt, TimeLimitMinutes: 30})

	require.ErrorIs(t, err, domain.ErrEmptyTitle)
}
