# 03: Question bank API

**Parent:** `.scratch/exam-portal/spec.md`

**What to build:** An Admin makes, lists, reads, changes and deletes Questions through the API. The rules that need an Exam or an Attempt come in tickets 04 and 05.

**Blocked by:** 02 (Admin login API)

**Status:** ready-for-agent

- [x] The spec defines `POST` and `GET /api/v1/admin/questions`, and `GET`, `PUT` and `DELETE /api/v1/admin/questions/{id}`, before the handlers exist.
- [x] The `exam` context has the Question and the Option in its domain. A Question has a text of 1 to 2000 characters and 2 to 6 Options. Each Option has 1 to 500 characters. Exactly one Option is correct.
- [x] A Question that breaks a rule returns 422. A body that the API cannot parse returns 400.
- [x] A successful `POST` returns 201 with the Question: ID, text and the Options in the given order, with the correct one marked.
- [x] The list returns all Questions, newest first, with no pagination.
- [x] An unknown ID returns 404. An ID that is not a UUID returns 400.
- [x] `PUT` replaces the text and all Options. `DELETE` returns 204.
- [x] All routes need an Admin session and return 401 without it.
- [x] A migration makes the `questions` and `options` tables. The IDs are UUIDv7 values that the domain makes.
- [x] `make gen` makes the port mocks. Fixture builders make valid Questions.
- [x] Unit tests cover the domain and the use cases with mocks. E2E tests cover each status code and check each response against the spec.
