//go:build e2e

package exam_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudradcruze/go-react-template/backend/api-test/e2e/testapi"
	"github.com/rudradcruze/go-react-template/backend/api-test/fixtures"
	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
)

type candidateOptionBody struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type candidateQuestionBody struct {
	ID      string                `json:"id"`
	Text    string                `json:"text"`
	Options []candidateOptionBody `json:"options"`
}

type attemptBody struct {
	ID        string                  `json:"id"`
	Status    string                  `json:"status"`
	ExamTitle string                  `json:"examTitle"`
	ExpiresAt string                  `json:"expiresAt"`
	Questions []candidateQuestionBody `json:"questions"`
	Answers   []struct {
		QuestionID string `json:"questionId"`
		OptionID   string `json:"optionId"`
	} `json:"answers"`
}

const startBody = `{"name":"  Ann Smith ","emailAddress":" Ann@Example.COM "}`

func startPath(exam domain.Exam) string { return "/api/v1/exam-links/" + exam.LinkToken + "/attempts" }

// openExamWithQuestions saves an Exam with an open window, 30 minutes of Time limit and 2 Questions.
func openExamWithQuestions(t *testing.T, api *testapi.API) (domain.Exam, []domain.Question) {
	t.Helper()
	first, second := fixtures.SaveQuestion(t, api.Questions), fixtures.SaveQuestion(t, api.Questions)
	exam := fixtures.SaveExam(t, api.Exams, api.Questions,
		fixtures.WithQuestionIDs(second.ID, first.ID), fixtures.WithTimeLimit(30))
	return exam, []domain.Question{second, first}
}

func TestAPI_GetExamLink_GivesTheExamWithoutItsQuestionsAndNeedsNoSession(t *testing.T) {
	api := testapi.Start(t)
	exam, _ := openExamWithQuestions(t, api)

	res := api.Get(t, "/api/v1/exam-links/"+exam.LinkToken)

	require.Equal(t, http.StatusOK, res.StatusCode, string(res.Body))
	got := decode[map[string]any](t, res)
	assert.Equal(t, exam.Title, got["title"])
	assert.EqualValues(t, 30, got["timeLimitMinutes"])
	assert.EqualValues(t, 2, got["questionCount"])
	assert.ElementsMatch(t, []string{"title", "opensAt", "closesAt", "timeLimitMinutes", "questionCount"}, keys(got))
}

func keys(m map[string]any) []string {
	result := make([]string, 0, len(m))
	for key := range m {
		result = append(result, key)
	}
	return result
}

func TestAPI_GetExamLink_Returns404ForAnUnknownTokenAndForATokenThatARegenerationReplaced(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	exam, _ := openExamWithQuestions(t, api)

	requireProblem(t, api.Get(t, "/api/v1/exam-links/not-a-token"), http.StatusNotFound, "Exam link is not valid")
	regenerated := decode[examBody](t, api.JSONAs(t, token, http.MethodPost, "/api/v1/admin/exams/"+exam.ID.String()+"/exam-link", ""))

	requireProblem(t, api.Get(t, "/api/v1/exam-links/"+exam.LinkToken), http.StatusNotFound, "Exam link is not valid")
	require.Equal(t, http.StatusOK, api.Get(t, "/api/v1/exam-links/"+regenerated.LinkToken).StatusCode)
}

func TestAPI_StartAttempt_Returns201WithTheQuestionsInOrderAndNoCorrectOption(t *testing.T) {
	api := testapi.Start(t)
	exam, questions := openExamWithQuestions(t, api)

	res := api.PostJSON(t, startPath(exam), startBody)

	require.Equal(t, http.StatusCreated, res.StatusCode, string(res.Body))
	assert.NotContains(t, string(res.Body), "correct", "A response to a Candidate must not say which Option is correct.")
	got := decode[attemptBody](t, res)
	id, err := uuid.Parse(got.ID)
	require.NoError(t, err)
	assert.Equal(t, uuid.Version(4), id.Version())
	assert.Equal(t, "open", got.Status)
	assert.Equal(t, exam.Title, got.ExamTitle)
	expiresAt, err := time.Parse(time.RFC3339Nano, got.ExpiresAt)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(30*time.Minute), expiresAt, time.Minute)
	require.Len(t, got.Questions, 2)
	for i, want := range questions {
		assert.Equal(t, want.ID.String(), got.Questions[i].ID)
		assert.Equal(t, want.Text, got.Questions[i].Text)
		require.Len(t, got.Questions[i].Options, len(want.Options))
		for j, option := range want.Options {
			assert.Equal(t, option.ID.String(), got.Questions[i].Options[j].ID)
			assert.Equal(t, option.Text, got.Questions[i].Options[j].Text)
		}
	}
	assert.NotNil(t, decode[map[string]any](t, res)["answers"], "The answers are an empty array, not missing.")
	assert.Empty(t, got.Answers)

	var name, email string
	require.NoError(t, api.DB.QueryRowContext(t.Context(),
		"SELECT candidate_name, email_address FROM attempts WHERE id = $1", id).Scan(&name, &email))
	assert.Equal(t, "Ann Smith", name)
	assert.Equal(t, "ann@example.com", email)
}

