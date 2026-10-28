//go:build e2e

// Package testapi starts the real API for an E2E test. It checks each response against the OpenAPI spec.
package testapi

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/rudradcruze/go-react-template/backend/api"
	"github.com/rudradcruze/go-react-template/backend/api-test/e2e/testenv"
	"github.com/rudradcruze/go-react-template/backend/api-test/fixtures"
	adminapp "github.com/rudradcruze/go-react-template/backend/internal/admin/app"
	admininfra "github.com/rudradcruze/go-react-template/backend/internal/admin/infra"
	"github.com/rudradcruze/go-react-template/backend/internal/bootstrap"
	examapp "github.com/rudradcruze/go-react-template/backend/internal/exam/app"
	examinfra "github.com/rudradcruze/go-react-template/backend/internal/exam/infra"
	"github.com/rudradcruze/go-react-template/backend/internal/platform/cache"
	"github.com/rudradcruze/go-react-template/backend/internal/platform/config"
	"github.com/rudradcruze/go-react-template/backend/internal/platform/database"
	"github.com/rudradcruze/go-react-template/backend/internal/platform/queue"
)

// logBuffer is a buffer that the API and the test can use at the same time.
type logBuffer struct {
	mu   sync.Mutex
	data bytes.Buffer
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.data.Write(p)
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.data.String()
}

// API is a running API, with its own database and its own Redis data.
type API struct {
	URL string
	// DB is a connection to the database of the test, to check the data after a request.
	DB *sql.DB
	// Admins and Hasher work on the database of the test. Use them with fixtures.SaveAdmin to prepare data.
	Admins adminapp.AdminRepository
	Hasher adminapp.PasswordHasher
	// Questions works on the database of the test. Use it with fixtures.SaveQuestion to prepare data.
	Questions examapp.QuestionRepository
	// Exams works on the database of the test. Use it with fixtures.SaveExam to prepare data.
	Exams examapp.ExamRepository
	// Attempts works on the database of the test. Use it with fixtures.SaveAttempt to prepare data.
	Attempts examapp.AttemptRepository
	// SeedAdmin is the use case that the migrate command runs, on the database of the test.
	SeedAdmin *adminapp.SeedAdmin
	// Redis is a client on the Redis data of the test, to check the cache after a request.
	Redis *redis.Client
	// QueueInspector reads the job queue, to check the tasks after a request.
	QueueInspector *asynq.Inspector
	logs           *logBuffer    // The logs of the API.
	redisSettings  config.Redis  // The Redis settings of the test.
	gormDB         *gorm.DB      // The database connection that the API and the worker use.
	queueClient    *asynq.Client // The queue client that the API uses.
	apiPool        *sql.DB       // The connection pool that the API uses.
	apiCache       *redis.Client // The Redis client that the API uses.
	validator      validator.Validator
}

