//go:build e2e

package admin_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudradcruze/go-react-template/backend/api-test/e2e/testapi"
	"github.com/rudradcruze/go-react-template/backend/api-test/fixtures"
	"github.com/rudradcruze/go-react-template/backend/internal/platform/problem"
)

func decode[T any](t *testing.T, res testapi.Response) T {
	t.Helper()
	var v T
	require.NoError(t, json.Unmarshal(res.Body, &v), string(res.Body))
	return v
}

func requireProblem(t *testing.T, res testapi.Response, status int, detailPart string) problem.Body {
	t.Helper()
	require.Equal(t, status, res.StatusCode, string(res.Body))
	assert.Equal(t, "application/problem+json", res.Header.Get("Content-Type"))
	body := decode[problem.Body](t, res)
	assert.Equal(t, status, body.Status)
	assert.Equal(t, http.StatusText(status), body.Title)
	assert.Contains(t, body.Detail, detailPart)
	return body
}

func loginBody(emailAddress, password string) string {
	body, _ := json.Marshal(map[string]string{"emailAddress": emailAddress, "password": password})
	return string(body)
}

// sessionCookie gives the cookie "session" of a response.
func sessionCookie(t *testing.T, res testapi.Response) *http.Cookie {
	t.Helper()
	for _, cookie := range (&http.Response{Header: res.Header}).Cookies() {
		if cookie.Name == "session" {
			return cookie
		}
	}
	t.Fatalf("The response has no session cookie. Headers: %v", res.Header)
	return nil
}

func logIn(t *testing.T, api *testapi.API, emailAddress, password string) string {
	t.Helper()
	res := api.PostJSON(t, "/api/v1/admin/session", loginBody(emailAddress, password))
	require.Equal(t, http.StatusNoContent, res.StatusCode, string(res.Body))
	return sessionCookie(t, res).Value
}

