//go:build e2e

package exam_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudradcruze/go-react-template/backend/api-test/e2e/testapi"
	"github.com/rudradcruze/go-react-template/backend/api-test/fixtures"
)

type examBody struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	QuestionIDs      []string `json:"questionIds"`
	OpensAt          string   `json:"opensAt"`
	ClosesAt         string   `json:"closesAt"`
	TimeLimitMinutes int      `json:"timeLimitMinutes"`
	LinkToken        string   `json:"linkToken"`
	CreatedAt        string   `json:"createdAt"`
}

type examSummaryBody struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	OpensAt          string `json:"opensAt"`
	ClosesAt         string `json:"closesAt"`
	TimeLimitMinutes int    `json:"timeLimitMinutes"`
	QuestionCount    int    `json:"questionCount"`
}

func examJSON(title string, questionIDs []uuid.UUID, timeLimit int) string {
	ids := make([]string, len(questionIDs))
	for i, id := range questionIDs {
		ids[i] = `"` + id.String() + `"`
	}
	return fmt.Sprintf(`{"title":%q,"questionIds":[%s],"opensAt":"2026-11-01T09:00:00+06:00","closesAt":"2026-11-01T17:00:00Z","timeLimitMinutes":%d}`,
		title, strings.Join(ids, ","), timeLimit)
}

func TestAPI_Exams_Return401WithoutAnAdminSession(t *testing.T) {
	api := testapi.Start(t)
	id := "0198f2a0-0000-7000-8000-000000000001"
	routes := []struct{ method, path string }{
		{http.MethodPost, "/api/v1/admin/exams"},
		{http.MethodGet, "/api/v1/admin/exams"},
		{http.MethodGet, "/api/v1/admin/exams/" + id},
		{http.MethodPut, "/api/v1/admin/exams/" + id},
		{http.MethodDelete, "/api/v1/admin/exams/" + id},
		{http.MethodPost, "/api/v1/admin/exams/" + id + "/exam-link"},
	}
	for _, route := range routes {
		res := api.Do(t, route.method, route.path, "application/json", []byte("{}"))
		requireProblem(t, res, http.StatusUnauthorized, "not logged in")
	}
}

func TestAPI_CreateExam_Returns201AndSavesTheExamWithItsQuestionsInOrderAndALinkToken(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	first, second, third := fixtures.SaveQuestion(t, api.Questions), fixtures.SaveQuestion(t, api.Questions), fixtures.SaveQuestion(t, api.Questions)
	order := []uuid.UUID{third.ID, first.ID, second.ID}

	res := api.JSONAs(t, token, http.MethodPost, "/api/v1/admin/exams", examJSON("  Go basics ", order, 45))

	require.Equal(t, http.StatusCreated, res.StatusCode, string(res.Body))
	got := decode[examBody](t, res)
	id, err := uuid.Parse(got.ID)
	require.NoError(t, err)
	assert.Equal(t, uuid.Version(7), id.Version())
	assert.Equal(t, "Go basics", got.Title)
	assert.Equal(t, []string{third.ID.String(), first.ID.String(), second.ID.String()}, got.QuestionIDs)
	assert.Equal(t, 45, got.TimeLimitMinutes)
	assert.Regexp(t, `^[0-9A-Za-z]{22}$`, got.LinkToken)
	opens, err := time.Parse(time.RFC3339Nano, got.OpensAt)
	require.NoError(t, err)
	assert.True(t, time.Date(2026, 11, 1, 3, 0, 0, 0, time.UTC).Equal(opens), "The API returns the open time in UTC.")
	assert.True(t, strings.HasSuffix(got.OpensAt, "Z"))
	createdAt, err := time.Parse(time.RFC3339Nano, got.CreatedAt)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), createdAt, time.Minute)

	read := decode[examBody](t, api.GetAs(t, token, "/api/v1/admin/exams/"+got.ID))
	assert.Equal(t, got, read)
}

func TestAPI_CreateExam_Returns422WhenARuleRejectsTheInput(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	question := fixtures.SaveQuestion(t, api.Questions)
	one := []uuid.UUID{question.ID}
	tests := map[string]struct {
		body       string
		detailPart string
	}{
		"title is empty":          {examJSON("  ", one, 30), "title of the Exam is empty"},
		"title is too long":       {examJSON(strings.Repeat("a", 201), one, 30), "title of the Exam is too long"},
		"no Question":             {examJSON("T", nil, 30), "too few Questions"},
		"the same Question twice": {examJSON("T", []uuid.UUID{question.ID, question.ID}, 30), "same Question 2 times"},
		"a Question that is gone": {examJSON("T", []uuid.UUID{uuid.New()}, 30), "Question of the Exam does not exist"},
		"Time limit of 0":         {examJSON("T", one, 0), "Time limit is not valid"},
		"Time limit of 481":       {examJSON("T", one, 481), "Time limit is not valid"},
		"close time before open":  {`{"title":"T","questionIds":["` + question.ID.String() + `"],"opensAt":"2026-11-02T00:00:00Z","closesAt":"2026-11-01T00:00:00Z","timeLimitMinutes":30}`, "Exam window is not valid"},
		"times are missing":       {`{"title":"T","questionIds":["` + question.ID.String() + `"],"timeLimitMinutes":30}`, "Exam window is not valid"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			res := api.JSONAs(t, token, http.MethodPost, "/api/v1/admin/exams", tt.body)

			requireProblem(t, res, http.StatusUnprocessableEntity, tt.detailPart)
			assert.Zero(t, count(t, api, "SELECT count(*) FROM exams"))
			assert.Zero(t, count(t, api, "SELECT count(*) FROM exam_questions"))
		})
	}
}

