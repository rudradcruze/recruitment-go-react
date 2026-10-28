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

func options(n int) []domain.OptionInput {
	result := make([]domain.OptionInput, n)
	for i := range result {
		result[i] = domain.OptionInput{Text: "Option " + string(rune('A'+i))}
	}
	result[0].Correct = true
	return result
}

func TestQuestion_New_MakesAQuestionWithIDsOfVersion7AndTheOptionsInOrder(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 27, 10, 0, 0, 123456789, time.UTC)

	got, err := domain.NewQuestion("  What is 2 + 2?  ", []domain.OptionInput{
		{Text: " 3 "}, {Text: "4", Correct: true}, {Text: "5"},
	}, now)

	require.NoError(t, err)
	assert.Equal(t, uuid.Version(7), got.ID.Version())
	assert.Equal(t, "What is 2 + 2?", got.Text)
	assert.Equal(t, time.Date(2026, 10, 27, 10, 0, 0, 123456000, time.UTC), got.CreatedAt)
	require.Len(t, got.Options, 3)
	for i, want := range []struct {
		text    string
		correct bool
	}{{"3", false}, {"4", true}, {"5", false}} {
		assert.Equal(t, want.text, got.Options[i].Text)
		assert.Equal(t, want.correct, got.Options[i].Correct)
		assert.Equal(t, uuid.Version(7), got.Options[i].ID.Version())
	}
	assert.NotEqual(t, got.Options[0].ID, got.Options[1].ID)
}

func TestQuestion_New_AcceptsAMultiLineTextAndExactlyTheLimits(t *testing.T) {
	t.Parallel()
	opts := options(6)
	opts[1].Text = strings.Repeat("é", domain.MaxOptionTextLength)

	_, err := domain.NewQuestion(strings.Repeat("é", domain.MaxQuestionTextLength), opts, time.Now())
	require.NoError(t, err)

	_, err = domain.NewQuestion("Line one\nline two\ttabbed", options(2), time.Now())
	require.NoError(t, err)
}

func TestQuestion_New_RejectsTheInputThatBreaksARule(t *testing.T) {
	t.Parallel()
	twoCorrect := options(3)
	twoCorrect[1].Correct = true
	noCorrect := options(3)
	noCorrect[0].Correct = false
	emptyOption := options(3)
	emptyOption[2].Text = "  \t "
	longOption := options(3)
	longOption[1].Text = strings.Repeat("a", domain.MaxOptionTextLength+1)
	nulOption := options(3)
	nulOption[1].Text = "a\x00b"
	newlineOption := options(3)
	newlineOption[1].Text = "a\nb"
	tests := map[string]struct {
		text    string
		options []domain.OptionInput
		want    error
	}{
		"text is empty":                 {"  ", options(2), domain.ErrEmptyQuestionText},
		"text is too long":              {strings.Repeat("a", domain.MaxQuestionTextLength+1), options(2), domain.ErrQuestionTextTooLong},
		"text has a NUL character":      {"a\x00b", options(2), domain.ErrTextNotAllowed},
		"1 option":                      {"Q", options(1), domain.ErrTooFewOptions},
		"no option":                     {"Q", nil, domain.ErrTooFewOptions},
		"7 options":                     {"Q", options(7), domain.ErrTooManyOptions},
		"2 correct options":             {"Q", twoCorrect, domain.ErrCorrectOptionCount},
		"no correct option":             {"Q", noCorrect, domain.ErrCorrectOptionCount},
		"an option text is empty":       {"Q", emptyOption, domain.ErrEmptyOptionText},
		"an option text is too long":    {"Q", longOption, domain.ErrOptionTextTooLong},
		"an option has a NUL character": {"Q", nulOption, domain.ErrTextNotAllowed},
		"an option has a new line":      {"Q", newlineOption, domain.ErrTextNotAllowed},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.NewQuestion(tt.text, tt.options, time.Now())

			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestQuestion_Change_KeepsTheIDAndTheCreationTimeAndMakesNewOptionIDs(t *testing.T) {
	t.Parallel()
	original, err := domain.NewQuestion("Old", options(2), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	changed, err := original.Change("New", options(3))

	require.NoError(t, err)
	assert.Equal(t, original.ID, changed.ID)
	assert.Equal(t, original.CreatedAt, changed.CreatedAt)
	assert.Equal(t, "New", changed.Text)
	require.Len(t, changed.Options, 3)
	assert.NotEqual(t, original.Options[0].ID, changed.Options[0].ID)
}

func TestQuestion_Change_ReturnsTheErrorOfARuleAndLeavesTheQuestionAsItIs(t *testing.T) {
	t.Parallel()
	original, err := domain.NewQuestion("Old", options(2), time.Now())
	require.NoError(t, err)

	_, err = original.Change("", options(2))

	require.ErrorIs(t, err, domain.ErrEmptyQuestionText)
	assert.Equal(t, "Old", original.Text)
}
