package platform_test

import (
	"log/slog"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudradcruze/go-react-template/backend/internal/platform/config"
)

func TestConfig_LoadFrom_UsesTheDefaultsWhenTheEnvironmentIsEmpty(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFrom(map[string]string{})

	require.NoError(t, err)
	assert.Equal(t, 5653, cfg.APIPort)
	assert.Equal(t, slog.LevelInfo, cfg.LogLevel)
	assert.False(t, cfg.Docs.Enabled)
	assert.True(t, cfg.Docs.AuthEnabled)
	assert.Empty(t, cfg.Docs.Username)
	assert.Empty(t, cfg.Docs.Password)
}

func TestConfig_LoadFrom_UsesTheDefaultsForPostgresAndRedis(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFrom(map[string]string{})

	require.NoError(t, err)
	assert.Equal(t, config.Postgres{
		Host: "localhost", Port: 5654, User: "app", DB: "app", SSLMode: "disable",
	}, cfg.Postgres)
	assert.Equal(t, config.Redis{
		Host: "localhost", Port: 5652, CacheDB: 0, QueueDB: 1,
	}, cfg.Redis)
}

func TestConfig_LoadFrom_ReadsThePostgresAndRedisValues(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFrom(map[string]string{
		"POSTGRES_HOST":     "db.internal",
		"POSTGRES_PORT":     "5432",
		"POSTGRES_USER":     "owner",
		"POSTGRES_PASSWORD": "secret",
		"POSTGRES_DB":       "orders",
		"POSTGRES_SSLMODE":  "require",
		"REDIS_HOST":        "cache.internal",
		"REDIS_PORT":        "6379",
		"REDIS_PASSWORD":    "other",
		"REDIS_CACHE_DB":    "2",
		"REDIS_QUEUE_DB":    "3",
	})

	require.NoError(t, err)
	assert.Equal(t, config.Postgres{
		Host: "db.internal", Port: 5432, User: "owner", Password: "secret", DB: "orders", SSLMode: "require",
	}, cfg.Postgres)
	assert.Equal(t, config.Redis{
		Host: "cache.internal", Port: 6379, Password: "other", CacheDB: 2, QueueDB: 3,
	}, cfg.Redis)
}

func TestPostgres_DSN_EscapesSpecialCharacters(t *testing.T) {
	t.Parallel()
	pg := config.Postgres{ //nolint:gosec // The password is a test value, not a real credential.
		Host: "db.internal", Port: 5654, User: "ow ner", Password: "p@ss:w/rd#?%", DB: "app", SSLMode: "require",
	}

	parsed, err := url.Parse(pg.DSN())

	require.NoError(t, err)
	password, _ := parsed.User.Password()
	assert.Equal(t, "postgres", parsed.Scheme)
	assert.Equal(t, "ow ner", parsed.User.Username())
	assert.Equal(t, "p@ss:w/rd#?%", password)
	assert.Equal(t, "db.internal:5654", parsed.Host)
	assert.Equal(t, "/app", parsed.Path)
	assert.Equal(t, "require", parsed.Query().Get("sslmode"))
}

func TestRedis_Addr_JoinsTheHostAndThePort(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "localhost:5652", config.Redis{Host: "localhost", Port: 5652}.Addr())
	assert.Equal(t, "[::1]:5652", config.Redis{Host: "::1", Port: 5652}.Addr())
}

func TestConfig_LoadFrom_ReadsTheGivenValues(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFrom(map[string]string{
		"LOG_LEVEL":             "debug",
		"API_PORT":              "6000",
		"API_DOCS_ENABLED":      "true",
		"API_DOCS_AUTH_ENABLED": "true",
		"API_DOCS_USERNAME":     "reader",
		"API_DOCS_PASSWORD":     "secret",
	})

	require.NoError(t, err)
	assert.Equal(t, 6000, cfg.APIPort)
	assert.Equal(t, slog.LevelDebug, cfg.LogLevel)
	assert.True(t, cfg.Docs.Enabled)
	assert.True(t, cfg.Docs.AuthEnabled)
	assert.Equal(t, "reader", cfg.Docs.Username)
	assert.Equal(t, "secret", cfg.Docs.Password)
}

func TestConfig_LoadFrom_RejectsDocsAuthWithoutCredentials(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		env  map[string]string
	}{
		{
			name: "no username and no password",
			env:  map[string]string{"API_DOCS_ENABLED": "true"},
		},
		{
			name: "no username",
			env:  map[string]string{"API_DOCS_ENABLED": "true", "API_DOCS_PASSWORD": "secret"},
		},
		{
			name: "no password",
			env:  map[string]string{"API_DOCS_ENABLED": "true", "API_DOCS_USERNAME": "reader"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := config.LoadFrom(tt.env)

			require.ErrorContains(t, err, "API_DOCS_USERNAME")
			require.ErrorContains(t, err, "API_DOCS_PASSWORD")
		})
	}
}

