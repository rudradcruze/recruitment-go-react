# go-react-template

This repo is a template for a Go and React application. Its example application is an online exam portal. Admins write multiple-choice Questions and schedule Exams. Candidates take an Exam through a public link.

## Language

**Admin**:
A person who logs in to write Questions and schedule Exams. The operator creates each Admin. An Admin cannot sign up.
_Avoid_: user, teacher, examiner, administrator

**Candidate**:
A person who takes an Exam through the Exam link. A Candidate does not log in. A Candidate gives a name and an email address.
_Avoid_: user, student, participant, examinee

**Email address**:
The address that identifies a Candidate for one Exam. Two addresses that are different only in letter case are the same address.
_Avoid_: login, username

**Question**:
A multiple-choice question in the Question bank. It has 2 to 6 Options, and exactly one Option is correct. A Question cannot change after an Attempt uses it.
_Avoid_: item, problem

**Option**:
One possible Answer to a Question. It is plain text.
_Avoid_: choice, alternative

**Question bank**:
The set of all Questions. An Exam takes its Questions from the Question bank.
_Avoid_: pool, library

**Exam**:
A named set of Questions with an Exam window and a Time limit.
_Avoid_: quiz, test, assessment

**Exam window**:
The period between the open time and the close time of an Exam. A Candidate can start an Attempt only in this period.
_Avoid_: schedule, slot

**Time limit**:
The number of minutes that a Candidate has to finish an Attempt. It starts when the Candidate starts the Attempt.
_Avoid_: duration, timer

**Exam link**:
The public link of an Exam. One Exam has one Exam link at a time. When an Admin regenerates it, the old Exam link stops working.
_Avoid_: public link, URL, invite

**Attempt**:
One try of one Candidate at one Exam. A Candidate has at most one Attempt for each Exam. An Attempt is open until the Candidate submits it or the Time limit ends. Then it is submitted or timed out.
_Avoid_: submission, session, sitting

**Answer**:
The Option that a Candidate selects for one Question in an Attempt.
_Avoid_: response, selection

**Score**:
The number of Questions in an Attempt that the Candidate answered with the correct Option. Each correct Answer is 1 point. A wrong or skipped Question is 0 points.
_Avoid_: mark, grade, result
