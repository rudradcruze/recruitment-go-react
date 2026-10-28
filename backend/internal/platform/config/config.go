// Package config reads the settings of the application from environment variables.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"reflect"
	"slices"
	"strconv"

	"github.com/caarlos0/env/v11"
)

// Logging holds the settings of the logger.
type Logging struct {
	LogLevel slog.Level `env:"LOG_LEVEL" envDefault:"info"`
}

// Config holds all settings of the API.
type Config struct {
	Logging
	APIPort  int `env:"API_PORT" envDefault:"5653"`
	Docs     Docs
	Postgres Postgres
	Redis    Redis
}

// Migrate holds the settings of the migrate command. It has only the settings that the command uses.
type Migrate struct {
	Logging
	Postgres Postgres
	Dir      string `env:"MIGRATIONS_DIR" envDefault:"migrations"`
	// AdminEmail and AdminPassword make the first Admin after the migrations. The command checks them only for "up".
	AdminEmail    string `env:"ADMIN_EMAIL"`
	AdminPassword string `env:"ADMIN_PASSWORD"`
}

// Worker holds the settings of the worker. It has only the settings that the worker uses.
type Worker struct {
	Logging
	Postgres Postgres
	Redis    Redis
	// Concurrency is the number of tasks that the worker runs at the same time.
	Concurrency int `env:"WORKER_CONCURRENCY" envDefault:"10"`
}

// Docs holds the settings of the API docs.
type Docs struct {
	Enabled     bool   `env:"API_DOCS_ENABLED" envDefault:"false"`
	AuthEnabled bool   `env:"API_DOCS_AUTH_ENABLED" envDefault:"true"`
	Username    string `env:"API_DOCS_USERNAME"`
	Password    string `env:"API_DOCS_PASSWORD"`
}

var sslModes = []string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"}

// Postgres holds the settings of the connection to Postgres.
type Postgres struct {
	Host     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	Port     int    `env:"POSTGRES_PORT" envDefault:"5654"`
	User     string `env:"POSTGRES_USER" envDefault:"app"`
	Password string `env:"POSTGRES_PASSWORD"`
	DB       string `env:"POSTGRES_DB" envDefault:"app"`
	// SSLMode is "disable" by default, for a private network.
	// On a public network, use "require", "verify-ca" or "verify-full".
	SSLMode string `env:"POSTGRES_SSLMODE" envDefault:"disable"`
}

// Addr gives the address of Postgres in the form host:port.
func (p Postgres) Addr() string {
	return net.JoinHostPort(p.Host, strconv.Itoa(p.Port))
}

// DSN gives the connection string of the database. It escapes special characters.
func (p Postgres) DSN() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(p.User, p.Password),
		Host:     p.Addr(),
		Path:     "/" + p.DB,
		RawQuery: url.Values{"sslmode": {p.SSLMode}}.Encode(),
	}
	return u.String()
}

func (p Postgres) validate() []error {
	errs := checkPort("POSTGRES_PORT", p.Port)
	if !slices.Contains(sslModes, p.SSLMode) {
		errs = append(errs, errors.New("POSTGRES_SSLMODE is not valid. Use disable, allow, prefer, require, verify-ca or verify-full."))
	}
	return errs
}

// Redis holds the settings of the connection to Redis.
// The cache and the job queue use different databases of the same Redis.
type Redis struct {
	Host     string `env:"REDIS_HOST" envDefault:"localhost"`
	Port     int    `env:"REDIS_PORT" envDefault:"5652"`
	Password string `env:"REDIS_PASSWORD"`
	CacheDB  int    `env:"REDIS_CACHE_DB" envDefault:"0"`
	QueueDB  int    `env:"REDIS_QUEUE_DB" envDefault:"1"`
}

// Addr gives the address of Redis in the form host:port.
func (r Redis) Addr() string {
	return net.JoinHostPort(r.Host, strconv.Itoa(r.Port))
}

func (r Redis) validate() []error {
	errs := checkPort("REDIS_PORT", r.Port)
	for _, d := range []struct {
		name string
		db   int
	}{{"REDIS_CACHE_DB", r.CacheDB}, {"REDIS_QUEUE_DB", r.QueueDB}} {
		if d.db < 0 {
			errs = append(errs, fmt.Errorf("%s is not valid. Use a number of 0 or more.", d.name))
		}
	}
	if r.CacheDB == r.QueueDB {
		errs = append(errs, errors.New("REDIS_CACHE_DB and REDIS_QUEUE_DB have the same value. Use different numbers."))
	}
	return errs
}

