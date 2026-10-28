//go:build e2e

package exam_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudradcruze/go-react-template/backend/api-test/e2e/testapi"
	"github.com/rudradcruze/go-react-template/backend/api-test/fixtures"
	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
)

type scoreBody struct {
	Correct int `json:"correct"`
	Total   int `json:"total"`
}

func answerPath(attemptID, questionID uuid.UUID) string {
	return fmt.Sprintf("/api/v1/attempts/%s/answers/%s", attemptID, questionID)
}

func answerJSON(optionID uuid.UUID) string { return `{"optionId":"` + optionID.String() + `"}` }

func putAnswer(t *testing.T, api *testapi.API, attemptID uuid.UUID, question domain.Question, option domain.Option) testapi.Response {
	t.Helper()
	return api.Do(t, http.MethodPut, answerPath(attemptID, question.ID), "application/json", []byte(answerJSON(option.ID)))
}

func storedOption(t *testing.T, api *testapi.API, attemptID, questionID uuid.UUID) (uuid.UUID, bool) {
	t.Helper()
	rows, err := api.DB.QueryContext(t.Context(), "SELECT option_id FROM answers WHERE attempt_id = $1 AND question_id = $2", attemptID, questionID)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return uuid.Nil, false
	}
	var id uuid.UUID
	require.NoError(t, rows.Scan(&id))
	return id, true
}

// startedAttempt saves an Exam with 3 Questions, starts an Attempt through the API, and gives them.
func startedAttempt(t *testing.T, api *testapi.API) (domain.Exam, []domain.Question, uuid.UUID) {
	t.Helper()
	q1, q2, q3 := fixtures.SaveQuestion(t, api.Questions), fixtures.SaveQuestion(t, api.Questions), fixtures.SaveQuestion(t, api.Questions)
	exam := fixtures.SaveExam(t, api.Exams, api.Questions, fixtures.WithQuestionIDs(q1.ID, q2.ID, q3.ID), fixtures.WithTimeLimit(30))
	res := api.PostJSON(t, startPath(exam), startBody)
	require.Equal(t, http.StatusCreated, res.StatusCode, string(res.Body))
	return exam, []domain.Question{q1, q2, q3}, uuid.MustParse(decode[attemptBody](t, res).ID)
}

func TestAPI_SaveAnswer_Returns204AndAFollowingCallReplacesTheAnswer(t *testing.T) {
	api := testapi.Start(t)
	_, questions, attemptID := startedAttempt(t, api)

	first := putAnswer(t, api, attemptID, questions[0], questions[0].Options[1])
	require.Equal(t, http.StatusNoContent, first.StatusCode, string(first.Body))
	stored, found := storedOption(t, api, attemptID, questions[0].ID)
	require.True(t, found)
	assert.Equal(t, questions[0].Options[1].ID, stored)

	second := putAnswer(t, api, attemptID, questions[0], questions[0].Options[3])
	require.Equal(t, http.StatusNoContent, second.StatusCode, string(second.Body))
	stored, _ = storedOption(t, api, attemptID, questions[0].ID)
	assert.Equal(t, questions[0].Options[3].ID, stored)
	assert.Equal(t, 1, count(t, api, "SELECT count(*) FROM answers"))

	read := decode[attemptBody](t, api.Get(t, "/api/v1/attempts/"+attemptID.String()))
	require.Len(t, read.Answers, 1)
	assert.Equal(t, questions[0].Options[3].ID.String(), read.Answers[0].OptionID)
}

func TestAPI_SaveAnswer_Returns422ForAQuestionOutsideTheExamAndForAnOptionOfAnotherQuestion(t *testing.T) {
	api := testapi.Start(t)
	_, questions, attemptID := startedAttempt(t, api)
	outside := fixtures.SaveQuestion(t, api.Questions)

	requireProblem(t, putAnswer(t, api, attemptID, outside, outside.Options[0]), http.StatusUnprocessableEntity, "Answer is not valid")
	requireProblem(t, putAnswer(t, api, attemptID, questions[0], questions[1].Options[0]), http.StatusUnprocessableEntity, "Answer is not valid")
	requireProblem(t, api.Do(t, http.MethodPut, answerPath(attemptID, questions[0].ID), "application/json", []byte(`{"optionId":"`+uuid.NewString()+`"}`)),
		http.StatusUnprocessableEntity, "Answer is not valid")
	requireProblem(t, api.Do(t, http.MethodPut, answerPath(attemptID, questions[0].ID), "application/json", []byte(`{}`)),
		http.StatusUnprocessableEntity, "Answer is not valid")
	assert.Zero(t, count(t, api, "SELECT count(*) FROM answers"))
}

