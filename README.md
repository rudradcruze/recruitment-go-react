# go-react-template

This repo is an online exam portal. An Admin writes multiple-choice Questions and schedules an Exam. A Candidate takes the Exam through a public link.

The terms are in `CONTEXT.md`. The rules for the code are in `.claude/rules/`. The tickets are in `.scratch/`.

## Requirements

- Go
- Node.js and pnpm
- Docker with Docker Compose

## Start the application for development

1. Run `make install`.
2. Copy `.env.example` to `.env`.
3. Set `POSTGRES_PASSWORD` and `ADMIN_PASSWORD` in `.env`.
4. Run `make up`.
5. Run `make migrate-up`.
6. Run `make dev`.
7. Open `http://localhost:5651/admin/login`.

## Start the full stack

Stop `make dev` first. Then run `make stack-up`. The web service listens on `http://localhost:5651`.

## The first Admin

An Admin cannot sign up. The `migrate` command makes the first Admin from two settings in `.env`:

| Setting | Meaning |
| --- | --- |
| `ADMIN_EMAIL` | The email address of the Admin. |
| `ADMIN_PASSWORD` | The password of the Admin. Use 12 to 72 bytes. |

- Set both values before you run `make migrate-up` or `make stack-up`. The command stops with an error when a value is not valid.
- The command does nothing when an Admin with this email address exists. It does not change the password of that Admin.
- An Admin who enters a wrong password 5 times in 15 minutes must wait until the 15 minutes end.

## The demo Admin on the login page

The login page shows a demo Admin when `VITE_DEMO_ADMIN_EMAIL` and `VITE_DEMO_ADMIN_PASSWORD` have a value in `.env`.

- Vite puts these values in the files of the page, so everyone can read them.
- Use the same values as `ADMIN_EMAIL` and `ADMIN_PASSWORD` only for a demo.
- Do not use a real password. Leave both settings empty in deployment, and the page shows no demo Admin.
- After you change a value, restart `make dev`. For `make stack-up`, the build of the web image reads the values.

## How an Exam works

1. The Admin logs in and adds Questions to the Question bank. Each Question has 2 to 6 Options, and one Option is correct.
2. The Admin makes an Exam. The Admin selects the Questions, the Exam window and the Time limit.
3. The Admin copies the Exam link from the page of the Exam, and gives it to the Candidates.
4. The Candidate opens the link, enters a name and an email address, and starts the Exam.
5. The Candidate selects an Option for each Question. The application saves each Answer at once.
6. The Candidate submits the Exam, or waits until the Time limit ends. The Candidate sees the Score one time.
7. The Admin reads the results on the page of the Exam.

Each email address has one Attempt for each Exam. The application does not check the email address (see `docs/adr/0004-candidate-email-is-not-verified.md`).

## Tests

| Command | Tests |
| --- | --- |
| `make test` | The unit tests of the backend and the frontend. |
| `make test-e2e` | The backend E2E tests, with real Postgres and Redis in containers. |
| `make test-web-e2e` | The browser tests, against the full stack on the port 5655. |

## API docs

Set `API_DOCS_ENABLED=true` in `.env`. Then open `http://localhost:5653/docs`.
