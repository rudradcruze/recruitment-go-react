//go:build e2e

package exam_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudradcruze/go-react-template/backend/api-test/e2e/testapi"
	"github.com/rudradcruze/go-react-template/backend/api-test/fixtures"
)

type resultBody struct {
	Name         string     `json:"name"`
	EmailAddress string     `json:"emailAddress"`
	Status       string     `json:"status"`
	Score        *scoreBody `json:"score"`
	StartedAt    string     `json:"startedAt"`
	FinishedAt   *string    `json:"finishedAt"`
}

func TestAPI_ListExamAttempts_GivesOneRowForEachAttemptInTheOrderOfTheStartTime(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	now := time.Now()
	q1, q2 := fixtures.SaveQuestion(t, api.Questions), fixtures.SaveQuestion(t, api.Questions)
	exam := fixtures.SaveExam(t, api.Exams, api.Questions, fixtures.WithQuestionIDs(q1.ID, q2.ID), fixtures.WithTimeLimit(30))
	other := fixtures.SaveExam(t, api.Exams, api.Questions)
	fixtures.SaveAttempt(t, api.Attempts, other) // An Attempt of another Exam does not show.

	submittedAt := now.Add(-70 * time.Minute)
	submitted := fixtures.SaveAttempt(t, api.Attempts, exam, fixtures.WithCandidateName("Bob"), fixtures.WithCandidateEmail("bob@example.com"),
		fixtures.WithStartedAt(now.Add(-80*time.Minute)), fixtures.WithSubmittedAt(submittedAt))
	abandoned := fixtures.SaveAttempt(t, api.Attempts, exam, fixtures.WithCandidateName("Cy"), fixtures.WithCandidateEmail("cy@example.com"),
		fixtures.WithStartedAt(now.Add(-60*time.Minute)))
	open := fixtures.SaveAttempt(t, api.Attempts, exam, fixtures.WithCandidateName("Ann"), fixtures.WithCandidateEmail("ann@example.com"),
		fixtures.WithStartedAt(now.Add(-5*time.Minute)))
	for _, answer := range []struct{ attempt, question, option any }{
		{submitted.ID, q1.ID, q1.Options[0].ID}, {submitted.ID, q2.ID, q2.Options[0].ID},
		{abandoned.ID, q1.ID, q1.Options[0].ID}, {abandoned.ID, q2.ID, q2.Options[1].ID},
		{open.ID, q1.ID, q1.Options[0].ID},
	} {
		_, err := api.DB.ExecContext(t.Context(), "INSERT INTO answers (attempt_id, question_id, option_id) VALUES ($1, $2, $3)",
			answer.attempt, answer.question, answer.option)
		require.NoError(t, err)
	}

	res := api.GetAs(t, token, "/api/v1/admin/exams/"+exam.ID.String()+"/attempts")

	require.Equal(t, http.StatusOK, res.StatusCode, string(res.Body))
	got := decode[[]resultBody](t, res)
	require.Len(t, got, 3)
	assert.Equal(t, []string{"Bob", "Cy", "Ann"}, []string{got[0].Name, got[1].Name, got[2].Name})

	assert.Equal(t, "bob@example.com", got[0].EmailAddress)
	assert.Equal(t, "submitted", got[0].Status)
	assert.Equal(t, &scoreBody{Correct: 2, Total: 2}, got[0].Score)
	require.NotNil(t, got[0].FinishedAt)
	finished, err := time.Parse(time.RFC3339Nano, *got[0].FinishedAt)
	require.NoError(t, err)
	assert.WithinDuration(t, submittedAt, finished, time.Second)

	assert.Equal(t, "timed_out", got[1].Status)
	assert.Equal(t, &scoreBody{Correct: 1, Total: 2}, got[1].Score)
	require.NotNil(t, got[1].FinishedAt)
	expiry, err := time.Parse(time.RFC3339Nano, *got[1].FinishedAt)
	require.NoError(t, err)
	assert.True(t, abandoned.ExpiresAt.Equal(expiry), "Without a submit, the finish time is the expiry.")

	assert.Equal(t, "open", got[2].Status)
	assert.Nil(t, got[2].Score)
	assert.Nil(t, got[2].FinishedAt)
	assert.Contains(t, string(res.Body), `"score":null`)
	assert.Contains(t, string(res.Body), `"finishedAt":null`)
}

func TestAPI_ListExamAttempts_HasNoAttemptIDBecauseItIsTheSecretOfTheCandidate(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	exam := fixtures.SaveExam(t, api.Exams, api.Questions)
	attempt := fixtures.SaveAttempt(t, api.Attempts, exam)

	res := api.GetAs(t, token, "/api/v1/admin/exams/"+exam.ID.String()+"/attempts")

	require.Equal(t, http.StatusOK, res.StatusCode, string(res.Body))
	assert.NotContains(t, string(res.Body), attempt.ID.String())
	assert.NotContains(t, strings.ToLower(string(res.Body)), `"id"`)
}

func TestAPI_ListExamAttempts_GivesAnEmptyArrayForAnExamWithoutAttempts(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	exam := fixtures.SaveExam(t, api.Exams, api.Questions)

	res := api.GetAs(t, token, "/api/v1/admin/exams/"+exam.ID.String()+"/attempts")

	require.Equal(t, http.StatusOK, res.StatusCode, string(res.Body))
	assert.JSONEq(t, `[]`, string(res.Body))
}

func TestAPI_ListExamAttempts_Returns404401And400(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	exam := fixtures.SaveExam(t, api.Exams, api.Questions)

	requireProblem(t, api.GetAs(t, token, "/api/v1/admin/exams/0198f2a0-0000-7000-8000-000000000000/attempts"), http.StatusNotFound, "Exam does not exist")
	requireProblem(t, api.GetAs(t, token, "/api/v1/admin/exams/not-a-uuid/attempts"), http.StatusBadRequest, "ID is not valid")
	requireProblem(t, api.Get(t, "/api/v1/admin/exams/"+exam.ID.String()+"/attempts"), http.StatusUnauthorized, "not logged in")
}