func TestAPI_StartAttempt_Returns200WithTheSameOpenAttemptForTheSameEmailAddressInAnyLetterCase(t *testing.T) {
	api := testapi.Start(t)
	exam, _ := openExamWithQuestions(t, api)
	first := decode[attemptBody](t, api.PostJSON(t, startPath(exam), startBody))

	res := api.PostJSON(t, startPath(exam), `{"name":"Another name","emailAddress":"ANN@example.com"}`)

	require.Equal(t, http.StatusOK, res.StatusCode, string(res.Body))
	assert.Equal(t, first.ID, decode[attemptBody](t, res).ID)
	assert.Equal(t, 1, count(t, api, "SELECT count(*) FROM attempts"))
}

func TestAPI_StartAttempt_AllowsTheSameEmailAddressForAnotherExam(t *testing.T) {
	api := testapi.Start(t)
	first, _ := openExamWithQuestions(t, api)
	second, _ := openExamWithQuestions(t, api)

	require.Equal(t, http.StatusCreated, api.PostJSON(t, startPath(first), startBody).StatusCode)
	require.Equal(t, http.StatusCreated, api.PostJSON(t, startPath(second), startBody).StatusCode)
}

func TestAPI_StartAttempt_Returns409OutsideTheExamWindowForANewAttempt(t *testing.T) {
	now := time.Now()
	tests := map[string]struct {
		opens, closes time.Time
		detailPart    string
	}{
		"before the open time": {now.Add(time.Hour), now.Add(2 * time.Hour), "not open yet"},
		"after the close time": {now.Add(-2 * time.Hour), now.Add(-time.Hour), "closed"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			api := testapi.Start(t)
			exam := fixtures.SaveExam(t, api.Exams, api.Questions, fixtures.WithExamWindow(tt.opens, tt.closes))

			res := api.PostJSON(t, startPath(exam), startBody)

			requireProblem(t, res, http.StatusConflict, tt.detailPart)
			assert.Zero(t, count(t, api, "SELECT count(*) FROM attempts"))
		})
	}
}

func TestAPI_StartAttempt_GivesTheOpenAttemptAfterTheExamWindowClosed(t *testing.T) {
	api := testapi.Start(t)
	now := time.Now()
	exam := fixtures.SaveExam(t, api.Exams, api.Questions, fixtures.WithExamWindow(now.Add(-2*time.Hour), now.Add(-time.Minute)), fixtures.WithTimeLimit(60))
	open := fixtures.SaveAttempt(t, api.Attempts, exam, fixtures.WithCandidateEmail("ann@example.com"), fixtures.WithStartedAt(now.Add(-10*time.Minute)))

	res := api.PostJSON(t, startPath(exam), startBody)

	require.Equal(t, http.StatusOK, res.StatusCode, string(res.Body))
	assert.Equal(t, open.ID.String(), decode[attemptBody](t, res).ID)
}

func TestAPI_StartAttempt_Returns409WithoutAScoreForASubmittedOrATimedOutAttempt(t *testing.T) {
	api := testapi.Start(t)
	now := time.Now()
	exam := fixtures.SaveExam(t, api.Exams, api.Questions, fixtures.WithTimeLimit(30))
	fixtures.SaveAttempt(t, api.Attempts, exam, fixtures.WithCandidateEmail("ann@example.com"),
		fixtures.WithStartedAt(now.Add(-10*time.Minute)), fixtures.WithSubmittedAt(now.Add(-5*time.Minute)))
	fixtures.SaveAttempt(t, api.Attempts, exam, fixtures.WithCandidateEmail("bob@example.com"), fixtures.WithStartedAt(now.Add(-2*time.Hour)))

	for _, email := range []string{"ann@example.com", "bob@example.com"} {
		res := api.PostJSON(t, startPath(exam), `{"name":"X","emailAddress":"`+email+`"}`)

		requireProblem(t, res, http.StatusConflict, "already submitted")
		assert.NotContains(t, strings.ToLower(string(res.Body)), "score")
	}
}

