package exam_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
)

var (
	examOpens  = time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	examCloses = time.Date(2026, 11, 1, 17, 0, 0, 0, time.UTC)
)

func examInput(questionCount int) domain.ExamInput {
	ids := make([]uuid.UUID, questionCount)
	for i := range ids {
		ids[i] = uuid.New()
	}
	return domain.ExamInput{Title: "Go basics", QuestionIDs: ids, OpensAt: examOpens, ClosesAt: examCloses, TimeLimitMinutes: 60}
}

func TestExam_New_MakesAnExamWithAnIDOfVersion7AndALinkToken(t *testing.T) {
	t.Parallel()
	input := examInput(3)
	input.Title = "  Go basics "
	input.OpensAt = time.Date(2026, 11, 1, 9, 0, 0, 123456789, time.FixedZone("UTC+6", 6*3600))
	now := time.Date(2026, 10, 27, 10, 0, 0, 123456789, time.UTC)

	got, err := domain.NewExam(input, now)

	require.NoError(t, err)
	assert.Equal(t, uuid.Version(7), got.ID.Version())
	assert.Equal(t, "Go basics", got.Title)
	assert.Equal(t, input.QuestionIDs, got.QuestionIDs)
	assert.Equal(t, time.Date(2026, 11, 1, 3, 0, 0, 123456000, time.UTC), got.OpensAt)
	assert.Equal(t, time.UTC, got.OpensAt.Location())
	assert.Equal(t, 60, got.TimeLimitMinutes)
	assert.Equal(t, time.Date(2026, 10, 27, 10, 0, 0, 123456000, time.UTC), got.CreatedAt)
	assert.Regexp(t, `^[0-9A-Za-z]{22}$`, got.LinkToken)
}

func TestExam_New_MakesADifferentLinkTokenForEachExam(t *testing.T) {
	t.Parallel()
	tokens := map[string]bool{}
	for range 50 {
		got, err := domain.NewExam(examInput(1), examOpens)
		require.NoError(t, err)
		tokens[got.LinkToken] = true
	}

	assert.Len(t, tokens, 50)
}

func TestExam_New_AcceptsTheLimitsAndRejectsTheInputThatBreaksARule(t *testing.T) {
	t.Parallel()
	with := func(change func(*domain.ExamInput)) domain.ExamInput {
		input := examInput(2)
		change(&input)
		return input
	}
	duplicate := examInput(2)
	duplicate.QuestionIDs[1] = duplicate.QuestionIDs[0]
	tests := map[string]struct {
		input domain.ExamInput
		want  error // nil means that the input is valid.
	}{
		"200 Questions":                 {examInput(200), nil},
		"201 Questions":                 {examInput(201), domain.ErrTooManyQuestions},
		"no Question":                   {examInput(0), domain.ErrTooFewQuestions},
		"the same Question twice":       {duplicate, domain.ErrDuplicateQuestion},
		"title of 200 characters":       {with(func(i *domain.ExamInput) { i.Title = strings.Repeat("é", 200) }), nil},
		"title of 201 characters":       {with(func(i *domain.ExamInput) { i.Title = strings.Repeat("a", 201) }), domain.ErrTitleTooLong},
		"empty title":                   {with(func(i *domain.ExamInput) { i.Title = "  " }), domain.ErrEmptyTitle},
		"title with a new line":         {with(func(i *domain.ExamInput) { i.Title = "a\nb" }), domain.ErrTextNotAllowed},
		"Time limit of 1 minute":        {with(func(i *domain.ExamInput) { i.TimeLimitMinutes = 1 }), nil},
		"Time limit of 480 minutes":     {with(func(i *domain.ExamInput) { i.TimeLimitMinutes = 480 }), nil},
		"Time limit of 0 minutes":       {with(func(i *domain.ExamInput) { i.TimeLimitMinutes = 0 }), domain.ErrInvalidTimeLimit},
		"Time limit of 481 minutes":     {with(func(i *domain.ExamInput) { i.TimeLimitMinutes = 481 }), domain.ErrInvalidTimeLimit},
		"close time equal to open time": {with(func(i *domain.ExamInput) { i.ClosesAt = i.OpensAt }), domain.ErrInvalidWindow},
		"close time before open time":   {with(func(i *domain.ExamInput) { i.ClosesAt = i.OpensAt.Add(-time.Second) }), domain.ErrInvalidWindow},
		"open time missing":             {with(func(i *domain.ExamInput) { i.OpensAt = time.Time{} }), domain.ErrInvalidWindow},
		"close time missing":            {with(func(i *domain.ExamInput) { i.ClosesAt = time.Time{} }), domain.ErrInvalidWindow},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.NewExam(tt.input, examOpens)

			if tt.want == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tt.want)
			}
		})
	}
}

func TestExam_Change_KeepsTheIDTheLinkTokenAndTheCreationTime(t *testing.T) {
	t.Parallel()
	original, err := domain.NewExam(examInput(2), examOpens)
	require.NoError(t, err)
	input := examInput(1)
	input.Title = "New title"

	changed, err := original.Change(input)

	require.NoError(t, err)
	assert.Equal(t, original.ID, changed.ID)
	assert.Equal(t, original.LinkToken, changed.LinkToken)
	assert.Equal(t, original.CreatedAt, changed.CreatedAt)
	assert.Equal(t, "New title", changed.Title)
	assert.Len(t, changed.QuestionIDs, 1)
}

func TestExam_Change_ReturnsTheErrorOfARule(t *testing.T) {
	t.Parallel()
	original, err := domain.NewExam(examInput(2), examOpens)
	require.NoError(t, err)

	_, err = original.Change(examInput(0))

	require.ErrorIs(t, err, domain.ErrTooFewQuestions)
}

func TestExam_RegenerateLink_MakesANewTokenAndChangesNothingElse(t *testing.T) {
	t.Parallel()
	original, err := domain.NewExam(examInput(2), examOpens)
	require.NoError(t, err)

	got, err := original.RegenerateLink()

	require.NoError(t, err)
	assert.NotEqual(t, original.LinkToken, got.LinkToken)
	assert.Regexp(t, `^[0-9A-Za-z]{22}$`, got.LinkToken)
	got.LinkToken = original.LinkToken
	assert.Equal(t, original, got)
}
