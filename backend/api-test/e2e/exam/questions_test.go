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
)

type optionBody struct {
	ID      string `json:"id"`
	Text    string `json:"text"`
	Correct bool   `json:"correct"`
}

type questionBody struct {
	ID        string       `json:"id"`
	Text      string       `json:"text"`
	Options   []optionBody `json:"options"`
	CreatedAt string       `json:"createdAt"`
}

const validQuestion = `{"text":"What is 2 + 2?","options":[{"text":"3","correct":false},{"text":"4","correct":true},{"text":"5","correct":false}]}`

func TestAPI_Questions_Return401WithoutAnAdminSession(t *testing.T) {
	api := testapi.Start(t)
	id := "0198f2a0-0000-7000-8000-000000000001"
	routes := []struct{ method, path string }{
		{http.MethodPost, "/api/v1/admin/questions"},
		{http.MethodGet, "/api/v1/admin/questions"},
		{http.MethodGet, "/api/v1/admin/questions/" + id},
		{http.MethodPut, "/api/v1/admin/questions/" + id},
		{http.MethodDelete, "/api/v1/admin/questions/" + id},
	}
	for _, route := range routes {
		res := api.Do(t, route.method, route.path, "application/json", []byte(validQuestion))
		requireProblem(t, res, http.StatusUnauthorized, "not logged in")
	}
}

func TestAPI_CreateQuestion_Returns201AndSavesTheQuestionWithItsOptionsInOrder(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)

	res := api.JSONAs(t, token, http.MethodPost, "/api/v1/admin/questions",
		`{"text":"  What is 2 + 2? ","options":[{"text":" 3 ","correct":false},{"text":"4","correct":true},{"text":"5","correct":false}]}`)

	require.Equal(t, http.StatusCreated, res.StatusCode, string(res.Body))
	got := decode[questionBody](t, res)
	id, err := uuid.Parse(got.ID)
	require.NoError(t, err)
	assert.Equal(t, uuid.Version(7), id.Version())
	assert.Equal(t, "What is 2 + 2?", got.Text)
	require.Len(t, got.Options, 3)
	for i, want := range []struct {
		text    string
		correct bool
	}{{"3", false}, {"4", true}, {"5", false}} {
		assert.Equal(t, want.text, got.Options[i].Text)
		assert.Equal(t, want.correct, got.Options[i].Correct)
		optionID, err := uuid.Parse(got.Options[i].ID)
		require.NoError(t, err)
		assert.Equal(t, uuid.Version(7), optionID.Version())
	}
	createdAt, err := time.Parse(time.RFC3339Nano, got.CreatedAt)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), createdAt, time.Minute)

	rows, err := api.DB.QueryContext(t.Context(), "SELECT text, correct FROM options WHERE question_id = $1 ORDER BY position", id)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var stored []string
	for rows.Next() {
		var text string
		var correct bool
		require.NoError(t, rows.Scan(&text, &correct))
		stored = append(stored, text+"/"+map[bool]string{true: "correct", false: "wrong"}[correct])
	}
	assert.Equal(t, []string{"3/wrong", "4/correct", "5/wrong"}, stored)
}

func TestAPI_CreateQuestion_Returns422WhenARuleRejectsTheInput(t *testing.T) {
	opt := func(text string, correct bool) string {
		return `{"text":"` + text + `","correct":` + map[bool]string{true: "true", false: "false"}[correct] + `}`
	}
	tests := map[string]struct {
		body       string
		detailPart string
	}{
		"text is empty":           {`{"text":"  ","options":[` + opt("A", true) + `,` + opt("B", false) + `]}`, "text of the Question is empty"},
		"text is missing":         {`{"options":[` + opt("A", true) + `,` + opt("B", false) + `]}`, "text of the Question is empty"},
		"text is too long":        {`{"text":"` + strings.Repeat("a", 2001) + `","options":[` + opt("A", true) + `,` + opt("B", false) + `]}`, "too long"},
		"text has a NUL":          {`{"text":"a\u0000b","options":[` + opt("A", true) + `,` + opt("B", false) + `]}`, "character that is not allowed"},
		"options are missing":     {`{"text":"Q"}`, "too few Options"},
		"1 option":                {`{"text":"Q","options":[` + opt("A", true) + `]}`, "too few Options"},
		"7 options":               {`{"text":"Q","options":[` + strings.Join([]string{opt("A", true), opt("B", false), opt("C", false), opt("D", false), opt("E", false), opt("F", false), opt("G", false)}, ",") + `]}`, "too many Options"},
		"an option text is empty": {`{"text":"Q","options":[` + opt("A", true) + `,` + opt(" ", false) + `]}`, "text of an Option is empty"},
		"an option is too long":   {`{"text":"Q","options":[` + opt("A", true) + `,` + opt(strings.Repeat("b", 501), false) + `]}`, "text of an Option is too long"},
		"no correct option":       {`{"text":"Q","options":[` + opt("A", false) + `,` + opt("B", false) + `]}`, "exactly one correct Option"},
		"2 correct options":       {`{"text":"Q","options":[` + opt("A", true) + `,` + opt("B", true) + `]}`, "exactly one correct Option"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			api := testapi.Start(t)
			token := api.LogIn(t)

			res := api.JSONAs(t, token, http.MethodPost, "/api/v1/admin/questions", tt.body)

			requireProblem(t, res, http.StatusUnprocessableEntity, tt.detailPart)
			assert.Zero(t, count(t, api, "SELECT count(*) FROM questions"))
		})
	}
}

