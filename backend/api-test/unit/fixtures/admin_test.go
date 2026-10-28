package fixtures_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/rudradcruze/go-react-template/backend/api-test/fixtures"
)

func TestAdmin_GivesAValidAdminWithAUniqueEmailAddressInEachCall(t *testing.T) {
	t.Parallel()

	first, second := fixtures.Admin(), fixtures.Admin()

	assert.NotEqual(t, first.EmailAddress, second.EmailAddress)
	assert.NotEqual(t, first.ID, second.ID)
}

func TestAdmin_AppliesTheOptions(t *testing.T) {
	t.Parallel()

	got := fixtures.Admin(fixtures.WithAdminEmailAddress("Ann@Example.com"))

	assert.EqualValues(t, "ann@example.com", got.EmailAddress)
}

func TestExam_GivesAValidExamWithAnOpenWindowAndAUniqueTitle(t *testing.T) {
	t.Parallel()

	first, second := fixtures.Exam(), fixtures.Exam()

	assert.NotEqual(t, first.Title, second.Title)
	assert.True(t, first.OpensAt.Before(time.Now()))
	assert.True(t, first.ClosesAt.After(time.Now()))
	assert.Len(t, first.QuestionIDs, 1)
}
