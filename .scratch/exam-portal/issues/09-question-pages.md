# 09: Question pages

**Parent:** `.scratch/exam-portal/spec.md`

**What to build:** An Admin sees the Questions, and makes, changes and deletes a Question on the frontend.

**Blocked by:** 03 (Question bank API), 08 (Admin login page)

**Status:** ready-for-agent

- [x] The Admin area has a link to the Questions. The page `/admin/questions` lists the Questions with their text and the number of Options. The page has an empty state.
- [x] The pages `/admin/questions/new` and `/admin/questions/:id` show one form. It has a text field and 2 to 6 Option fields. The Admin can add and remove an Option.
- [x] The Admin marks the correct Option with a radio button. Each field has a label.
- [x] A save sends the Question through the typed client, then opens the list.
- [x] A delete asks the Admin to confirm. A 409 or 422 response shows the `detail` text of the problem response.
- [x] A person can use the pages with only the keyboard.
- [x] All page text obeys the STE rule.
- [x] Unit tests use MSW and fixture builders. They cover the list, a save, a delete and a 409 response.
