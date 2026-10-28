# 11: Candidate pages

**Parent:** `.scratch/exam-portal/spec.md`

**What to build:** A Candidate opens the Exam link, enters a name and an email address, answers the Questions and submits. The page shows the Score one time.

**Blocked by:** 05 (Candidate start API), 06 (Answer and submit API)

**Status:** ready-for-agent

- [x] The page `/e/:token` shows the Exam title, the Exam window, the Time limit and a form with a name field and an email address field. Each field has a label.
- [x] A start opens `/attempts/:id`. A 409 response shows the `detail` text of the problem response. A 404 response shows "The Exam link is not valid."
- [x] The page `/attempts/:id` shows all Questions on one page, with the Options as radio buttons in a group that has the Question as its label.
- [x] The page shows a countdown to `expiresAt` in an element with `role="timer"`.
- [x] A selected Option is sent at once. The page shows when the Answer is saved. A failed save shows the `detail` text.
- [x] A refresh shows the saved Answers again.
- [x] A "Submit" button sends the Submit. When the countdown reaches 0, the page sends the Submit by itself.
- [x] After the Submit, the page shows the Score, for example "Your Score is 7 of 10."
- [x] A visit to a finished Attempt shows "This Exam is already submitted." The page does not show the Score or the Questions.
- [x] A person can use the pages with only the keyboard.
- [x] All page text obeys the STE rule.
- [x] Unit tests use MSW and fixture builders. They cover the start, the 409 response, an Answer, the refresh, the Submit, the automatic Submit and a finished Attempt.
