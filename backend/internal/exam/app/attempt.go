package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
)

// AttemptView is an Attempt as a Candidate sees it. For an open Attempt it has the Exam, the Questions
// in the order of the Exam, and the saved Answers. For another Attempt, it has only the Attempt and its status.
type AttemptView struct {
	Attempt   domain.Attempt
	Status    domain.AttemptStatus
	Exam      domain.Exam
	Questions []domain.Question
	Answers   []domain.Answer
}

// attemptViewer builds the view of an Attempt.
type attemptViewer struct {
	exams     ExamRepository
	questions QuestionRepository
	attempts  AttemptRepository
}

func (v attemptViewer) view(ctx context.Context, attempt domain.Attempt, now time.Time) (AttemptView, error) {
	status := attempt.Status(now)
	if status != domain.StatusOpen {
		return AttemptView{Attempt: attempt, Status: status}, nil
	}
	exam, err := v.exams.FindByID(ctx, attempt.ExamID)
	if err != nil {
		return AttemptView{}, fmt.Errorf("The application did not read the Exam of the Attempt. Cause: %w", err)
	}
	questions, err := v.questions.FindByIDs(ctx, exam.QuestionIDs)
	if err != nil {
		return AttemptView{}, fmt.Errorf("The application did not read the Questions of the Exam. Cause: %w", err)
	}
	answers, err := v.attempts.Answers(ctx, attempt.ID)
	if err != nil {
		return AttemptView{}, fmt.Errorf("The application did not read the Answers of the Attempt. Cause: %w", err)
	}
	return AttemptView{Attempt: attempt, Status: status, Exam: exam, Questions: questions, Answers: answers}, nil
}

// GetExamLink is the use case that reads the Exam of an Exam link.
type GetExamLink struct {
	exams ExamRepository
}

// NewGetExamLink builds the use case.
func NewGetExamLink(exams ExamRepository) *GetExamLink {
	return &GetExamLink{exams: exams}
}

// Execute gives the Exam of the Exam link token. It returns domain.ErrExamLinkNotFound when the token is not known.
func (g *GetExamLink) Execute(ctx context.Context, token string) (domain.Exam, error) {
	return g.exams.FindByToken(ctx, token)
}

// StartAttemptInput is the data of a start, as the caller gives it.
type StartAttemptInput struct {
	Token        string
	Name         string
	EmailAddress string
}

// StartAttempt is the use case that starts an Attempt, or gives the open Attempt of the email address.
type StartAttempt struct {
	exams    ExamRepository
	attempts AttemptRepository
	viewer   attemptViewer
	now      func() time.Time
}

// NewStartAttempt builds the use case. now gives the time of the start.
func NewStartAttempt(exams ExamRepository, questions QuestionRepository, attempts AttemptRepository, now func() time.Time) *StartAttempt {
	return &StartAttempt{
		exams: exams, attempts: attempts, now: now,
		viewer: attemptViewer{exams: exams, questions: questions, attempts: attempts},
	}
}

// Execute starts an Attempt for the Candidate. created is true for a new Attempt.
//
// An email address with an open Attempt gets that Attempt, also after the Exam window. An email address with a
// submitted or timed-out Attempt gets domain.ErrAttemptFinished. A new Attempt needs the Exam window
// (domain.ErrExamNotOpen, domain.ErrExamClosed). It returns domain.ErrExamLinkNotFound for an unknown token,
// and an error of the domain when the name or the email address breaks a rule.
func (s *StartAttempt) Execute(ctx context.Context, input StartAttemptInput) (view AttemptView, created bool, err error) {
	exam, err := s.exams.FindByToken(ctx, input.Token)
	if err != nil {
		return AttemptView{}, false, err
	}
	candidate, err := domain.NewCandidate(input.Name, input.EmailAddress)
	if err != nil {
		return AttemptView{}, false, err
	}
	now := s.now()

	existing, err := s.attempts.FindByExamAndEmail(ctx, exam.ID, candidate.EmailAddress)
	if err == nil {
		view, err := s.viewExisting(ctx, existing, now)
		return view, false, err
	}
	if !errors.Is(err, domain.ErrAttemptNotFound) {
		return AttemptView{}, false, fmt.Errorf("The application did not read the Attempt. Cause: %w", err)
	}

	attempt, err := domain.NewAttempt(exam, candidate, now)
	if err != nil {
		return AttemptView{}, false, err
	}
	err = s.attempts.Create(ctx, attempt)
	if errors.Is(err, domain.ErrAttemptExists) {
		// Another request with the same email address saved its Attempt first. Give that Attempt.
		existing, findErr := s.attempts.FindByExamAndEmail(ctx, exam.ID, candidate.EmailAddress)
		if findErr != nil {
			return AttemptView{}, false, fmt.Errorf("The application did not read the Attempt. Cause: %w", findErr)
		}
		view, err := s.viewExisting(ctx, existing, now)
		return view, false, err
	}
	if err != nil {
		return AttemptView{}, false, fmt.Errorf("The application did not save the Attempt. Cause: %w", err)
	}
	view, err = s.viewer.view(ctx, attempt, now)
	return view, true, err
}

func (s *StartAttempt) viewExisting(ctx context.Context, attempt domain.Attempt, now time.Time) (AttemptView, error) {
	if attempt.Status(now) != domain.StatusOpen {
		return AttemptView{}, domain.ErrAttemptFinished
	}
	return s.viewer.view(ctx, attempt, now)
}

// GetAttempt is the use case that reads an Attempt by its ID.
type GetAttempt struct {
	attempts AttemptRepository
	viewer   attemptViewer
	now      func() time.Time
}

// NewGetAttempt builds the use case. now gives the time that the status uses.
func NewGetAttempt(exams ExamRepository, questions QuestionRepository, attempts AttemptRepository, now func() time.Time) *GetAttempt {
	return &GetAttempt{
		attempts: attempts, now: now,
		viewer: attemptViewer{exams: exams, questions: questions, attempts: attempts},
	}
}

// Execute gives the Attempt with this ID. It returns domain.ErrAttemptNotFound when there is none.
func (g *GetAttempt) Execute(ctx context.Context, id uuid.UUID) (AttemptView, error) {
	attempt, err := g.attempts.FindByID(ctx, id)
	if err != nil {
		return AttemptView{}, err
	}
	return g.viewer.view(ctx, attempt, g.now())
}