func TestConfig_LoadFrom_AcceptsDocsWithoutAuthAndWithoutCredentials(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFrom(map[string]string{
		"API_DOCS_ENABLED":      "true",
		"API_DOCS_AUTH_ENABLED": "false",
	})

	require.NoError(t, err)
	assert.True(t, cfg.Docs.Enabled)
	assert.False(t, cfg.Docs.AuthEnabled)
}

func TestConfig_LoadFrom_RejectsInvalidValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		env      map[string]string
		variable string
	}{
		{name: "port is not a number", env: map[string]string{"API_PORT": "abc"}, variable: "API_PORT"},
		{name: "port is zero", env: map[string]string{"API_PORT": "0"}, variable: "API_PORT"},
		{name: "port is too high", env: map[string]string{"API_PORT": "70000"}, variable: "API_PORT"},
		{name: "log level is unknown", env: map[string]string{"LOG_LEVEL": "loud"}, variable: "LOG_LEVEL"},
		{name: "docs flag is not a boolean", env: map[string]string{"API_DOCS_ENABLED": "maybe"}, variable: "API_DOCS_ENABLED"},
		{name: "Postgres port is zero", env: map[string]string{"POSTGRES_PORT": "0"}, variable: "POSTGRES_PORT"},
		{name: "Redis port is too high", env: map[string]string{"REDIS_PORT": "70000"}, variable: "REDIS_PORT"},
		{name: "Redis cache database is negative", env: map[string]string{"REDIS_CACHE_DB": "-1"}, variable: "REDIS_CACHE_DB"},
		{name: "Redis queue database is negative", env: map[string]string{"REDIS_QUEUE_DB": "-1"}, variable: "REDIS_QUEUE_DB"},
		{name: "Postgres SSL mode is unknown", env: map[string]string{"POSTGRES_SSLMODE": "maybe"}, variable: "POSTGRES_SSLMODE"},
		{
			name:     "cache and queue use the same Redis database",
			env:      map[string]string{"REDIS_CACHE_DB": "2", "REDIS_QUEUE_DB": "2"},
			variable: "REDIS_QUEUE_DB",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := config.LoadFrom(tt.env)

			require.ErrorContains(t, err, tt.variable)
		})
	}
}

func TestConfig_LoadMigrateFrom_UsesTheDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadMigrateFrom(map[string]string{})

	require.NoError(t, err)
	assert.Equal(t, "migrations", cfg.Dir)
	assert.Equal(t, slog.LevelInfo, cfg.LogLevel)
	assert.Equal(t, "localhost", cfg.Postgres.Host)
	assert.Equal(t, 5654, cfg.Postgres.Port)
}

func TestConfig_LoadMigrateFrom_ReadsTheGivenValues(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadMigrateFrom(map[string]string{
		"MIGRATIONS_DIR":    "/migrations",
		"LOG_LEVEL":         "debug",
		"POSTGRES_HOST":     "postgres",
		"POSTGRES_PASSWORD": "secret",
	})

	require.NoError(t, err)
	assert.Equal(t, "/migrations", cfg.Dir)
	assert.Equal(t, slog.LevelDebug, cfg.LogLevel)
	assert.Equal(t, "postgres", cfg.Postgres.Host)
	assert.Equal(t, "secret", cfg.Postgres.Password)
}

func TestConfig_LoadMigrateFrom_IgnoresSettingsThatTheCommandDoesNotUse(t *testing.T) {
	t.Parallel()

	_, err := config.LoadMigrateFrom(map[string]string{
		"API_DOCS_ENABLED": "true", // The docs have no credentials, so config.LoadFrom rejects this.
		"API_PORT":         "0",
		"REDIS_PORT":       "0",
	})

	require.NoError(t, err)
}

func TestConfig_LoadMigrateFrom_RejectsInvalidPostgresValues(t *testing.T) {
	t.Parallel()

	_, err := config.LoadMigrateFrom(map[string]string{"POSTGRES_PORT": "0", "POSTGRES_SSLMODE": "maybe"})

	require.ErrorContains(t, err, "POSTGRES_PORT")
	require.ErrorContains(t, err, "POSTGRES_SSLMODE")
}

