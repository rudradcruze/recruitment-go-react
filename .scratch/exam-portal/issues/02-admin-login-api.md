# 02: Admin login API

**Parent:** `.scratch/exam-portal/spec.md`

**What to build:** An Admin logs in with an email address and a password, and gets a session. The `migrate` service creates the first Admin. One guard protects all Admin routes.

**Blocked by:** 01 (Remove the User example)

**Status:** ready-for-agent

- [x] The spec defines `POST`, `GET` and `DELETE /api/v1/admin/session`, with their responses, before the handlers exist.
- [x] A migration makes the `admins` table with a unique index on the email address. The ID is a UUIDv7 that the domain makes. The table keeps a bcrypt hash of the password.
- [x] `cmd/migrate up` creates one Admin from `ADMIN_EMAIL` and `ADMIN_PASSWORD`. It does nothing when an Admin with that email address exists.
- [x] `cmd/migrate up` stops with a clear error when `ADMIN_EMAIL` is empty, or when the password does not have 12 to 72 bytes.
- [x] A login with the correct email address and password returns 204. The response sets an HttpOnly cookie with `SameSite=Lax` and a life of 8 hours.
- [x] Redis keeps a hash of the session token for 8 hours.
- [x] A login with a wrong email address or a wrong password returns 401. The text is the same for both causes.
- [x] After 5 failed logins for one email address in 15 minutes, the API returns 429 with a `Retry-After` header. Redis keeps the count, and it expires after 15 minutes.
- [x] `GET` returns the ID and the email address of the Admin, or 401. `DELETE` removes the session and returns 204.
- [x] One guard in `bootstrap` checks the session for every `/api/v1/admin/*` route except login. It asks the `admin` `app` package. A request without a valid session gets 401.
- [x] `.env.example` lists `ADMIN_EMAIL` and `ADMIN_PASSWORD` with no secret. `compose.yaml` gives them to the `migrate` service.
- [x] Unit tests cover the domain and the use cases with mocks. E2E tests with real Postgres and Redis cover each status code and check each response against the spec.
