package app

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
)

// AttemptResult is one row of the results of an Exam.
type AttemptResult struct {
	Attempt domain.Attempt
	Status  domain.AttemptStatus
	// Score is nil while the Attempt is open. It counts the saved Answers.
	Score *domain.Score
	// FinishedAt is the submit time. For a timed-out Attempt without a submit, it is the expiry. It is nil while the Attempt is open.
	FinishedAt *time.Time
}

// ListExamAttempts is the use case that gives the results of an Exam to an Admin.
type ListExamAttempts struct {
	exams     ExamRepository
	questions QuestionRepository
	attempts  AttemptRepository
	now       func() time.Time
}

// NewListExamAttempts builds the use case. now gives the time that the status uses.
func NewListExamAttempts(exams ExamRepository, questions QuestionRepository, attempts AttemptRepository, now func() time.Time) *ListExamAttempts {
	return &ListExamAttempts{exams: exams, questions: questions, attempts: attempts, now: now}
}

// Execute gives one row for each Attempt of the Exam, in the order of the start time.
// It returns domain.ErrExamNotFound when the Exam does not exist.
func (l *ListExamAttempts) Execute(ctx context.Context, examID uuid.UUID) ([]AttemptResult, error) {
	exam, err := l.exams.FindByID(ctx, examID)
	if err != nil {
		return nil, err
	}
	questions, err := l.questions.FindByIDs(ctx, exam.QuestionIDs)
	if err != nil {
		return nil, fmt.Errorf("The application did not read the Questions of the Exam. Cause: %w", err)
	}
	attempts, err := l.attempts.ListByExam(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("The application did not read the Attempts of the Exam. Cause: %w", err)
	}
	answers, err := l.attempts.AnswersOfExam(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("The application did not read the Answers of the Exam. Cause: %w", err)
	}

	now := l.now()
	results := make([]AttemptResult, len(attempts))
	for i, attempt := range attempts {
		result := AttemptResult{Attempt: attempt, Status: attempt.Status(now)}
		if result.Status != domain.StatusOpen {
			score := domain.ScoreOf(questions, answers[attempt.ID])
			result.Score = &score
			finishedAt := attempt.ExpiresAt
			if attempt.SubmittedAt != nil {
				finishedAt = *attempt.SubmittedAt
			}
			result.FinishedAt = &finishedAt
		}
		results[i] = result
	}
	return results, nil
}
