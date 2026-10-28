// Package app holds the use cases of the Exam context, and the ports that the use cases call.
package app

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
)

// QuestionRepository stores the Questions of the Question bank.
type QuestionRepository interface {
	// Create saves a new Question with its Options.
	Create(ctx context.Context, question domain.Question) error
	// FindByID gives the Question with this ID. It returns domain.ErrQuestionNotFound when there is none.
	FindByID(ctx context.Context, id uuid.UUID) (domain.Question, error)
	// List gives all Questions, newest first.
	List(ctx context.Context) ([]domain.Question, error)
	// Update replaces the text and the Options of the Question. It returns domain.ErrQuestionNotFound when there is none.
	Update(ctx context.Context, question domain.Question) error
	// FindByIDs gives the Questions with these IDs, in the order of the IDs. It skips an ID that has no Question.
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Question, error)
	// UsedByAttempt tells if an Attempt of an Exam that has this Question exists.
	UsedByAttempt(ctx context.Context, id uuid.UUID) (bool, error)
	// Delete removes the Question and its Options. It returns domain.ErrQuestionNotFound when there is none,
	// and domain.ErrQuestionInUse when an Exam uses the Question.
	Delete(ctx context.Context, id uuid.UUID) error
}

// ExamRepository stores Exams with the IDs of their Questions.
type ExamRepository interface {
	// Create saves a new Exam. It returns domain.ErrUnknownQuestion when a Question of the Exam does not exist.
	Create(ctx context.Context, exam domain.Exam) error
	// FindByID gives the Exam with this ID. It returns domain.ErrExamNotFound when there is none.
	FindByID(ctx context.Context, id uuid.UUID) (domain.Exam, error)
	// List gives all Exams, newest first.
	List(ctx context.Context) ([]domain.Exam, error)
	// Update replaces all values of the Exam. It returns domain.ErrExamNotFound when there is none,
	// and domain.ErrUnknownQuestion when a Question of the Exam does not exist.
	Update(ctx context.Context, exam domain.Exam) error
	// FindByToken gives the Exam with this Exam link token. It returns domain.ErrExamLinkNotFound when there is none.
	FindByToken(ctx context.Context, token string) (domain.Exam, error)
	// Delete removes the Exam. It returns domain.ErrExamNotFound when there is none,
	// and domain.ErrExamLocked when the Exam has Attempts.
	Delete(ctx context.Context, id uuid.UUID) error
}

// AttemptRepository stores the Attempts and the Answers of the Candidates.
type AttemptRepository interface {
	// Create saves a new Attempt. It returns domain.ErrAttemptExists when the email address has an Attempt for the Exam.
	Create(ctx context.Context, attempt domain.Attempt) error
	// FindByID gives the Attempt with this ID. It returns domain.ErrAttemptNotFound when there is none.
	FindByID(ctx context.Context, id uuid.UUID) (domain.Attempt, error)
	// FindByExamAndEmail gives the Attempt of this email address for the Exam. It returns domain.ErrAttemptNotFound when there is none.
	FindByExamAndEmail(ctx context.Context, examID uuid.UUID, emailAddress domain.EmailAddress) (domain.Attempt, error)
	// ExistsForExam tells if the Exam has at least one Attempt.
	ExistsForExam(ctx context.Context, examID uuid.UUID) (bool, error)
	// SaveAnswer saves the Answer, or replaces the Answer for the same Question. It returns domain.ErrAttemptClosed
	// when the Attempt has a submit time or when the expiry is before now. The check and the save are one statement.
	SaveAnswer(ctx context.Context, attemptID uuid.UUID, answer domain.Answer, now time.Time) error
	// MarkSubmitted stores the submit time. It returns domain.ErrAttemptAlreadySubmitted when the Attempt has a submit time.
	MarkSubmitted(ctx context.Context, attemptID uuid.UUID, submittedAt time.Time) error
	// ListByExam gives all Attempts of the Exam, in the order of the start time.
	ListByExam(ctx context.Context, examID uuid.UUID) ([]domain.Attempt, error)
	// AnswersOfExam gives the saved Answers of all Attempts of the Exam, grouped by the ID of the Attempt.
	AnswersOfExam(ctx context.Context, examID uuid.UUID) (map[uuid.UUID][]domain.Answer, error)
	// Answers gives the saved Answers of the Attempt, in the order that the Candidate saved them.
	Answers(ctx context.Context, attemptID uuid.UUID) ([]domain.Answer, error)
}
