# 05: Candidate start API

**Parent:** `.scratch/exam-portal/spec.md`

**What to build:** A Candidate opens an Exam link, gives a name and an email address, and starts an Attempt. A Candidate with an open Attempt resumes it. This ticket also adds the Exam lock rules, because they need the Attempts. See ADR 0004.

**Blocked by:** 04 (Exam API)

**Status:** ready-for-agent

- [x] The spec defines `GET /api/v1/exam-links/{token}`, `POST /api/v1/exam-links/{token}/attempts` and `GET /api/v1/attempts/{id}` before the handlers exist. These routes need no session.
- [x] `GET` on the Exam link returns the title, the Exam window, the Time limit and the number of Questions. It does not return the Questions. An unknown token returns 404.
- [x] A start with a name of 1 to 100 characters and a valid email address of up to 254 characters works. The API trims both and changes the email address to lowercase. Other input returns 422.
- [x] A start before the open time returns 409 with "not open yet". A start after the close time returns 409 with "closed". The `detail` text states which.
- [x] A new email address returns 201 with the Attempt. The Attempt ID is a random UUIDv4. The domain makes it.
- [x] An email address with an open Attempt returns 200 with that Attempt.
- [x] An email address with a submitted or timed-out Attempt returns 409. The response has no Score.
- [x] An open Attempt response has the Exam title, `expiresAt`, the Questions and Options in the Admin's order, and the saved Answers. No Option in any response has a `correct` field.
- [x] `GET /attempts/{id}` returns the same response for an open Attempt, and only the status for another Attempt. An unknown ID returns 404. An ID that is not a UUID returns 400.
- [x] A migration makes the `attempts` table, with a unique index on the Exam and the email address, and the `answers` table.
- [x] After the first Attempt of an Exam, `PUT` on the Exam works only when it changes the close time, and `DELETE` returns 409. Other changes return 409.
- [x] After an Attempt uses a Question, `PUT` on that Question returns 409.
- [x] Unit tests use a fixed clock. E2E tests check each status code and check the raw JSON for a missing `correct` field.
