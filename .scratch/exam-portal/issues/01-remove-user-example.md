# 01: Remove the User example

**Parent:** `.scratch/exam-portal/spec.md`

**What to build:** Delete the template User example, so that the app has no User. The worker and Asynqmon stay in the template, with no task handler.

**Blocked by:** None

**Status:** ready-for-agent

- [x] The spec has no `/users` paths and no User schemas.
- [x] `backend/internal/user/` does not exist, and no code imports it. The Welcome email job and its worker handler are gone.
- [x] A new migration drops the `users` table. Its down step makes the table again. `00001_create_users.sql` does not change.
- [x] The worker starts, checks the Redis connection and has no task handler.
- [x] The frontend has no Registration page, no User page and no route to them. The Home page shows the name of the app and no link.
- [x] The tests, mocks and fixtures of the deleted code are deleted. A browser test opens the Home page and checks the name of the app.
- [x] `README.md` and `docs/` do not name the User example. The ADRs do not change.
- [x] `make test`, `make test-e2e` and `make test-web-e2e` pass.