func TestAPI_Login_Returns204AndSetsAnHttpOnlyCookieForTheCorrectPassword(t *testing.T) {
	api := testapi.Start(t)
	admin, password := fixtures.SaveAdmin(t, api.Admins, api.Hasher)

	res := api.PostJSON(t, "/api/v1/admin/session", loginBody("  "+strings.ToUpper(string(admin.EmailAddress))+" ", password))

	require.Equal(t, http.StatusNoContent, res.StatusCode, string(res.Body))
	cookie := sessionCookie(t, res)
	assert.NotEmpty(t, cookie.Value)
	assert.True(t, cookie.HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	assert.Equal(t, "/", cookie.Path)
	assert.Equal(t, 8*60*60, cookie.MaxAge)
}

func TestAPI_Login_KeepsTheSessionInRedisFor8HoursWithoutTheToken(t *testing.T) {
	api := testapi.Start(t)
	admin, password := fixtures.SaveAdmin(t, api.Admins, api.Hasher)

	token := logIn(t, api, string(admin.EmailAddress), password)

	keys, err := api.Redis.Keys(t.Context(), "admin:session:*").Result()
	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.NotContains(t, keys[0], token, "Redis must have a hash of the token, not the token.")
	value, err := api.Redis.Get(t.Context(), keys[0]).Result()
	require.NoError(t, err)
	assert.Equal(t, admin.ID.String(), value)
	ttl, err := api.Redis.TTL(t.Context(), keys[0]).Result()
	require.NoError(t, err)
	assert.InDelta(t, (8 * time.Hour).Seconds(), ttl.Seconds(), 60)
}

func TestAPI_Login_Returns401WithTheSameTextForAWrongPasswordAndAnUnknownEmailAddress(t *testing.T) {
	api := testapi.Start(t)
	admin, _ := fixtures.SaveAdmin(t, api.Admins, api.Hasher)

	wrongPassword := api.PostJSON(t, "/api/v1/admin/session", loginBody(string(admin.EmailAddress), "not the password"))
	unknownEmail := api.PostJSON(t, "/api/v1/admin/session", loginBody("nobody@example.com", "not the password"))

	first := requireProblem(t, wrongPassword, http.StatusUnauthorized, "email address or the password is not correct")
	second := requireProblem(t, unknownEmail, http.StatusUnauthorized, "email address or the password is not correct")
	assert.Equal(t, first, second)
	assert.Empty(t, (&http.Response{Header: wrongPassword.Header}).Cookies())
}

func TestAPI_Login_Returns401ForAnEmailAddressThatIsNotValidAndForAnEmptyPassword(t *testing.T) {
	api := testapi.Start(t)

	for _, body := range []string{loginBody("admin", "a password"), loginBody("admin@example.com", ""), `{}`} {
		requireProblem(t, api.PostJSON(t, "/api/v1/admin/session", body), http.StatusUnauthorized, "not correct")
	}
}

func TestAPI_Login_Returns429AfterFiveFailuresAlsoForTheCorrectPasswordAndOnlyForThatEmailAddress(t *testing.T) {
	api := testapi.Start(t)
	admin, password := fixtures.SaveAdmin(t, api.Admins, api.Hasher)
	other, otherPassword := fixtures.SaveAdmin(t, api.Admins, api.Hasher)
	for range 5 {
		requireProblem(t, api.PostJSON(t, "/api/v1/admin/session", loginBody(string(admin.EmailAddress), "wrong")),
			http.StatusUnauthorized, "not correct")
	}

	blocked := api.PostJSON(t, "/api/v1/admin/session", loginBody(strings.ToUpper(string(admin.EmailAddress)), password))

	requireProblem(t, blocked, http.StatusTooManyRequests, "too many failed logins")
	retryAfter, err := strconv.Atoi(blocked.Header.Get("Retry-After"))
	require.NoError(t, err)
	assert.InDelta(t, (15 * time.Minute).Seconds(), float64(retryAfter), 60)
	assert.Empty(t, (&http.Response{Header: blocked.Header}).Cookies())
	logIn(t, api, string(other.EmailAddress), otherPassword)
}

func TestAPI_Login_AllowsTheFifthTryWhenTheFirstFourFailed(t *testing.T) {
	api := testapi.Start(t)
	admin, password := fixtures.SaveAdmin(t, api.Admins, api.Hasher)
	for range 4 {
		api.PostJSON(t, "/api/v1/admin/session", loginBody(string(admin.EmailAddress), "wrong"))
	}

	logIn(t, api, string(admin.EmailAddress), password)
}

func TestAPI_Login_RejectsABodyThatIsNotValid(t *testing.T) {
	api := testapi.Start(t)

	requireProblem(t, api.PostJSON(t, "/api/v1/admin/session", `{"emailAddress":`), http.StatusBadRequest, "not valid JSON")
	requireProblem(t, api.PostJSON(t, "/api/v1/admin/session", `{"emailAddress":"a@example.com","password":"p","role":"x"}`),
		http.StatusBadRequest, "not valid JSON")
	requireProblem(t, api.Do(t, http.MethodPost, "/api/v1/admin/session", "text/plain", []byte("x")),
		http.StatusUnsupportedMediaType, "content type")
	requireProblem(t, api.PostJSON(t, "/api/v1/admin/session", `{"password":"`+strings.Repeat("a", 1_500_000)+`"}`),
		http.StatusRequestEntityTooLarge, "larger than 1 MiB")
}

func TestAPI_GetSession_Returns200WithTheAdminOfTheSession(t *testing.T) {
	api := testapi.Start(t)
	admin, password := fixtures.SaveAdmin(t, api.Admins, api.Hasher)
	token := logIn(t, api, string(admin.EmailAddress), password)

	res := api.DoWithSession(t, token, http.MethodGet, "/api/v1/admin/session", "", nil)

	require.Equal(t, http.StatusOK, res.StatusCode, string(res.Body))
	got := decode[map[string]string](t, res)
	assert.Equal(t, map[string]string{"id": admin.ID.String(), "emailAddress": string(admin.EmailAddress)}, got)
	assert.NotContains(t, string(res.Body), "assword")
}

func TestAPI_GetSession_Returns401WithoutAValidSession(t *testing.T) {
	api := testapi.Start(t)

	requireProblem(t, api.Get(t, "/api/v1/admin/session"), http.StatusUnauthorized, "not logged in")
	requireProblem(t, api.DoWithSession(t, "an-unknown-token", http.MethodGet, "/api/v1/admin/session", "", nil),
		http.StatusUnauthorized, "not logged in")
}

func TestAPI_GetSession_Returns401WhenTheSessionEnded(t *testing.T) {
	api := testapi.Start(t)
	admin, password := fixtures.SaveAdmin(t, api.Admins, api.Hasher)
	token := logIn(t, api, string(admin.EmailAddress), password)
	keys, err := api.Redis.Keys(t.Context(), "admin:session:*").Result()
	require.NoError(t, err)
	require.NoError(t, api.Redis.Del(t.Context(), keys...).Err())

	res := api.DoWithSession(t, token, http.MethodGet, "/api/v1/admin/session", "", nil)

	requireProblem(t, res, http.StatusUnauthorized, "not logged in")
}

func TestAPI_Logout_Returns204EndsTheSessionAndRemovesTheCookie(t *testing.T) {
	api := testapi.Start(t)
	admin, password := fixtures.SaveAdmin(t, api.Admins, api.Hasher)
	token := logIn(t, api, string(admin.EmailAddress), password)

	res := api.DoWithSession(t, token, http.MethodDelete, "/api/v1/admin/session", "", nil)

	require.Equal(t, http.StatusNoContent, res.StatusCode, string(res.Body))
	assert.Negative(t, sessionCookie(t, res).MaxAge)
	requireProblem(t, api.DoWithSession(t, token, http.MethodGet, "/api/v1/admin/session", "", nil),
		http.StatusUnauthorized, "not logged in")
}

func TestAPI_Logout_Returns401WithoutASession(t *testing.T) {
	api := testapi.Start(t)

	requireProblem(t, api.Do(t, http.MethodDelete, "/api/v1/admin/session", "", nil), http.StatusUnauthorized, "not logged in")
}

func TestAPI_Login_Returns500WhenRedisFails(t *testing.T) {
	api := testapi.Start(t)
	admin, password := fixtures.SaveAdmin(t, api.Admins, api.Hasher)
	api.BreakCache(t)

	res := api.PostJSON(t, "/api/v1/admin/session", loginBody(string(admin.EmailAddress), password))

	requireProblem(t, res, http.StatusInternalServerError, "cannot complete the request")
	assert.NotContains(t, string(res.Body), "redis")
}
