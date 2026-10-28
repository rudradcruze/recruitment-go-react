# Architecture

The backend is Go with Gin and Domain-Driven Design (DDD). The frontend is React with TypeScript on the Vite+ toolchain. The ADRs in `docs/adr/` give the reasons for the main decisions.

## Repo layout

```text
/
├── backend/                   Go module
│   ├── api/openapi.yaml       API contract (OpenAPI 3.2.0)
│   ├── cmd/api/               HTTP server
│   ├── cmd/worker/            Asynq worker
│   ├── cmd/migrate/           applies or reverts the migrations
│   ├── internal/<context>/    one folder for each bounded context
│   ├── internal/bootstrap/    connects the contexts and builds the HTTP handler
│   ├── internal/platform/     shared infrastructure
│   ├── migrations/            goose SQL migrations
│   └── api-test/              all backend tests
├── frontend/                  React app (pnpm workspace package)
│   ├── src/
│   └── test/                  all frontend tests
├── compose.yaml               full stack, as deployed
├── compose.dev.yaml           development parts for compose.yaml
├── compose.e2e.yaml           settings of the browser tests for compose.yaml
├── scripts/test-web-e2e.sh    runs the browser tests in a Compose project of their own
├── Makefile
├── package.json               root of the pnpm workspace
├── pnpm-workspace.yaml        workspace packages and the Vite+ catalog
├── pnpm-lock.yaml
└── .env.example
```

## Bounded contexts

| Context | Contents |
| --- | --- |
| `admin` | The Admin, the login, the session in Redis and the guard for the Admin routes. |
| `exam` | The Question bank, the Exams, the Attempts, the Answers and the Score. |

- `bootstrap` gives the guard of `admin` to the Admin routes of `exam`, as a Gin handler. `exam` does not import `admin`.
- A Candidate has no context of its own. A Candidate exists only as the name and the email address of an Attempt.

## Bounded context layers

Each bounded context in `backend/internal/<context>/` has the same folders:

| Folder | Contents | Can import |
| --- | --- | --- |
| `domain/` | Entities, value objects, domain errors | Standard library, small pure libraries (for example `github.com/google/uuid`) |
| `app/` | One use case for each operation, and the port interfaces that the use cases call | `domain` |
| `infra/` | Adapters for the ports: GORM repository, Redis cache, Asynq enqueuer | `app`, `domain`, `platform` |
| `http/` | Gin handlers, and the mapping between HTTP and the use cases | `app`, `domain`, `platform` |
| `jobs/` | Asynq task handlers that call use cases | `app`, `domain`, `platform` |

- The `app` layer declares each port as a Go interface where it uses the port. The `infra` layer implements the ports.
- A bounded context uses a different bounded context only through the `app` package of that context.
- `internal/platform/` holds the shared infrastructure: `config`, `logger`, `database`, `cache`, `queue`, `problem`, `httpjson` and `server`. It imports no bounded context.
- `internal/bootstrap/` is the only place that knows all contexts. It builds the adapters and the use cases, and gives the routes to `server.New`. `cmd/api` and the E2E tests both call it.
- The Go package of the folder `http/` has the name `<context>http`, for example `userhttp`, so that it does not hide `net/http`.
- `cmd/api` and `cmd/worker` connect all parts with plain constructor calls, without a dependency injection framework.

## Errors

- The domain returns typed errors, for example `ErrEmailTaken`.
- `infra` changes database errors into domain errors.
- `http` changes domain errors into RFC 9457 problem details. See `api.md`.

## Background jobs

The application has no background task now. The worker starts, checks Redis and has no handler. Add a handler in `bootstrap.NewWorkerMux` when a task is needed.

- Asynq runs the jobs. The queue uses Redis DB 1.
- Task type names have the form `<context>:<action>`, for example `user:send_welcome_email`.
- Task payloads are JSON.
- Each task has a maximum of 3 retries and a timeout of 1 minute for each try. After the last retry, Asynq archives the task. A task that cannot succeed later, for example a payload that is not valid, returns `asynq.SkipRetry`.
- The queue delivers a task at least once. A mailer adapter must tolerate a repeated call.
- The worker logs each task that fails (`queue.NewErrorHandler`), and checks the Redis connection when it starts.
- Enqueue a task only after the database transaction commits. If the enqueue fails, log the error and let the request succeed.
- When a job must not be lost, use a transactional outbox, and record that decision in an ADR.

## Logs

- Use `log/slog` with the JSON handler. Give the `*slog.Logger` to each part through its constructor.
- The domain does not write logs.

## Configuration

- `internal/platform/config` reads all environment variables into one typed struct with `caarlos0/env`.
- The config checks all values at startup. An incorrect value stops the process with a clear error.
- `ADMIN_EMAIL` and `ADMIN_PASSWORD` are settings of the `migrate` command only. After `migrate up`, the command makes the first Admin, and it stops with a clear error when a value is not valid.

## Ports

| Port | Service |
| --- | --- |
| 5650 | Asynqmon dashboard (development only) |
| 5651 | Web: Vite dev server, or nginx in deployment |
| 5652 | Redis (host side) |
| 5653 | API, with the API docs at `/docs` |
| 5654 | Postgres (host side) |
| 5655 | Web in the frontend E2E stack |

Inside Docker, Postgres and Redis use the ports of their images (5432 and 6379). The worker has no port.

## Deployment

Docker Compose runs the stack.

- `compose.yaml` holds the full stack, as deployed: `postgres`, `redis`, `migrate`, `api`, `worker` and `web`. Postgres and Redis have no host ports. Run it with `make stack-up`. Stop `make dev` first, because both use the host ports 5651 (web) and 5653 (API, on the local machine only). `make up` and `make stack-up` use the same Compose project and the same data volumes.
- The API, the worker and the migrate command are in one image (`backend/Dockerfile`). Each service chooses its command with `entrypoint`.
- The one-time `migrate` service runs `cmd/migrate up`. The `api` and the `worker` start only after it ends without an error. If a migration fails, they do not start.
- The `web` service starts after the `api` is healthy. The API has `GET /healthz`, outside the OpenAPI spec and outside `/api/v1`. It is public and does not check Postgres or Redis.
- `compose.dev.yaml` adds the development parts to `compose.yaml`: host ports for Postgres and Redis, and Asynqmon.
- In development, `make up` starts only `postgres`, `redis` and `asynqmon` in Docker. `make dev` runs the API, the worker and the Vite dev server on the host.
- In deployment, nginx in the `web` container (`frontend/nginx.conf`) serves the built frontend and sends `/api`, `/docs` and `/openapi.yaml` to the API. It listens on 8080 inside the container, because a process without special rights cannot use port 80, and Compose publishes it on 5651. nginx finds the `api` service again every 10 seconds, so it follows a new container. A request body of more than 2 MiB gets a 413 problem detail from nginx, and the API answers the bodies of more than 1 MiB below that. In development, the Vite proxy does this. The browser always sees one origin, so the API needs no CORS settings.
