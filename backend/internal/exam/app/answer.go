package app

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
)

// SaveAnswer is the use case that saves the Option that a Candidate selected for a Question.
type SaveAnswer struct {
	exams     ExamRepository
	questions QuestionRepository
	attempts  AttemptRepository
	now       func() time.Time
}

// NewSaveAnswer builds the use case. now gives the time that the Time limit check uses.
func NewSaveAnswer(exams ExamRepository, questions QuestionRepository, attempts AttemptRepository, now func() time.Time) *SaveAnswer {
	return &SaveAnswer{exams: exams, questions: questions, attempts: attempts, now: now}
}

// Execute saves the Answer. A second call for the same Question replaces the Answer. It returns
// domain.ErrAttemptNotFound for an unknown Attempt, domain.ErrAttemptClosed after the expiry or the submit,
// and domain.ErrInvalidAnswer for a Question that is not in the Exam or an Option that is not an Option of the Question.
func (s *SaveAnswer) Execute(ctx context.Context, attemptID uuid.UUID, answer domain.Answer) error {
	attempt, err := s.attempts.FindByID(ctx, attemptID)
	if err != nil {
		return err
	}
	now := s.now()
	if err := attempt.AcceptsAnswers(now); err != nil {
		return err
	}
	exam, err := s.exams.FindByID(ctx, attempt.ExamID)
	if err != nil {
		return fmt.Errorf("The application did not read the Exam of the Attempt. Cause: %w", err)
	}
	questions, err := s.questions.FindByIDs(ctx, exam.QuestionIDs)
	if err != nil {
		return fmt.Errorf("The application did not read the Questions of the Exam. Cause: %w", err)
	}
	if err := domain.ValidateAnswer(exam, questions, answer); err != nil {
		return err
	}
	// The repository checks the expiry and the submit again in the same statement as the save.
	return s.attempts.SaveAnswer(ctx, attemptID, answer, now)
}

// SubmitAttempt is the use case that ends an Attempt and gives its Score.
type SubmitAttempt struct {
	exams     ExamRepository
	questions QuestionRepository
	attempts  AttemptRepository
	now       func() time.Time
}

// NewSubmitAttempt builds the use case. now gives the submit time.
func NewSubmitAttempt(exams ExamRepository, questions QuestionRepository, attempts AttemptRepository, now func() time.Time) *SubmitAttempt {
	return &SubmitAttempt{exams: exams, questions: questions, attempts: attempts, now: now}
}

// Execute stores the submit time and gives the Score. It works after the expiry too: the Answers cannot change
// after the expiry, so the Score uses only Answers that the Candidate saved in time. It returns
// domain.ErrAttemptNotFound for an unknown Attempt, and domain.ErrAttemptAlreadySubmitted for a second submit.
func (s *SubmitAttempt) Execute(ctx context.Context, attemptID uuid.UUID) (domain.Score, error) {
	attempt, err := s.attempts.FindByID(ctx, attemptID)
	if err != nil {
		return domain.Score{}, err
	}
	submitted, err := attempt.Submit(s.now())
	if err != nil {
		return domain.Score{}, err
	}
	if err := s.attempts.MarkSubmitted(ctx, attemptID, *submitted.SubmittedAt); err != nil {
		return domain.Score{}, err
	}
	exam, err := s.exams.FindByID(ctx, attempt.ExamID)
	if err != nil {
		return domain.Score{}, fmt.Errorf("The application did not read the Exam of the Attempt. Cause: %w", err)
	}
	questions, err := s.questions.FindByIDs(ctx, exam.QuestionIDs)
	if err != nil {
		return domain.Score{}, fmt.Errorf("The application did not read the Questions of the Exam. Cause: %w", err)
	}
	answers, err := s.attempts.Answers(ctx, attemptID)
	if err != nil {
		return domain.Score{}, fmt.Errorf("The application did not read the Answers of the Attempt. Cause: %w", err)
	}
	return domain.ScoreOf(questions, answers), nil
}
