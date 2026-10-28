# 06: Answer and submit API

**Parent:** `.scratch/exam-portal/spec.md`

**What to build:** A Candidate saves an Answer each time they select an Option, and submits the Attempt. The API enforces the Time limit and returns the Score.

**Blocked by:** 05 (Candidate start API)

**Status:** ready-for-agent

- [x] The spec defines `PUT /api/v1/attempts/{id}/answers/{questionId}` and `POST /api/v1/attempts/{id}/submission` before the handlers exist.
- [x] `PUT` takes an Option ID and returns 204. A second `PUT` for the same Question replaces the Answer.
- [x] A Question that is not in the Exam, or an Option that is not an Option of that Question, returns 422.
- [x] A `PUT` after `expiresAt`, or after the Submit, returns 409. An unknown Attempt returns 404.
- [x] `POST` stores the submit time and returns 200 with the number of correct Answers and the number of Questions. A skipped Question counts as 0 points.
- [x] `POST` after `expiresAt` also works if the Attempt has no submit time. The Score uses only the saved Answers. A second `POST` returns 409.
- [x] The status of an Attempt is `open`, `submitted` or `timed out`, as the spec defines. The API works it out when it reads the Attempt. No background job runs.
- [x] Unit tests use a fixed clock and cover the edge of `expiresAt`. E2E tests make an Attempt with a start time in the past through the fixtures, and cover each status code.
