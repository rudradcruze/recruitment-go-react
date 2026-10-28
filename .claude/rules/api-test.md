---
paths:
  - "backend/api-test/**"
  - "backend/.mockery.yaml"
---

# Backend test rules

All backend tests are in `backend/api-test/`. Write each test before the code that it tests (see the constitution).

## Layout

```text
backend/api-test/
├── unit/<context>/<layer>_test.go    unit tests, for example unit/exam/domain_test.go and unit/exam/app_test.go
├── unit/fixtures/                    tests of the builders in fixtures/
├── e2e/<context>/<feature>_test.go   E2E tests
├── e2e/testenv/                      helpers for E2E tests: containers and one database for each test
├── e2e/testapi/                      starts the real API and checks the responses against the spec
├── fixtures/                         builders for test data
└── mocks/                            mocks from mockery (generated)
```

The tests are in different packages from the code. They use only exported identifiers, so they test behaviour through the public interface (see ADR 0003). When a test needs an unexported identifier, test the exported behaviour that uses it. Export an identifier only when production code needs it.

## Conventions

- Name each test `Test<Type>_<Method>_<Behaviour>`, for example `TestCreateExam_Execute_RejectsUnknownQuestion`.
- Test one behaviour in each test. Use a table-driven test when the cases have the same steps.
- Use `require` for preconditions and `assert` for checks. Both come from Testify.
- Call `t.Parallel()` in unit tests. E2E tests run one after the other.
- Run unit tests with `make test`, and E2E tests with `make test-e2e`. For coverage, add `-coverpkg=./...`.

## Mocks

- A unit test replaces every port with a mock. A unit test does not touch Postgres, Redis or the network.
- mockery generates the mocks from the port interfaces into `mocks/`. Its configuration is `backend/.mockery.yaml`. Run `make gen` after a port changes. Do not edit the files in `mocks/`.
- Make each mock with its `NewMock<Port>(t)` constructor. The mock then checks its expectations when the test ends.
- Set expectations with `.EXPECT()`, and give exact arguments. Use `mock.Anything` only for a `context.Context`. When an argument has a random value, for example a new ID, match its other fields with `mock.MatchedBy`.

## Fixtures

- `fixtures/` has a builder for each entity, for example `fixtures.Question(fixtures.WithText("What is 2 + 2?"))`.
- A builder gives a valid object with default values. Each call gives unique values for unique fields, for example the title.
- Unit tests use the builders to make domain objects. E2E tests use the save helpers, for example `fixtures.SaveQuestion(t, api.Questions)`, which save through the repository.

## E2E isolation

- Each E2E file starts with `//go:build e2e`.
- `TestMain` in each E2E package calls `os.Exit(testenv.Main(m))`. `testenv.Main` starts Postgres and Redis with testcontainers-go on random ports. It makes a template database and runs the goose migrations on it one time. A package can set `testenv.MigrationsDir` before it calls `Main`, to use its own test migrations.
- Each test calls `testenv.Begin(t)`. The helper makes a new database from the template, with the name `test_<random>`. It drops the database in `t.Cleanup`, also when the test fails.
- `testenv.BeginEmpty(t)` makes a database without tables. Use it to test the migrations, because the template already has applied migrations.
- `testenv.Begin` also removes all Redis data (`FLUSHALL`) before the test starts.
- Each test sends real HTTP requests to the real Gin engine, with no mocks.
- Start the API with `testapi.Start(t)`. It builds the real handler with `bootstrap.NewAPI`, on the database of the test. Send requests with `api.Get`, `api.PostJSON` or `api.Do`: they check each response against `backend/api/openapi.yaml` with `pb33f/libopenapi-validator`, so a test cannot skip the check. `api.LogIn(t)` saves an Admin and logs in through the API, and gives the session token for `api.GetAs` and `api.JSONAs`. `api.DB` and `api.Redis` read the data of the test. `api.Logs()` gives the log lines of the API. `api.BreakDatabase`, `api.BreakCache` and `api.BreakQueue` make the next call to Postgres, to the cache or to the queue fail, to test an unexpected error. `api.QueueInspector` reads the job queue, `api.Enqueue` adds a task, and `api.StartWorker(t)` starts a worker on the data of the test.
- When `TestMain` ends, it stops the containers. This also removes the template database.