// Response is the part of an HTTP response that the tests check.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// Start starts the API on a test port, with the database of the test (see testenv.Begin).
// The test closes everything when it ends.
func Start(t *testing.T) *API {
	t.Helper()
	res := testenv.Begin(t)

	gormDB, err := database.Open(t.Context(), res.Postgres)
	if err != nil {
		t.Fatalf("The API database did not open. Cause: %v", err)
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		t.Fatalf("The API database is not available. Cause: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	apiCache, err := cache.Open(t.Context(), res.Redis)
	if err != nil {
		t.Fatalf("The API cache did not open. Cause: %v", err)
	}
	t.Cleanup(func() { _ = apiCache.Close() })

	queueClient, err := queue.Open(res.Redis)
	if err != nil {
		t.Fatalf("The API queue did not open. Cause: %v", err)
	}
	t.Cleanup(func() { _ = queueClient.Close() })
	inspector := asynq.NewInspector(queue.WorkerOpt(res.Redis))
	t.Cleanup(func() { _ = inspector.Close() })

	logs := &logBuffer{}
	handler := bootstrap.NewAPI(config.Config{}, slog.New(slog.NewJSONHandler(logs, nil)), gormDB, apiCache, queueClient)
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	doc, err := libopenapi.NewDocument(api.Spec)
	if err != nil {
		t.Fatalf("The OpenAPI spec did not load. Cause: %v", err)
	}
	specValidator, errs := validator.NewValidator(doc)
	if len(errs) > 0 {
		t.Fatalf("The OpenAPI validator did not start. Cause: %v", errs)
	}
	return &API{
		URL:            srv.URL,
		DB:             testenv.OpenDB(t, res.Postgres),
		Admins:         admininfra.NewAdminRepository(gormDB),
		Hasher:         admininfra.NewBcryptHasher(),
		Questions:      examinfra.NewQuestionRepository(gormDB),
		Exams:          examinfra.NewExamRepository(gormDB),
		Attempts:       examinfra.NewAttemptRepository(gormDB),
		SeedAdmin:      bootstrap.NewSeedAdmin(gormDB),
		Redis:          testenv.OpenRedis(t, res.Redis),
		QueueInspector: inspector,
		logs:           logs,
		redisSettings:  res.Redis,
		gormDB:         gormDB,
		queueClient:    queueClient,
		apiPool:        sqlDB,
		apiCache:       apiCache,
		validator:      specValidator,
	}
}

// BreakQueue closes the queue client of the API, so that the next call of the API to the queue fails.
func (a *API) BreakQueue(t *testing.T) {
	t.Helper()
	if err := a.queueClient.Close(); err != nil {
		t.Fatalf("The queue client of the API did not close. Cause: %v", err)
	}
}

// Enqueue adds a task to the queue of the API.
func (a *API) Enqueue(t *testing.T, task *asynq.Task, options ...asynq.Option) {
	t.Helper()
	if _, err := a.queueClient.EnqueueContext(t.Context(), task, options...); err != nil {
		t.Fatalf("The task did not enter the queue. Cause: %v", err)
	}
}

// StartWorker starts a worker on the queue and on the database of the test. The worker writes its logs
// to the logs of the API (see Logs). The test stops the worker when it ends.
func (a *API) StartWorker(t *testing.T) {
	t.Helper()
	log := slog.New(slog.NewJSONHandler(a.logs, nil))
	server := asynq.NewServer(queue.WorkerOpt(a.redisSettings), asynq.Config{
		Concurrency:         2,
		TaskCheckInterval:   100 * time.Millisecond,
		Logger:              queue.NewLogger(log),
		ErrorHandler:        queue.NewErrorHandler(log),
		HealthCheckInterval: time.Second,
	})
	if err := server.Start(bootstrap.NewWorkerMux(log, a.gormDB)); err != nil {
		t.Fatalf("The worker did not start. Cause: %v", err)
	}
	t.Cleanup(server.Shutdown)
}

// Logs gives all the log lines that the API wrote until now.
func (a *API) Logs() string { return a.logs.String() }

// BreakDatabase closes the connection pool of the API, so that the next database call of the API fails.
// Use it to test an unexpected error.
func (a *API) BreakDatabase(t *testing.T) {
	t.Helper()
	if err := a.apiPool.Close(); err != nil {
		t.Fatalf("The connection pool of the API did not close. Cause: %v", err)
	}
}

// BreakCache closes the Redis client of the API, so that the next cache call of the API fails.
// Use it to test that the API still works when Redis fails.
func (a *API) BreakCache(t *testing.T) {
	t.Helper()
	if err := a.apiCache.Close(); err != nil {
		t.Fatalf("The Redis client of the API did not close. Cause: %v", err)
	}
}

// Do sends a request and checks the response against the spec. The check fails the test when they do not match.
func (a *API) Do(t *testing.T, method, path, contentType string, body []byte) Response {
	t.Helper()
	return a.send(t, "", method, path, contentType, body)
}

// DoWithSession is like Do, but it sends the session cookie with this token.
func (a *API) DoWithSession(t *testing.T, token, method, path, contentType string, body []byte) Response {
	t.Helper()
	return a.send(t, token, method, path, contentType, body)
}

func (a *API) send(t *testing.T, sessionToken, method, path, contentType string, body []byte) Response {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, a.URL+path, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("The request is not valid. Cause: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if sessionToken != "" {
		req.Header.Set("Cookie", "session="+sessionToken)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("The request failed. Cause: %v", err)
	}
	defer func() { _ = res.Body.Close() }()
	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("The test did not read the response body. Cause: %v", err)
	}

	res.Body = io.NopCloser(bytes.NewReader(responseBody))
	if ok, errs := a.validator.ValidateHttpResponse(req, res); !ok {
		for _, e := range errs {
			t.Errorf("The response does not match the OpenAPI spec: %s. %s", e.Message, e.Reason)
		}
	}
	return Response{StatusCode: res.StatusCode, Header: res.Header, Body: responseBody}
}

// LogIn saves a new Admin, logs in as that Admin through the API, and gives the session token.
// Send it with DoWithSession, PostJSONAs or the other helpers.
func (a *API) LogIn(t *testing.T) string {
	t.Helper()
	admin, password := fixtures.SaveAdmin(t, a.Admins, a.Hasher)
	body := fmt.Sprintf(`{"emailAddress":%q,"password":%q}`, admin.EmailAddress, password)
	res := a.PostJSON(t, "/api/v1/admin/session", body)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("The login of the test Admin failed with status %d: %s", res.StatusCode, res.Body)
	}
	for _, cookie := range (&http.Response{Header: res.Header}).Cookies() {
		if cookie.Name == "session" {
			return cookie.Value
		}
	}
	t.Fatalf("The login of the test Admin gave no session cookie.")
	return ""
}

// JSONAs sends a request with a JSON body and the session cookie of an Admin.
func (a *API) JSONAs(t *testing.T, token, method, path, body string) Response {
	t.Helper()
	return a.send(t, token, method, path, "application/json", []byte(body))
}

// GetAs sends a GET request with the session cookie of an Admin.
func (a *API) GetAs(t *testing.T, token, path string) Response {
	t.Helper()
	return a.send(t, token, http.MethodGet, path, "", nil)
}

// Get sends a GET request.
func (a *API) Get(t *testing.T, path string) Response {
	t.Helper()
	return a.Do(t, http.MethodGet, path, "", nil)
}

// PostJSON sends a POST request with a JSON body.
func (a *API) PostJSON(t *testing.T, path, body string) Response {
	t.Helper()
	return a.Do(t, http.MethodPost, path, "application/json", []byte(body))
}