func TestAPI_CreateQuestion_RejectsABodyThatIsNotValid(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)

	requireProblem(t, api.JSONAs(t, token, http.MethodPost, "/api/v1/admin/questions", `{"text":`), http.StatusBadRequest, "not valid JSON")
	requireProblem(t, api.JSONAs(t, token, http.MethodPost, "/api/v1/admin/questions", `{"text":"Q","options":[],"extra":1}`),
		http.StatusBadRequest, "not valid JSON")
	requireProblem(t, api.DoWithSession(t, token, http.MethodPost, "/api/v1/admin/questions", "text/plain", []byte("x")),
		http.StatusUnsupportedMediaType, "content type")
	requireProblem(t, api.JSONAs(t, token, http.MethodPost, "/api/v1/admin/questions", `{"text":"`+strings.Repeat("a", 1_500_000)+`"}`),
		http.StatusRequestEntityTooLarge, "larger than 1 MiB")
}

func TestDatabase_Options_RejectASecondCorrectOptionOfAQuestion(t *testing.T) {
	api := testapi.Start(t)
	question := fixtures.SaveQuestion(t, api.Questions)

	_, err := api.DB.ExecContext(t.Context(),
		"INSERT INTO options (id, question_id, position, text, correct) VALUES (gen_random_uuid(), $1, 5, 'x', true)", question.ID)

	require.Error(t, err)
}

func TestAPI_ListQuestions_GivesAllQuestionsNewestFirstWithTheirOptions(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	now := time.Now()
	oldest := fixtures.SaveQuestion(t, api.Questions, fixtures.WithQuestionCreatedAt(now.Add(-2*time.Hour)))
	newest := fixtures.SaveQuestion(t, api.Questions, fixtures.WithQuestionCreatedAt(now))
	middle := fixtures.SaveQuestion(t, api.Questions, fixtures.WithQuestionCreatedAt(now.Add(-time.Hour)))

	res := api.GetAs(t, token, "/api/v1/admin/questions")

	require.Equal(t, http.StatusOK, res.StatusCode, string(res.Body))
	got := decode[[]questionBody](t, res)
	require.Len(t, got, 3)
	assert.Equal(t, []string{newest.ID.String(), middle.ID.String(), oldest.ID.String()}, []string{got[0].ID, got[1].ID, got[2].ID})
	require.Len(t, got[0].Options, 4)
	assert.Equal(t, newest.Options[0].Text, got[0].Options[0].Text)
	assert.True(t, got[0].Options[0].Correct)
}

func TestAPI_ListQuestions_GivesAnEmptyArrayWhenTheBankIsEmpty(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)

	res := api.GetAs(t, token, "/api/v1/admin/questions")

	require.Equal(t, http.StatusOK, res.StatusCode, string(res.Body))
	assert.JSONEq(t, `[]`, string(res.Body))
}

func TestAPI_GetQuestion_Returns200404And400(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	question := fixtures.SaveQuestion(t, api.Questions)

	found := api.GetAs(t, token, "/api/v1/admin/questions/"+question.ID.String())
	require.Equal(t, http.StatusOK, found.StatusCode, string(found.Body))
	assert.Equal(t, question.Text, decode[questionBody](t, found).Text)

	requireProblem(t, api.GetAs(t, token, "/api/v1/admin/questions/0198f2a0-0000-7000-8000-000000000000"),
		http.StatusNotFound, "Question does not exist")
	requireProblem(t, api.GetAs(t, token, "/api/v1/admin/questions/not-a-uuid"), http.StatusBadRequest, "ID is not valid")
	requireProblem(t, api.GetAs(t, token, "/api/v1/admin/questions/%7B"+question.ID.String()+"%7D"), http.StatusBadRequest, "ID is not valid")
}