func TestAPI_CreateExam_RejectsABodyThatIsNotValid(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)

	requireProblem(t, api.JSONAs(t, token, http.MethodPost, "/api/v1/admin/exams", `{"title":`), http.StatusBadRequest, "not valid JSON")
	requireProblem(t, api.JSONAs(t, token, http.MethodPost, "/api/v1/admin/exams", `{"title":"T","questionIds":["not-a-uuid"]}`),
		http.StatusBadRequest, "not valid JSON")
	requireProblem(t, api.JSONAs(t, token, http.MethodPost, "/api/v1/admin/exams", `{"extra":1}`), http.StatusBadRequest, "not valid JSON")
	requireProblem(t, api.DoWithSession(t, token, http.MethodPost, "/api/v1/admin/exams", "text/plain", []byte("x")),
		http.StatusUnsupportedMediaType, "content type")
}

func TestAPI_ListExams_GivesAllExamsNewestFirstWithTheNumberOfQuestions(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	q1, q2 := fixtures.SaveQuestion(t, api.Questions), fixtures.SaveQuestion(t, api.Questions)
	older := fixtures.SaveExam(t, api.Exams, api.Questions, fixtures.WithQuestionIDs(q1.ID))
	time.Sleep(5 * time.Millisecond)
	newer := fixtures.SaveExam(t, api.Exams, api.Questions, fixtures.WithQuestionIDs(q1.ID, q2.ID))

	res := api.GetAs(t, token, "/api/v1/admin/exams")

	require.Equal(t, http.StatusOK, res.StatusCode, string(res.Body))
	got := decode[[]examSummaryBody](t, res)
	require.Len(t, got, 2)
	assert.Equal(t, newer.ID.String(), got[0].ID)
	assert.Equal(t, 2, got[0].QuestionCount)
	assert.Equal(t, older.ID.String(), got[1].ID)
	assert.Equal(t, 1, got[1].QuestionCount)
	assert.NotContains(t, string(res.Body), "linkToken")
}

func TestAPI_ListExams_GivesAnEmptyArrayWhenThereIsNoExam(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)

	res := api.GetAs(t, token, "/api/v1/admin/exams")

	require.Equal(t, http.StatusOK, res.StatusCode, string(res.Body))
	assert.JSONEq(t, `[]`, string(res.Body))
}

func TestAPI_GetExam_Returns404And400(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)

	requireProblem(t, api.GetAs(t, token, "/api/v1/admin/exams/0198f2a0-0000-7000-8000-000000000000"), http.StatusNotFound, "Exam does not exist")
	requireProblem(t, api.GetAs(t, token, "/api/v1/admin/exams/not-a-uuid"), http.StatusBadRequest, "ID is not valid")
}

func TestAPI_UpdateExam_ReplacesAllValuesAndKeepsTheIDTheLinkTokenAndTheCreationTime(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	exam := fixtures.SaveExam(t, api.Exams, api.Questions)
	other := fixtures.SaveQuestion(t, api.Questions)

	res := api.JSONAs(t, token, http.MethodPut, "/api/v1/admin/exams/"+exam.ID.String(), examJSON("New title", []uuid.UUID{other.ID}, 90))

	require.Equal(t, http.StatusOK, res.StatusCode, string(res.Body))
	got := decode[examBody](t, res)
	assert.Equal(t, exam.ID.String(), got.ID)
	assert.Equal(t, exam.LinkToken, got.LinkToken)
	assert.Equal(t, "New title", got.Title)
	assert.Equal(t, []string{other.ID.String()}, got.QuestionIDs)
	assert.Equal(t, 90, got.TimeLimitMinutes)
	createdAt, err := time.Parse(time.RFC3339Nano, got.CreatedAt)
	require.NoError(t, err)
	assert.True(t, exam.CreatedAt.Equal(createdAt))
	assert.Equal(t, got, decode[examBody](t, api.GetAs(t, token, "/api/v1/admin/exams/"+exam.ID.String())))
	assert.Equal(t, 1, count(t, api, "SELECT count(*) FROM exam_questions WHERE exam_id = $1", exam.ID))
}

func TestAPI_UpdateExam_Returns422AndKeepsTheOldDataForAnInputThatIsNotValidOrAQuestionThatIsGone(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	exam := fixtures.SaveExam(t, api.Exams, api.Questions)
	path := "/api/v1/admin/exams/" + exam.ID.String()

	requireProblem(t, api.JSONAs(t, token, http.MethodPut, path, examJSON("T", nil, 30)), http.StatusUnprocessableEntity, "too few Questions")
	requireProblem(t, api.JSONAs(t, token, http.MethodPut, path, examJSON("T", []uuid.UUID{uuid.New()}, 30)),
		http.StatusUnprocessableEntity, "Question of the Exam does not exist")

	read := decode[examBody](t, api.GetAs(t, token, path))
	assert.Equal(t, exam.Title, read.Title)
	assert.Equal(t, []string{exam.QuestionIDs[0].String()}, read.QuestionIDs)
}