func TestConfig_LoadWorkerFrom_UsesTheDefaultsAndReadsTheGivenValues(t *testing.T) {
	t.Parallel()

	defaults, err := config.LoadWorkerFrom(map[string]string{})
	require.NoError(t, err)
	given, givenErr := config.LoadWorkerFrom(map[string]string{
		"LOG_LEVEL":      "debug",
		"POSTGRES_HOST":  "postgres",
		"REDIS_HOST":     "redis",
		"REDIS_QUEUE_DB": "3",
	})

	assert.Equal(t, slog.LevelInfo, defaults.LogLevel)
	assert.Equal(t, 5654, defaults.Postgres.Port)
	assert.Equal(t, config.Redis{Host: "localhost", Port: 5652, CacheDB: 0, QueueDB: 1}, defaults.Redis)
	require.NoError(t, givenErr)
	assert.Equal(t, slog.LevelDebug, given.LogLevel)
	assert.Equal(t, "postgres", given.Postgres.Host)
	assert.Equal(t, "redis", given.Redis.Host)
	assert.Equal(t, 3, given.Redis.QueueDB)
}

func TestConfig_LoadWorkerFrom_ReadsTheConcurrency(t *testing.T) {
	t.Parallel()

	defaults, err := config.LoadWorkerFrom(map[string]string{})
	require.NoError(t, err)
	given, givenErr := config.LoadWorkerFrom(map[string]string{"WORKER_CONCURRENCY": "4"})
	_, zeroErr := config.LoadWorkerFrom(map[string]string{"WORKER_CONCURRENCY": "0"})

	assert.Equal(t, 10, defaults.Concurrency)
	require.NoError(t, givenErr)
	assert.Equal(t, 4, given.Concurrency)
	require.ErrorContains(t, zeroErr, "WORKER_CONCURRENCY")
}

func TestConfig_LoadWorkerFrom_IgnoresSettingsThatTheWorkerDoesNotUse(t *testing.T) {
	t.Parallel()

	_, err := config.LoadWorkerFrom(map[string]string{
		"API_DOCS_ENABLED": "true", // The docs have no credentials, so config.LoadFrom rejects this.
		"API_PORT":         "0",
	})

	require.NoError(t, err)
}

func TestConfig_LoadWorkerFrom_RejectsInvalidPostgresAndRedisValues(t *testing.T) {
	t.Parallel()

	_, err := config.LoadWorkerFrom(map[string]string{"POSTGRES_PORT": "0", "REDIS_PORT": "0", "REDIS_QUEUE_DB": "0"})

	require.ErrorContains(t, err, "POSTGRES_PORT")
	require.ErrorContains(t, err, "REDIS_PORT")
	require.ErrorContains(t, err, "REDIS_QUEUE_DB") // The cache database is 0 as well, so the 2 databases are the same.
}

// A person copies .env.example to .env, so the file must give a valid config.
func TestConfig_LoadFrom_AcceptsTheEnvExampleFile(t *testing.T) {
	t.Parallel()
	content, err := os.ReadFile("../../../../.env.example")
	require.NoError(t, err)
	environment := map[string]string{}
	for line := range strings.Lines(string(content)) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, _ := strings.Cut(line, "=")
		environment[key] = value
	}

	cfg, err := config.LoadFrom(environment)

	require.NoError(t, err)
	assert.Equal(t, 5653, cfg.APIPort)
	assert.False(t, cfg.Docs.Enabled)
}

// The next 3 tests cannot call t.Parallel, because t.Setenv changes the environment of the process.

func TestConfig_Load_ReadsTheEnvironmentOfTheProcess(t *testing.T) {
	t.Setenv("API_PORT", "6001")

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, 6001, cfg.APIPort)
}

func TestConfig_LoadFrom_IgnoresTheEnvironmentOfTheProcess(t *testing.T) {
	t.Setenv("API_PORT", "6002")

	cfg, err := config.LoadFrom(nil)

	require.NoError(t, err)
	assert.Equal(t, 5653, cfg.APIPort)
}

func TestConfig_LoadMigrate_ReadsTheEnvironmentOfTheProcess(t *testing.T) {
	t.Setenv("MIGRATIONS_DIR", "/from-process")

	cfg, err := config.LoadMigrate()

	require.NoError(t, err)
	assert.Equal(t, "/from-process", cfg.Dir)
}

func TestConfig_LoadMigrateFrom_ReadsTheSettingsOfTheFirstAdmin(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadMigrateFrom(map[string]string{"ADMIN_EMAIL": "owner@example.com", "ADMIN_PASSWORD": "a password of 12+"})

	require.NoError(t, err)
	assert.Equal(t, "owner@example.com", cfg.AdminEmail)
	assert.Equal(t, "a password of 12+", cfg.AdminPassword)
}

func TestConfig_LoadMigrateFrom_LeavesTheSettingsOfTheFirstAdminEmptyByDefault(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadMigrateFrom(map[string]string{})

	require.NoError(t, err)
	assert.Empty(t, cfg.AdminEmail)
	assert.Empty(t, cfg.AdminPassword)
}
