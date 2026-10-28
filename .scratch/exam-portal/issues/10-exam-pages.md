# 10: Exam pages

**Parent:** `.scratch/exam-portal/spec.md`

**What to build:** An Admin schedules an Exam, copies the Exam link, regenerates the link, and reads the results on the frontend.

**Blocked by:** 04 (Exam API), 07 (Results API), 08 (Admin login page)

**Status:** ready-for-agent

- [x] The Admin area has a link to the Exams. The page `/admin/exams` lists the Exams with their title, Exam window, Time limit and number of Questions. The page has an empty state.
- [x] The pages `/admin/exams/new` and `/admin/exams/:id` show one form with a title, a list of Questions to select, an open time, a close time and a Time limit in minutes. Each field has a label.
- [x] The Exam keeps the Questions in the order that the Admin selects them. The form shows the selected Questions in a numbered list.
- [x] The time fields use the timezone of the browser. The page sends the times in UTC.
- [x] The Exam page shows the full Exam link, `<origin>/e/<token>`, and a "Copy link" button.
- [x] A "Regenerate link" button asks the Admin to confirm. The text says that the old link stops working.
- [x] The page `/admin/exams/:id/results` shows a table of the Attempts: name, email address, status, Score, start time and finish time. It has an empty state.
- [x] A 409 or 422 response shows the `detail` text of the problem response. A delete asks the Admin to confirm.
- [x] A person can use the pages with only the keyboard.
- [x] All page text obeys the STE rule.
- [x] Unit tests use MSW and fixture builders. They cover a save, the regenerate step, the results table and a 409 response.