func TestAPI_UpdateQuestion_ReplacesTheTextAndTheOptionsAndKeepsTheIDAndTheCreationTime(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	question := fixtures.SaveQuestion(t, api.Questions)

	res := api.JSONAs(t, token, http.MethodPut, "/api/v1/admin/questions/"+question.ID.String(),
		`{"text":"New text","options":[{"text":"Yes","correct":false},{"text":"No","correct":true}]}`)

	require.Equal(t, http.StatusOK, res.StatusCode, string(res.Body))
	got := decode[questionBody](t, res)
	assert.Equal(t, question.ID.String(), got.ID)
	assert.Equal(t, "New text", got.Text)
	require.Len(t, got.Options, 2)
	assert.Equal(t, "No", got.Options[1].Text)
	assert.NotEqual(t, question.Options[0].ID.String(), got.Options[0].ID)
	createdAt, err := time.Parse(time.RFC3339Nano, got.CreatedAt)
	require.NoError(t, err)
	assert.True(t, question.CreatedAt.Equal(createdAt))
	assert.Equal(t, 2, count(t, api, "SELECT count(*) FROM options WHERE question_id = $1", question.ID))

	read := decode[questionBody](t, api.GetAs(t, token, "/api/v1/admin/questions/"+question.ID.String()))
	assert.Equal(t, got, read)
}

func TestAPI_UpdateQuestion_Returns422AndKeepsTheOldDataWhenARuleRejectsTheInput(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	question := fixtures.SaveQuestion(t, api.Questions)

	res := api.JSONAs(t, token, http.MethodPut, "/api/v1/admin/questions/"+question.ID.String(),
		`{"text":"New","options":[{"text":"A","correct":true}]}`)

	requireProblem(t, res, http.StatusUnprocessableEntity, "too few Options")
	read := decode[questionBody](t, api.GetAs(t, token, "/api/v1/admin/questions/"+question.ID.String()))
	assert.Equal(t, question.Text, read.Text)
	assert.Len(t, read.Options, 4)
}

func TestAPI_UpdateQuestion_Returns404And400(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)

	requireProblem(t, api.JSONAs(t, token, http.MethodPut, "/api/v1/admin/questions/0198f2a0-0000-7000-8000-000000000000", validQuestion),
		http.StatusNotFound, "Question does not exist")
	requireProblem(t, api.JSONAs(t, token, http.MethodPut, "/api/v1/admin/questions/not-a-uuid", validQuestion),
		http.StatusBadRequest, "ID is not valid")
}

func TestAPI_DeleteQuestion_Returns204AndRemovesTheQuestionAndItsOptions(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	question := fixtures.SaveQuestion(t, api.Questions)

	res := api.JSONAs(t, token, http.MethodDelete, "/api/v1/admin/questions/"+question.ID.String(), "")

	require.Equal(t, http.StatusNoContent, res.StatusCode, string(res.Body))
	assert.Zero(t, count(t, api, "SELECT count(*) FROM questions"))
	assert.Zero(t, count(t, api, "SELECT count(*) FROM options"))
	requireProblem(t, api.JSONAs(t, token, http.MethodDelete, "/api/v1/admin/questions/"+question.ID.String(), ""),
		http.StatusNotFound, "Question does not exist")
	requireProblem(t, api.JSONAs(t, token, http.MethodDelete, "/api/v1/admin/questions/not-a-uuid", ""),
		http.StatusBadRequest, "ID is not valid")
}

func TestAPI_ListQuestions_Returns500WithAGeneralTextWhenTheDatabaseFails(t *testing.T) {
	api := testapi.Start(t)
	token := api.LogIn(t)
	api.BreakDatabase(t)

	res := api.GetAs(t, token, "/api/v1/admin/questions")

	// The guard reads the Admin of the session from the database, so the guard fails first. The response is the same.
	requireProblem(t, res, http.StatusInternalServerError, "cannot complete the request")
	assert.NotContains(t, string(res.Body), "sql")
}