func TestAPI_StartAttempt_Returns422WhenARuleRejectsTheInput(t *testing.T) {
	api := testapi.Start(t)
	exam, _ := openExamWithQuestions(t, api)
	tests := map[string]struct {
		body       string
		detailPart string
	}{
		"name is empty":              {`{"name":"  ","emailAddress":"ann@example.com"}`, "name is empty"},
		"name is missing":            {`{"emailAddress":"ann@example.com"}`, "name is empty"},
		"name is too long":           {`{"name":"` + strings.Repeat("a", 101) + `","emailAddress":"ann@example.com"}`, "name is too long"},
		"name has a new line":        {`{"name":"a\nb","emailAddress":"ann@example.com"}`, "character that is not allowed"},
		"email address is not valid": {`{"name":"Ann","emailAddress":"ann"}`, "email address is not valid"},
		"email address is missing":   {`{"name":"Ann"}`, "email address is not valid"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			requireProblem(t, api.PostJSON(t, startPath(exam), tt.body), http.StatusUnprocessableEntity, tt.detailPart)
		})
	}
	assert.Zero(t, count(t, api, "SELECT count(*) FROM attempts"))
}

func TestAPI_StartAttempt_Returns404ForAnUnknownTokenAndRejectsABodyThatIsNotValid(t *testing.T) {
	api := testapi.Start(t)
	exam, _ := openExamWithQuestions(t, api)

	requireProblem(t, api.PostJSON(t, "/api/v1/exam-links/unknown/attempts", startBody), http.StatusNotFound, "Exam link is not valid")
	requireProblem(t, api.PostJSON(t, startPath(exam), `{"name":`), http.StatusBadRequest, "not valid JSON")
	requireProblem(t, api.PostJSON(t, startPath(exam), `{"name":"A","emailAddress":"a@example.com","x":1}`), http.StatusBadRequest, "not valid JSON")
	requireProblem(t, api.Do(t, http.MethodPost, startPath(exam), "text/plain", []byte("x")), http.StatusUnsupportedMediaType, "content type")
	requireProblem(t, api.PostJSON(t, startPath(exam), `{"name":"`+strings.Repeat("a", 1_500_000)+`"}`), http.StatusRequestEntityTooLarge, "larger than 1 MiB")
}

func TestAPI_GetAttempt_GivesAnOpenAttemptWithItsSavedAnswers(t *testing.T) {
	api := testapi.Start(t)
	exam, questions := openExamWithQuestions(t, api)
	started := decode[attemptBody](t, api.PostJSON(t, startPath(exam), startBody))
	_, err := api.DB.ExecContext(t.Context(), "INSERT INTO answers (attempt_id, question_id, option_id) VALUES ($1, $2, $3)",
		started.ID, questions[1].ID, questions[1].Options[2].ID)
	require.NoError(t, err)

	res := api.Get(t, "/api/v1/attempts/"+started.ID)

	require.Equal(t, http.StatusOK, res.StatusCode, string(res.Body))
	assert.NotContains(t, string(res.Body), "correct")
	got := decode[attemptBody](t, res)
	assert.Equal(t, "open", got.Status)
	require.Len(t, got.Questions, 2)
	require.Len(t, got.Answers, 1)
	assert.Equal(t, questions[1].ID.String(), got.Answers[0].QuestionID)
	assert.Equal(t, questions[1].Options[2].ID.String(), got.Answers[0].OptionID)
}