// Load reads the settings of the API from the environment of the process.
func Load() (Config, error) {
	return LoadFrom(env.ToMap(os.Environ()))
}

// LoadFrom reads the settings of the API from the given variables.
// It ignores the environment of the process, also when environment is nil.
func LoadFrom(environment map[string]string) (Config, error) {
	cfg, err := parse[Config](environment)
	if err != nil {
		return Config{}, err
	}
	errs := checkPort("API_PORT", cfg.APIPort)
	errs = append(errs, cfg.Postgres.validate()...)
	errs = append(errs, cfg.Redis.validate()...)
	if cfg.Docs.Enabled && cfg.Docs.AuthEnabled && (cfg.Docs.Username == "" || cfg.Docs.Password == "") {
		errs = append(errs, errors.New("The authentication of the API docs is on, "+
			"but API_DOCS_USERNAME or API_DOCS_PASSWORD is empty. "+
			"Set both variables. Or set API_DOCS_AUTH_ENABLED to false."))
	}
	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// LoadMigrate reads the settings of the migrate command from the environment of the process.
func LoadMigrate() (Migrate, error) {
	return LoadMigrateFrom(env.ToMap(os.Environ()))
}

// LoadMigrateFrom reads the settings of the migrate command from the given variables.
// It ignores the environment of the process, also when environment is nil.
func LoadMigrateFrom(environment map[string]string) (Migrate, error) {
	cfg, err := parse[Migrate](environment)
	if err != nil {
		return Migrate{}, err
	}
	if err := errors.Join(cfg.Postgres.validate()...); err != nil {
		return Migrate{}, err
	}
	return cfg, nil
}

// LoadWorker reads the settings of the worker from the environment of the process.
func LoadWorker() (Worker, error) {
	return LoadWorkerFrom(env.ToMap(os.Environ()))
}

// LoadWorkerFrom reads the settings of the worker from the given variables.
// It ignores the environment of the process, also when environment is nil.
func LoadWorkerFrom(environment map[string]string) (Worker, error) {
	cfg, err := parse[Worker](environment)
	if err != nil {
		return Worker{}, err
	}
	errs := append(cfg.Postgres.validate(), cfg.Redis.validate()...)
	if cfg.Concurrency < 1 {
		errs = append(errs, errors.New("WORKER_CONCURRENCY is not valid. Use a number of 1 or more."))
	}
	if err := errors.Join(errs...); err != nil {
		return Worker{}, err
	}
	return cfg, nil
}

func parse[T any](environment map[string]string) (T, error) {
	if environment == nil {
		environment = map[string]string{}
	}
	cfg, err := env.ParseAsWithOptions[T](env.Options{Environment: environment})
	if err != nil {
		var zero T
		return zero, nameVariables(err, reflect.TypeFor[T]())
	}
	return cfg, nil
}

func checkPort(name string, port int) []error {
	if port < 1 || port > 65535 {
		return []error{fmt.Errorf("%s is not valid. Use a number from 1 to 65535.", name)}
	}
	return nil
}

// nameVariables changes the field names in the parse errors of the env library
// to the names of the environment variables of the settings type.
func nameVariables(err error, settings reflect.Type) error {
	var aggregate env.AggregateError
	if !errors.As(err, &aggregate) {
		return err
	}
	errs := make([]error, 0, len(aggregate.Errors))
	for _, e := range aggregate.Errors {
		var parseErr env.ParseError
		if errors.As(e, &parseErr) {
			name, _ := variableName(settings, parseErr.Name)
			e = fmt.Errorf("%s is not valid. Correct the value. Cause: %w", name, parseErr.Err)
		}
		errs = append(errs, e)
	}
	return errors.Join(errs...)
}

// variableName finds the environment variable of a field in a struct, or in a struct inside it.
// shortcut: the first field with this name wins, so rename a field when 2 nested structs use the same name.
func variableName(t reflect.Type, field string) (string, bool) {
	for f := range t.Fields() {
		if f.Name == field && f.Tag.Get("env") != "" {
			return f.Tag.Get("env"), true
		}
		if f.Type.Kind() == reflect.Struct {
			if name, ok := variableName(f.Type, field); ok {
				return name, true
			}
		}
	}
	return field, false
}
