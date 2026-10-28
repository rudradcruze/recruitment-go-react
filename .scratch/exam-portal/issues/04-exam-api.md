# 04: Exam API

**Parent:** `.scratch/exam-portal/spec.md`

**What to build:** An Admin makes and manages Exams through the API. Each Exam has an Exam window, a Time limit, a list of Questions and an Exam link. The Admin can regenerate the Exam link.

**Blocked by:** 03 (Question bank API)

**Status:** ready-for-agent

- [x] The spec defines `POST` and `GET /api/v1/admin/exams`, `GET`, `PUT` and `DELETE /api/v1/admin/exams/{id}`, and `POST /api/v1/admin/exams/{id}/exam-link`, before the handlers exist.
- [x] The domain checks these rules: a title of 1 to 200 characters, 1 to 200 different Questions, a Time limit of 1 to 480 minutes, and an open time before the close time.
- [x] An Exam that breaks a rule returns 422. A Question ID that does not exist returns 422.
- [x] A successful `POST` returns 201 with the Exam: ID, title, open time, close time, Time limit, the Questions in the given order, and the Exam link token.
- [x] The token has 22 random base62 characters from `crypto/rand`. A unique index keeps each token different.
- [x] `POST .../exam-link` makes a new token and returns the Exam. The old token is not valid.
- [x] The list returns all Exams, newest first, with the title, the Exam window, the Time limit and the number of Questions.
- [x] `PUT` replaces all fields of the Exam. `DELETE` returns 204. An unknown ID returns 404.
- [x] A Question that an Exam uses cannot be deleted. The Question `DELETE` returns 409.
- [x] All routes need an Admin session and return 401 without it.
- [x] A migration makes the `exams` table and the table that holds the Questions of each Exam in their order.
- [x] Unit tests cover the domain and the use cases with mocks. E2E tests cover each status code and check each response against the spec.
