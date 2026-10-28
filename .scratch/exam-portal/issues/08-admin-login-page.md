# 08: Admin login page

**Parent:** `.scratch/exam-portal/spec.md`

**What to build:** An Admin logs in on the frontend and sees the Admin area. A page that needs a session sends a person without a session to the login page.

**Blocked by:** 01 (Remove the User example), 02 (Admin login API)

**Status:** ready-for-agent

- [x] `make gen` makes the types for the session routes from the spec.
- [x] The page `/admin/login` has an email address field and a password field. Each field has a label.
- [x] A successful login opens `/admin`. The page shows the email address of the Admin and a "Log out" button.
- [x] After a 401 or a 429 response, the page shows the `detail` text of the problem response.
- [x] A guard sends a person without a session from each `/admin/*` page, except the login page, to `/admin/login`.
- [x] "Log out" calls `DELETE` on the session and opens the login page.
- [x] A person can use the form with only the keyboard.
- [x] All page text obeys the STE rule.
- [x] Unit tests use MSW and fixture builders. They cover the success, the 401 response, the 429 response and the guard.
