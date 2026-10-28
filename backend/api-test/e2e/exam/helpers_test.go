//go:build e2e

package exam_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudradcruze/go-react-template/backend/api-test/e2e/testapi"
	"github.com/rudradcruze/go-react-template/backend/internal/platform/problem"
)

func decode[T any](t *testing.T, res testapi.Response) T {
	t.Helper()
	var v T
	require.NoError(t, json.Unmarshal(res.Body, &v), string(res.Body))
	return v
}

func requireProblem(t *testing.T, res testapi.Response, status int, detailPart string) {
	t.Helper()
	require.Equal(t, status, res.StatusCode, string(res.Body))
	assert.Equal(t, "application/problem+json", res.Header.Get("Content-Type"))
	body := decode[problem.Body](t, res)
	assert.Equal(t, status, body.Status)
	assert.Equal(t, http.StatusText(status), body.Title)
	assert.Contains(t, body.Detail, detailPart)
}

func count(t *testing.T, api *testapi.API, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, api.DB.QueryRowContext(t.Context(), query, args...).Scan(&n))
	return n
}
