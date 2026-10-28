# Exam portal (parent issue)

**Status:** ready-for-agent

## Goal

Build a basic exam portal on the template. An Admin logs in, writes multiple-choice Questions and schedules an Exam. The Admin shares the Exam link. A Candidate opens the link, enters a name and an email address, and takes the Exam. The app saves the Attempt and shows the Score.

The rules in `.claude/rules/` and the ADRs in `docs/adr/` apply to all tickets. The terms in `CONTEXT.md` apply to all text. ADR 0004 gives the reason for the Attempt rule. Ticket 01 deletes the template User example.

## Rules

The routes below are the first design. After ticket 02 starts, `backend/api/openapi.yaml` is the contract.

### Structure

- The backend has two bounded contexts: `admin` and `exam`.
- The `exam` context holds the Question bank, the Exams and the Attempts.
- The `exam` context asks the `admin` context only one question: is this session an Admin? It asks through the `app` package of `admin`.
- The frontend is one React app. The routes are `/admin/*`, `/e/:token` and `/attempts/:id`.
- The API and the database keep times in UTC (ISO 8601). Each browser shows times in its own timezone.

### Admin and login

- The `migrate` service creates one Admin from `ADMIN_EMAIL` and `ADMIN_PASSWORD` after the migrations. It does nothing when an Admin with that email address exists.
- An Admin logs in with an email address and a password. The password has 12 to 72 bytes. The database keeps a bcrypt hash.
- A login sets an HttpOnly cookie that holds an opaque session token. Redis keeps a hash of the token for 8 hours.
- Redis counts the failed logins of each email address. After 5 failures in 15 minutes, the API returns 429 until the count expires.
- All `/api/v1/admin/*` routes, except login, need a valid session. Without it, they return 401.

### Question

- The text has 1 to 2000 characters.
- A Question has 2 to 6 Options. Each Option has 1 to 500 characters. Exactly one Option is correct.
- An Admin cannot delete a Question that an Exam uses. An Admin cannot change a Question that an Attempt used. Both return 409.

### Exam

- The title has 1 to 200 characters. An Exam has 1 to 200 different Questions. The Time limit is 1 to 480 minutes. The open time is before the close time.
- The Exam keeps the Questions in the order that the Admin gives. The Options keep the order of the Question.
- The Exam link is a token of 22 random base62 characters. The Exam link URL is `/e/<token>`. An Admin can regenerate the token. The old token then gives 404.
- Before the first Attempt, the Admin can change every field and delete the Exam.
- After the first Attempt, the Admin can change only the close time. The Admin cannot delete the Exam. Both return 409.

### Attempt

- A Candidate can start an Attempt only in the Exam window. Before it, the API returns 409 with "not open yet". After it, the API returns 409 with "closed".
- The Candidate name has 1 to 100 characters. The email address is valid and has up to 254 characters. The API trims the name and the email address, and changes the email address to lowercase.
- A new email address starts a new Attempt. An email address with an open Attempt gets that Attempt. An email address with a finished Attempt gets 409, and the API gives no Score.
- The Attempt ID is a random UUIDv4. It is the only secret of the Attempt. No response to an Admin has it.
- `expiresAt` is the start time plus the Time limit.
- The API rejects an Answer after `expiresAt`, or after the Candidate submits. It returns 409.
- A Submit stores the submit time and returns the Score. It works after `expiresAt` also, because the Answers cannot change after that time. A second Submit returns 409.
- The status is `submitted` when the submit time is not after `expiresAt`. The status is `timed out` when the submit time is after `expiresAt`, or when there is no submit time and the time is after `expiresAt`. In all other cases, the status is `open`. No background job sets the status. The API works it out when it reads the Attempt.
- A response to a Candidate never says which Option is correct.
- Only an open Attempt gives its Questions. A finished Attempt gives only its status.
- The Score is the number of correct Answers and the number of Questions in the Exam.

### Results

- The Admin sees one row for each Attempt: name, email address, status, Score, start time and finish time.
- The rows are in the order of the start time. There is no pagination.

## Tickets

| # | Ticket | Blocked by |
| --- | --- | --- |
| 01 | [Remove the User example](issues/01-remove-user-example.md) | None |
| 02 | [Admin login API](issues/02-admin-login-api.md) | 01 |
| 03 | [Question bank API](issues/03-question-bank-api.md) | 02 |
| 04 | [Exam API](issues/04-exam-api.md) | 03 |
| 05 | [Candidate start API](issues/05-candidate-start-api.md) | 04 |
| 06 | [Answer and submit API](issues/06-answer-and-submit-api.md) | 05 |
| 07 | [Results API](issues/07-results-api.md) | 06 |
| 08 | [Admin login page](issues/08-admin-login-page.md) | 01, 02 |
| 09 | [Question pages](issues/09-question-pages.md) | 03, 08 |
| 10 | [Exam pages](issues/10-exam-pages.md) | 04, 07, 08 |
| 11 | [Candidate pages](issues/11-candidate-pages.md) | 05, 06 |
| 12 | [Full stack and browser tests](issues/12-full-stack-and-browser-tests.md) | 10, 11 |

## Waves

The tickets in one wave can be implemented at the same time. A ticket can start when all its blockers are merged. It does not have to wait for the full previous wave.

| Wave | Tickets (parallel) | Starts after |
| --- | --- | --- |
| 1 | 01 | None |
| 2 | 02 | 01 |
| 3 | 03, 08 | 02 |
| 4 | 04, 09 | 03 (09 also needs 08) |
| 5 | 05 | 04 |
| 6 | 06 | 05 |
| 7 | 07, 11 | 06 |
| 8 | 10 | 04, 07, 08 |
| 9 | 12 | 10, 11 |

The critical path is 01 → 02 → 03 → 04 → 05 → 06 → 07 → 10 → 12. A delay on this path delays the full project.

## Merge order

| Order | Ticket | Reason |
| --- | --- | --- |
| 1 | 01 | All tickets need a clean base. |
| 2 | 02 | All other tickets need the login. |
| 3 | 03 | It is on the critical path. |
| 4 | 08 | Rebase on 03. Both tickets change the spec or the generated types. |
| 5 | 04 | It is on the critical path. |
| 6 | 09 | It needs 03 and 08. |
| 7 | 05 | Rebase on 04. Both tickets change the `exam` context. |
| 8 | 06 | Rebase on 05. |
| 9 | 07 | Rebase on 06. |
| 10 | 11 | It needs 05 and 06. |
| 11 | 10 | It needs 04, 07 and 08. |
| 12 | 12 | It needs the full stack and all pages. |

## Merge rules

- Merge one ticket at a time, in the merge order.
- Before the merge, rebase the branch on `main`.
- When `main` has a spec change, run `make gen` after the rebase.
- Before the merge, all unit tests and E2E tests must pass.

## Done when

- [ ] All 12 tickets are merged.
- [x] `docker compose up` starts the full stack. An Admin logs in at `localhost:5651/admin/login`.
- [x] A Candidate takes an Exam through the Exam link and sees the Score.
- [x] `make test`, `make test-e2e` and `make test-web-e2e` pass.
