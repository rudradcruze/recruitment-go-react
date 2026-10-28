# 07: Results API

**Parent:** `.scratch/exam-portal/spec.md`

**What to build:** An Admin reads the Attempts of an Exam through the API.

**Blocked by:** 06 (Answer and submit API)

**Status:** ready-for-agent

- [x] The spec defines `GET /api/v1/admin/exams/{id}/attempts` before the handler exists.
- [x] Each row has the name, the email address, the status, the number of correct Answers, the number of Questions, the start time and the finish time.
- [x] The finish time is the submit time. For a timed-out Attempt with no submit time, it is `expiresAt`. For an open Attempt, it is null.
- [x] The Score of an open Attempt is null. The Score of a timed-out Attempt uses its saved Answers.
- [x] The rows are in the order of the start time. There is no pagination.
- [x] The response has no Attempt ID, because the Attempt ID is the secret of the Candidate.
- [x] An unknown Exam returns 404. A request without a session returns 401.
- [x] Unit tests cover the use case with mocks. E2E tests cover an open, a submitted and a timed-out Attempt, and check the response against the spec.
