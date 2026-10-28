# 12: Full stack and browser tests

**Parent:** `.scratch/exam-portal/spec.md`

**What to build:** The full stack in Docker Compose runs the portal. A browser test runs the full flow from the Admin to the Candidate, and back to the Admin.

**Blocked by:** 10 (Exam pages), 11 (Candidate pages)

**Status:** ready-for-agent

- [x] `compose.yaml` and `compose.e2e.yaml` give `ADMIN_EMAIL` and `ADMIN_PASSWORD` to the `migrate` service. The browser tests use test values that are not secret.
- [x] After `docker compose up`, an Admin logs in at `localhost:5651/admin/login`.
- [x] One browser test does these steps:
  - An Admin logs in and makes 2 Questions.
  - The Admin makes an Exam with an Exam window around the current time, and copies the Exam link.
  - A Candidate, in a new browser context, opens the link, enters a name and an email address, selects Options and submits.
  - The Candidate sees the Score.
  - The Admin opens the results and sees the row of the Candidate.
  - The Candidate opens the link again with the same email address and sees "This Exam is already submitted."
- [x] `README.md` tells how to set `ADMIN_EMAIL` and `ADMIN_PASSWORD`, how to start the app and how to log in.
- [x] `.claude/rules/architecture.md` describes the `admin` and `exam` contexts and the Admin variables.
- [x] `make test`, `make test-e2e` and `make test-web-e2e` pass.