func TestAPI_GetAttempt_GivesOnlyTheStatusForASubmittedOrATimedOutAttempt(t *testing.T) {
	api := testapi.Start(t)
	now := time.Now()
	exam := fixtures.SaveExam(t, api.Exams, api.Questions, fixtures.WithTimeLimit(30))
	submitted := fixtures.SaveAttempt(t, api.Attempts, exam, fixtures.WithStartedAt(now.Add(-10*time.Minute)), fixtures.WithSubmittedAt(now.Add(-5*time.Minute)))
	timedOut := fixtures.SaveAttempt(t, api.Attempts, exam, fixtures.WithStartedAt(now.Add(-2*time.Hour)))

	for id, status := range map[uuid.UUID]string{submitted.ID: "submitted", timedOut.ID: "timed_out"} {
		res := api.Get(t, "/api/v1/attempts/"+id.String())

		require.Equal(t, http.StatusOK, res.StatusCode, string(res.Body))
		assert.JSONEq(t, `{"id":"`+id.String()+`","status":"`+status+`"}`, string(res.Body))
	}
}

func TestAPI_GetAttempt_Returns404ForAnUnknownIDAnd400ForAnIDThatIsNotValid(t *testing.T) {
	api := testapi.Start(t)

	requireProblem(t, api.Get(t, "/api/v1/attempts/"+uuid.NewString()), http.StatusNotFound, "Attempt does not exist")
	requireProblem(t, api.Get(t, "/api/v1/attempts/not-a-uuid"), http.StatusBadRequest, "ID is not valid")
}

func TestAPI_ExamAndQuestionRules_LockAfterTheFirstAttempt(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	exam, questions := openExamWithQuestions(t, api)
	require.Equal(t, http.StatusCreated, api.PostJSON(t, startPath(exam), startBody).StatusCode)
	examPath := "/api/v1/admin/exams/" + exam.ID.String()
	ids := []uuid.UUID{questions[0].ID, questions[1].ID}
	sameExceptClose := func(closes string) string {
		return `{"title":"` + exam.Title + `","questionIds":["` + ids[0].String() + `","` + ids[1].String() +
			`"],"opensAt":"` + exam.OpensAt.Format(time.RFC3339Nano) + `","closesAt":"` + closes + `","timeLimitMinutes":30}`
	}

	extended := api.JSONAs(t, token, http.MethodPut, examPath, sameExceptClose(exam.ClosesAt.Add(time.Hour).Format(time.RFC3339Nano)))
	require.Equal(t, http.StatusOK, extended.StatusCode, string(extended.Body))
	assert.Equal(t, exam.LinkToken, decode[examBody](t, extended).LinkToken)

	requireProblem(t, api.JSONAs(t, token, http.MethodPut, examPath, examJSON("Another title", ids, 30)), http.StatusConflict, "Only the close time can change")
	requireProblem(t, api.JSONAs(t, token, http.MethodPut, examPath, examJSON(exam.Title, ids[:1], 30)), http.StatusConflict, "Only the close time can change")
	requireProblem(t, api.JSONAs(t, token, http.MethodDelete, examPath, ""), http.StatusConflict, "cannot delete the Exam")
	assert.Equal(t, 1, count(t, api, "SELECT count(*) FROM exams"))

	requireProblem(t, api.JSONAs(t, token, http.MethodPut, "/api/v1/admin/questions/"+ids[0].String(), validQuestion),
		http.StatusConflict, "You cannot change the Question")
	requireProblem(t, api.JSONAs(t, token, http.MethodDelete, "/api/v1/admin/questions/"+ids[0].String(), ""),
		http.StatusConflict, "An Exam uses the Question")

	regenerated := api.JSONAs(t, token, http.MethodPost, examPath+"/exam-link", "")
	require.Equal(t, http.StatusOK, regenerated.StatusCode, "An Admin can regenerate the Exam link after the first Attempt.")
}

func TestAPI_UpdateQuestion_StillWorksWhenOnlyAnExamWithoutAttemptsUsesTheQuestion(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	exam := fixtures.SaveExam(t, api.Exams, api.Questions)

	res := api.JSONAs(t, token, http.MethodPut, "/api/v1/admin/questions/"+exam.QuestionIDs[0].String(), validQuestion)

	require.Equal(t, http.StatusOK, res.StatusCode, string(res.Body))
}

func TestAPI_GetExamLink_Returns500WithAGeneralTextWhenTheDatabaseFails(t *testing.T) {
	api := testapi.Start(t)
	api.BreakDatabase(t)

	res := api.Get(t, "/api/v1/exam-links/anything")

	requireProblem(t, res, http.StatusInternalServerError, "cannot complete the request")
}