func TestAPI_UpdateExam_Returns404And400(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	question := fixtures.SaveQuestion(t, api.Questions)
	body := examJSON("T", []uuid.UUID{question.ID}, 30)

	requireProblem(t, api.JSONAs(t, token, http.MethodPut, "/api/v1/admin/exams/0198f2a0-0000-7000-8000-000000000000", body),
		http.StatusNotFound, "Exam does not exist")
	requireProblem(t, api.JSONAs(t, token, http.MethodPut, "/api/v1/admin/exams/not-a-uuid", body), http.StatusBadRequest, "ID is not valid")
}

func TestAPI_RegenerateExamLink_MakesANewTokenAndChangesNothingElse(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	exam := fixtures.SaveExam(t, api.Exams, api.Questions)

	res := api.JSONAs(t, token, http.MethodPost, "/api/v1/admin/exams/"+exam.ID.String()+"/exam-link", "")

	require.Equal(t, http.StatusOK, res.StatusCode, string(res.Body))
	got := decode[examBody](t, res)
	assert.NotEqual(t, exam.LinkToken, got.LinkToken)
	assert.Regexp(t, `^[0-9A-Za-z]{22}$`, got.LinkToken)
	assert.Equal(t, exam.Title, got.Title)
	assert.Equal(t, got.LinkToken, decode[examBody](t, api.GetAs(t, token, "/api/v1/admin/exams/"+exam.ID.String())).LinkToken)
	assert.Zero(t, count(t, api, "SELECT count(*) FROM exams WHERE link_token = $1", exam.LinkToken))

	requireProblem(t, api.JSONAs(t, token, http.MethodPost, "/api/v1/admin/exams/0198f2a0-0000-7000-8000-000000000000/exam-link", ""),
		http.StatusNotFound, "Exam does not exist")
	requireProblem(t, api.JSONAs(t, token, http.MethodPost, "/api/v1/admin/exams/not-a-uuid/exam-link", ""), http.StatusBadRequest, "ID is not valid")
}

func TestAPI_DeleteExam_Returns204AndRemovesTheExamButNotItsQuestions(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	exam := fixtures.SaveExam(t, api.Exams, api.Questions)

	res := api.JSONAs(t, token, http.MethodDelete, "/api/v1/admin/exams/"+exam.ID.String(), "")

	require.Equal(t, http.StatusNoContent, res.StatusCode, string(res.Body))
	assert.Zero(t, count(t, api, "SELECT count(*) FROM exams"))
	assert.Zero(t, count(t, api, "SELECT count(*) FROM exam_questions"))
	assert.Equal(t, 1, count(t, api, "SELECT count(*) FROM questions"))
	requireProblem(t, api.JSONAs(t, token, http.MethodDelete, "/api/v1/admin/exams/"+exam.ID.String(), ""), http.StatusNotFound, "Exam does not exist")
	requireProblem(t, api.JSONAs(t, token, http.MethodDelete, "/api/v1/admin/exams/not-a-uuid", ""), http.StatusBadRequest, "ID is not valid")
}

func TestAPI_DeleteQuestion_Returns409WhileAnExamUsesTheQuestionAndWorksAfterTheExamIsGone(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	exam := fixtures.SaveExam(t, api.Exams, api.Questions)
	questionPath := "/api/v1/admin/questions/" + exam.QuestionIDs[0].String()

	requireProblem(t, api.JSONAs(t, token, http.MethodDelete, questionPath, ""), http.StatusConflict, "An Exam uses the Question")
	assert.Equal(t, 1, count(t, api, "SELECT count(*) FROM questions"))

	require.Equal(t, http.StatusNoContent, api.JSONAs(t, token, http.MethodDelete, "/api/v1/admin/exams/"+exam.ID.String(), "").StatusCode)
	require.Equal(t, http.StatusNoContent, api.JSONAs(t, token, http.MethodDelete, questionPath, "").StatusCode)
}

func TestDatabase_Exams_RejectAnExamWhoseWindowOrTimeLimitBreaksTheRule(t *testing.T) {
	api := testapi.Start(t)

	_, windowErr := api.DB.ExecContext(t.Context(), `INSERT INTO exams (id, title, opens_at, closes_at, time_limit_minutes, link_token, created_at)
		VALUES (gen_random_uuid(), 'T', now(), now(), 30, '0123456789012345678901', now())`)
	_, limitErr := api.DB.ExecContext(t.Context(), `INSERT INTO exams (id, title, opens_at, closes_at, time_limit_minutes, link_token, created_at)
		VALUES (gen_random_uuid(), 'T', now(), now() + interval '1 hour', 481, '0123456789012345678902', now())`)

	require.Error(t, windowErr)
	require.Error(t, limitErr)
}
