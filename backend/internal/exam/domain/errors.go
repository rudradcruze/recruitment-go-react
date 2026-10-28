package domain

import "errors"

// These are the errors of the rules of the domain. A caller tests for them with errors.Is.
var (
	ErrEmptyQuestionText   = errors.New("The text of the Question is empty.")
	ErrQuestionTextTooLong = errors.New("The text of the Question is too long.")
	ErrTextNotAllowed      = errors.New("The text has a character that is not allowed.")
	ErrTooFewOptions       = errors.New("The Question has too few Options.")
	ErrTooManyOptions      = errors.New("The Question has too many Options.")
	ErrEmptyOptionText     = errors.New("The text of an Option is empty.")
	ErrOptionTextTooLong   = errors.New("The text of an Option is too long.")
	ErrCorrectOptionCount  = errors.New("The Question must have exactly one correct Option.")
	ErrQuestionNotFound    = errors.New("The Question does not exist.")
	ErrQuestionInUse       = errors.New("An Exam uses the Question.")

	ErrEmptyTitle        = errors.New("The title of the Exam is empty.")
	ErrTitleTooLong      = errors.New("The title of the Exam is too long.")
	ErrTooFewQuestions   = errors.New("The Exam has too few Questions.")
	ErrTooManyQuestions  = errors.New("The Exam has too many Questions.")
	ErrDuplicateQuestion = errors.New("The Exam has the same Question 2 times.")
	ErrUnknownQuestion   = errors.New("A Question of the Exam does not exist.")
	ErrInvalidTimeLimit  = errors.New("The Time limit is not valid.")
	ErrInvalidWindow     = errors.New("The Exam window is not valid.")
	ErrExamNotFound      = errors.New("The Exam does not exist.")
	ErrExamLocked        = errors.New("Candidates started the Exam, so it cannot change.")
	ErrQuestionLocked    = errors.New("An Attempt used the Question, so it cannot change.")

	ErrInvalidEmailAddress  = errors.New("The email address is not valid.")
	ErrEmptyCandidateName   = errors.New("The name of the Candidate is empty.")
	ErrCandidateNameTooLong = errors.New("The name of the Candidate is too long.")
	ErrExamLinkNotFound     = errors.New("The Exam link is not valid.")
	ErrExamNotOpen          = errors.New("The Exam is not open yet.")
	ErrExamClosed           = errors.New("The Exam is closed.")
	ErrAttemptFinished      = errors.New("This Exam is already submitted for this email address.")
	ErrAttemptExists        = errors.New("An Attempt of this email address exists.")
	ErrAttemptNotFound      = errors.New("The Attempt does not exist.")

	ErrAttemptClosed           = errors.New("The Attempt is closed.")
	ErrAttemptAlreadySubmitted = errors.New("The Attempt is already submitted.")
	ErrInvalidAnswer           = errors.New("The Answer is not valid.")
)