func TestAPI_SaveAnswer_Returns409AfterTheTimeLimitAndAfterTheSubmit(t *testing.T) {
	api := testapi.Start(t)
	now := time.Now()
	question := fixtures.SaveQuestion(t, api.Questions)
	exam := fixtures.SaveExam(t, api.Exams, api.Questions, fixtures.WithQuestionIDs(question.ID), fixtures.WithTimeLimit(30))
	expired := fixtures.SaveAttempt(t, api.Attempts, exam, fixtures.WithStartedAt(now.Add(-31*time.Minute)))
	submitted := fixtures.SaveAttempt(t, api.Attempts, exam, fixtures.WithStartedAt(now.Add(-5*time.Minute)), fixtures.WithSubmittedAt(now.Add(-time.Minute)))
	justInTime := fixtures.SaveAttempt(t, api.Attempts, exam, fixtures.WithStartedAt(now.Add(-29*time.Minute)))

	requireProblem(t, putAnswer(t, api, expired.ID, question, question.Options[0]), http.StatusConflict, "Attempt is closed")
	requireProblem(t, putAnswer(t, api, submitted.ID, question, question.Options[0]), http.StatusConflict, "Attempt is closed")
	assert.Zero(t, count(t, api, "SELECT count(*) FROM answers"))
	require.Equal(t, http.StatusNoContent, putAnswer(t, api, justInTime.ID, question, question.Options[0]).StatusCode)
}

func TestAPI_SaveAnswer_Returns404And400AndRejectsABodyThatIsNotValid(t *testing.T) {
	api := testapi.Start(t)
	_, questions, attemptID := startedAttempt(t, api)

	requireProblem(t, putAnswer(t, api, uuid.New(), questions[0], questions[0].Options[0]), http.StatusNotFound, "Attempt does not exist")
	requireProblem(t, api.Do(t, http.MethodPut, "/api/v1/attempts/not-a-uuid/answers/"+questions[0].ID.String(), "application/json", []byte(`{}`)),
		http.StatusBadRequest, "ID is not valid")
	requireProblem(t, api.Do(t, http.MethodPut, "/api/v1/attempts/"+attemptID.String()+"/answers/not-a-uuid", "application/json", []byte(`{}`)),
		http.StatusBadRequest, "ID is not valid")
	path := answerPath(attemptID, questions[0].ID)
	requireProblem(t, api.Do(t, http.MethodPut, path, "application/json", []byte(`{"optionId":`)), http.StatusBadRequest, "not valid JSON")
	requireProblem(t, api.Do(t, http.MethodPut, path, "application/json", []byte(`{"optionId":"not-a-uuid"}`)), http.StatusBadRequest, "not valid JSON")
	requireProblem(t, api.Do(t, http.MethodPut, path, "text/plain", []byte("x")), http.StatusUnsupportedMediaType, "content type")
}

func TestAPI_SubmitAttempt_Returns200WithTheScoreAndCountsSkippedAndWrongQuestionsAsZero(t *testing.T) {
	api := testapi.Start(t)
	_, questions, attemptID := startedAttempt(t, api)
	require.Equal(t, http.StatusNoContent, putAnswer(t, api, attemptID, questions[0], questions[0].Options[0]).StatusCode) // Correct.
	require.Equal(t, http.StatusNoContent, putAnswer(t, api, attemptID, questions[1], questions[1].Options[1]).StatusCode) // Wrong.
	// The third Question has no Answer.

	res := api.Do(t, http.MethodPost, "/api/v1/attempts/"+attemptID.String()+"/submission", "", nil)

	require.Equal(t, http.StatusOK, res.StatusCode, string(res.Body))
	assert.Equal(t, scoreBody{Correct: 1, Total: 3}, decode[scoreBody](t, res))
	var submittedAt time.Time
	require.NoError(t, api.DB.QueryRowContext(t.Context(), "SELECT submitted_at FROM attempts WHERE id = $1", attemptID).Scan(&submittedAt))
	assert.WithinDuration(t, time.Now(), submittedAt, time.Minute)
	assert.JSONEq(t, `{"id":"`+attemptID.String()+`","status":"submitted"}`, string(api.Get(t, "/api/v1/attempts/"+attemptID.String()).Body))
}

func TestAPI_SubmitAttempt_GivesZeroOfTheTotalForAnAttemptWithoutAnswers(t *testing.T) {
	api := testapi.Start(t)
	_, _, attemptID := startedAttempt(t, api)

	res := api.Do(t, http.MethodPost, "/api/v1/attempts/"+attemptID.String()+"/submission", "", nil)

	require.Equal(t, http.StatusOK, res.StatusCode, string(res.Body))
	assert.Equal(t, scoreBody{Correct: 0, Total: 3}, decode[scoreBody](t, res))
}

