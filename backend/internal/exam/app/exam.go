package app

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
)

// CreateExam is the use case that makes an Exam.
type CreateExam struct {
	exams ExamRepository
	now   func() time.Time
}

// NewCreateExam builds the use case. now gives the time of the creation.
func NewCreateExam(exams ExamRepository, now func() time.Time) *CreateExam {
	return &CreateExam{exams: exams, now: now}
}

// Execute creates an Exam. It returns an error of the domain when the input breaks a rule,
// and domain.ErrUnknownQuestion when a Question does not exist.
func (c *CreateExam) Execute(ctx context.Context, input domain.ExamInput) (domain.Exam, error) {
	exam, err := domain.NewExam(input, c.now())
	if err != nil {
		return domain.Exam{}, err
	}
	if err := c.exams.Create(ctx, exam); err != nil {
		return domain.Exam{}, fmt.Errorf("The application did not save the Exam. Cause: %w", err)
	}
	return exam, nil
}

// GetExam is the use case that reads one Exam.
type GetExam struct {
	exams ExamRepository
}

// NewGetExam builds the use case.
func NewGetExam(exams ExamRepository) *GetExam {
	return &GetExam{exams: exams}
}

// Execute gives the Exam with this ID. It returns domain.ErrExamNotFound when there is none.
func (g *GetExam) Execute(ctx context.Context, id uuid.UUID) (domain.Exam, error) {
	return g.exams.FindByID(ctx, id)
}

// ListExams is the use case that reads all Exams.
type ListExams struct {
	exams ExamRepository
}

// NewListExams builds the use case.
func NewListExams(exams ExamRepository) *ListExams {
	return &ListExams{exams: exams}
}

// Execute gives all Exams, newest first.
func (l *ListExams) Execute(ctx context.Context) ([]domain.Exam, error) {
	return l.exams.List(ctx)
}

// UpdateExam is the use case that replaces the values of an Exam.
type UpdateExam struct {
	exams    ExamRepository
	attempts AttemptRepository
}

// NewUpdateExam builds the use case.
func NewUpdateExam(exams ExamRepository, attempts AttemptRepository) *UpdateExam {
	return &UpdateExam{exams: exams, attempts: attempts}
}

// Execute changes the Exam. It returns domain.ErrExamNotFound when there is none, an error of the domain
// when the input breaks a rule, and domain.ErrUnknownQuestion when a Question does not exist.
// After the first Attempt, only the close time can change. Another change returns domain.ErrExamLocked.
func (u *UpdateExam) Execute(ctx context.Context, id uuid.UUID, input domain.ExamInput) (domain.Exam, error) {
	exam, err := u.exams.FindByID(ctx, id)
	if err != nil {
		return domain.Exam{}, err
	}
	hasAttempts, err := u.attempts.ExistsForExam(ctx, id)
	if err != nil {
		return domain.Exam{}, fmt.Errorf("The application did not read the Attempts of the Exam. Cause: %w", err)
	}
	change := exam.Change
	if hasAttempts {
		change = exam.ChangeAfterAttempts
	}
	changed, err := change(input)
	if err != nil {
		return domain.Exam{}, err
	}
	if err := u.exams.Update(ctx, changed); err != nil {
		return domain.Exam{}, err
	}
	return changed, nil
}

// RegenerateExamLink is the use case that gives an Exam a new Exam link token.
type RegenerateExamLink struct {
	exams ExamRepository
}

// NewRegenerateExamLink builds the use case.
func NewRegenerateExamLink(exams ExamRepository) *RegenerateExamLink {
	return &RegenerateExamLink{exams: exams}
}

// Execute makes a new token. The old Exam link stops working. It returns domain.ErrExamNotFound when there is none.
func (r *RegenerateExamLink) Execute(ctx context.Context, id uuid.UUID) (domain.Exam, error) {
	exam, err := r.exams.FindByID(ctx, id)
	if err != nil {
		return domain.Exam{}, err
	}
	changed, err := exam.RegenerateLink()
	if err != nil {
		return domain.Exam{}, err
	}
	if err := r.exams.Update(ctx, changed); err != nil {
		return domain.Exam{}, err
	}
	return changed, nil
}

// DeleteExam is the use case that removes an Exam.
type DeleteExam struct {
	exams ExamRepository
}

// NewDeleteExam builds the use case.
func NewDeleteExam(exams ExamRepository) *DeleteExam {
	return &DeleteExam{exams: exams}
}

// Execute deletes the Exam. It returns domain.ErrExamNotFound when there is none,
// and domain.ErrExamLocked when the Exam has Attempts.
func (d *DeleteExam) Execute(ctx context.Context, id uuid.UUID) error {
	return d.exams.Delete(ctx, id)
}