func TestAPI_SubmitAttempt_Returns409ForASecondSubmitAndKeepsTheFirstSubmitTime(t *testing.T) {
	api := testapi.Start(t)
	_, _, attemptID := startedAttempt(t, api)
	path := "/api/v1/attempts/" + attemptID.String() + "/submission"
	require.Equal(t, http.StatusOK, api.Do(t, http.MethodPost, path, "", nil).StatusCode)
	var first time.Time
	require.NoError(t, api.DB.QueryRowContext(t.Context(), "SELECT submitted_at FROM attempts WHERE id = $1", attemptID).Scan(&first))

	requireProblem(t, api.Do(t, http.MethodPost, path, "", nil), http.StatusConflict, "already submitted")

	var second time.Time
	require.NoError(t, api.DB.QueryRowContext(t.Context(), "SELECT submitted_at FROM attempts WHERE id = $1", attemptID).Scan(&second))
	assert.True(t, first.Equal(second))
}

func TestAPI_SubmitAttempt_WorksAfterTheTimeLimitUsesOnlyTheSavedAnswersAndMakesTheStatusTimedOut(t *testing.T) {
	api := testapi.Start(t)
	now := time.Now()
	q1, q2 := fixtures.SaveQuestion(t, api.Questions), fixtures.SaveQuestion(t, api.Questions)
	exam := fixtures.SaveExam(t, api.Exams, api.Questions, fixtures.WithQuestionIDs(q1.ID, q2.ID), fixtures.WithTimeLimit(30))
	attempt := fixtures.SaveAttempt(t, api.Attempts, exam, fixtures.WithStartedAt(now.Add(-time.Hour)))
	_, err := api.DB.ExecContext(t.Context(), "INSERT INTO answers (attempt_id, question_id, option_id) VALUES ($1, $2, $3)",
		attempt.ID, q1.ID, q1.Options[0].ID)
	require.NoError(t, err)

	res := api.Do(t, http.MethodPost, "/api/v1/attempts/"+attempt.ID.String()+"/submission", "", nil)

	require.Equal(t, http.StatusOK, res.StatusCode, string(res.Body))
	assert.Equal(t, scoreBody{Correct: 1, Total: 2}, decode[scoreBody](t, res))
	assert.JSONEq(t, `{"id":"`+attempt.ID.String()+`","status":"timed_out"}`, string(api.Get(t, "/api/v1/attempts/"+attempt.ID.String()).Body))
	requireProblem(t, api.Do(t, http.MethodPost, "/api/v1/attempts/"+attempt.ID.String()+"/submission", "", nil), http.StatusConflict, "already submitted")
}

func TestAPI_SubmitAttempt_Returns404And400(t *testing.T) {
	api := testapi.Start(t)

	requireProblem(t, api.Do(t, http.MethodPost, "/api/v1/attempts/"+uuid.NewString()+"/submission", "", nil), http.StatusNotFound, "Attempt does not exist")
	requireProblem(t, api.Do(t, http.MethodPost, "/api/v1/attempts/not-a-uuid/submission", "", nil), http.StatusBadRequest, "ID is not valid")
}

func TestAPI_SaveAnswer_NeverStoresAnAnswerAfterTheSubmitWhenBothRunAtTheSameTime(t *testing.T) {
	api := testapi.Start(t)
	_, questions, attemptID := startedAttempt(t, api)
	submitPath := "/api/v1/attempts/" + attemptID.String() + "/submission"

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			option := questions[i%3].Options[i%4]
			res := putAnswerRaw(api, attemptID, questions[i%3], option)
			assert.Contains(t, []int{http.StatusNoContent, http.StatusConflict}, res)
		}()
	}
	submit := api.Do(t, http.MethodPost, submitPath, "", nil)
	wg.Wait()

	require.Equal(t, http.StatusOK, submit.StatusCode, string(submit.Body))
	// Whatever the order was, the answers that exist now are the answers that the Score counted: a late save is rejected.
	var late int
	require.NoError(t, api.DB.QueryRowContext(t.Context(),
		"SELECT count(*) FROM answers a JOIN attempts t ON t.id = a.attempt_id WHERE a.saved_at > t.submitted_at").Scan(&late))
	assert.Zero(t, late, "No Answer is saved after the submit time.")
}

// putAnswerRaw sends the request without the test helpers that use t, so that a goroutine can call it.
func putAnswerRaw(api *testapi.API, attemptID uuid.UUID, question domain.Question, option domain.Option) int {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, api.URL+answerPath(attemptID, question.ID),
		strings.NewReader(answerJSON(option.ID)))
	if err != nil {
		return 0
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0
	}
	defer func() { _ = res.Body.Close() }()
	return res.StatusCode
}
