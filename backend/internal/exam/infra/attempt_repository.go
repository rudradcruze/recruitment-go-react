package infra

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rudradcruze/go-react-template/backend/internal/exam/app"
	"github.com/rudradcruze/go-react-template/backend/internal/exam/domain"
)

var _ app.AttemptRepository = (*AttemptRepository)(nil)

// attemptModel and answerModel are the rows of the tables attempts and answers. The domain does not see them.
type attemptModel struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	ExamID        uuid.UUID `gorm:"type:uuid"`
	CandidateName string
	EmailAddress  string
	StartedAt     time.Time
	ExpiresAt     time.Time
	SubmittedAt   *time.Time
}

func (attemptModel) TableName() string { return "attempts" }

func (m attemptModel) toDomain() domain.Attempt {
	attempt := domain.Attempt{
		ID:            m.ID,
		ExamID:        m.ExamID,
		CandidateName: m.CandidateName,
		EmailAddress:  domain.EmailAddress(m.EmailAddress),
		StartedAt:     m.StartedAt.UTC(),
		ExpiresAt:     m.ExpiresAt.UTC(),
	}
	if m.SubmittedAt != nil {
		submittedAt := m.SubmittedAt.UTC()
		attempt.SubmittedAt = &submittedAt
	}
	return attempt
}

type answerModel struct {
	AttemptID  uuid.UUID `gorm:"type:uuid;primaryKey"`
	QuestionID uuid.UUID `gorm:"type:uuid;primaryKey"`
	OptionID   uuid.UUID `gorm:"type:uuid"`
	SavedAt    time.Time
}

func (answerModel) TableName() string { return "answers" }

// AttemptRepository stores Attempts and Answers in Postgres with GORM.
type AttemptRepository struct {
	db *gorm.DB
}

// NewAttemptRepository builds a repository on a database connection.
func NewAttemptRepository(db *gorm.DB) *AttemptRepository {
	return &AttemptRepository{db: db}
}

// Create saves a new Attempt. It returns domain.ErrAttemptExists when the email address has an Attempt for the Exam.
func (r *AttemptRepository) Create(ctx context.Context, attempt domain.Attempt) error {
	model := attemptModel{
		ID: attempt.ID, ExamID: attempt.ExamID, CandidateName: attempt.CandidateName,
		EmailAddress: string(attempt.EmailAddress), StartedAt: attempt.StartedAt, ExpiresAt: attempt.ExpiresAt,
		SubmittedAt: attempt.SubmittedAt,
	}
	err := r.db.WithContext(ctx).Create(&model).Error
	// The unique index on the Exam and the email address is the only one that a caller can clash with.
	// The ID is a new random UUID, so a clash of the ID does not occur.
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return domain.ErrAttemptExists
	}
	return err
}

// FindByID gives the Attempt with this ID. It returns domain.ErrAttemptNotFound when there is none.
func (r *AttemptRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.Attempt, error) {
	return r.find(ctx, "id = ?", id)
}

// FindByExamAndEmail gives the Attempt of this email address for the Exam. It returns domain.ErrAttemptNotFound when there is none.
func (r *AttemptRepository) FindByExamAndEmail(ctx context.Context, examID uuid.UUID, emailAddress domain.EmailAddress) (domain.Attempt, error) {
	return r.find(ctx, "exam_id = ? AND email_address = ?", examID, string(emailAddress))
}

func (r *AttemptRepository) find(ctx context.Context, query string, args ...any) (domain.Attempt, error) {
	var model attemptModel
	result := r.db.WithContext(ctx).Where(query, args...).Limit(1).Find(&model)
	if result.Error != nil {
		return domain.Attempt{}, result.Error
	}
	if result.RowsAffected == 0 {
		return domain.Attempt{}, domain.ErrAttemptNotFound
	}
	return model.toDomain(), nil
}

// ExistsForExam tells if the Exam has at least one Attempt.
func (r *AttemptRepository) ExistsForExam(ctx context.Context, examID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.WithContext(ctx).
		Raw("SELECT EXISTS (SELECT 1 FROM attempts WHERE exam_id = ?)", examID).
		Scan(&exists).Error
	return exists, err
}

// SaveAnswer saves the Answer, or replaces the Answer for the same Question. The statement saves only when the Attempt
// has no submit time and the expiry is not before now, so a submit or the expiry between the check of the use case
// and this statement cannot let an Answer in. It returns domain.ErrAttemptClosed when the statement saves nothing.
func (r *AttemptRepository) SaveAnswer(ctx context.Context, attemptID uuid.UUID, answer domain.Answer, now time.Time) error {
	result := r.db.WithContext(ctx).Exec(`
		INSERT INTO answers (attempt_id, question_id, option_id, saved_at)
		SELECT id, ?, ?, ? FROM attempts WHERE id = ? AND submitted_at IS NULL AND expires_at >= ?
		ON CONFLICT (attempt_id, question_id) DO UPDATE SET option_id = EXCLUDED.option_id, saved_at = EXCLUDED.saved_at`,
		answer.QuestionID, answer.OptionID, now.UTC(), attemptID, now.UTC())
	if errors.Is(result.Error, gorm.ErrForeignKeyViolated) {
		return domain.ErrInvalidAnswer
	}
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrAttemptClosed
	}
	return nil
}

// MarkSubmitted stores the submit time. It returns domain.ErrAttemptAlreadySubmitted when the Attempt has a submit time.
func (r *AttemptRepository) MarkSubmitted(ctx context.Context, attemptID uuid.UUID, submittedAt time.Time) error {
	result := r.db.WithContext(ctx).Model(&attemptModel{}).
		Where("id = ? AND submitted_at IS NULL", attemptID).
		Update("submitted_at", submittedAt.UTC())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrAttemptAlreadySubmitted
	}
	return nil
}

// ListByExam gives all Attempts of the Exam, in the order of the start time.
func (r *AttemptRepository) ListByExam(ctx context.Context, examID uuid.UUID) ([]domain.Attempt, error) {
	var models []attemptModel
	if err := r.db.WithContext(ctx).Where("exam_id = ?", examID).Order("started_at, id").Find(&models).Error; err != nil {
		return nil, err
	}
	attempts := make([]domain.Attempt, len(models))
	for i, model := range models {
		attempts[i] = model.toDomain()
	}
	return attempts, nil
}

// AnswersOfExam gives the saved Answers of all Attempts of the Exam, grouped by the ID of the Attempt.
func (r *AttemptRepository) AnswersOfExam(ctx context.Context, examID uuid.UUID) (map[uuid.UUID][]domain.Answer, error) {
	var models []answerModel
	err := r.db.WithContext(ctx).
		Where("attempt_id IN (?)", r.db.Model(&attemptModel{}).Select("id").Where("exam_id = ?", examID)).
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	byAttempt := make(map[uuid.UUID][]domain.Answer)
	for _, model := range models {
		byAttempt[model.AttemptID] = append(byAttempt[model.AttemptID], domain.Answer{QuestionID: model.QuestionID, OptionID: model.OptionID})
	}
	return byAttempt, nil
}

// Answers gives the saved Answers of the Attempt, in the order that the Candidate saved them.
func (r *AttemptRepository) Answers(ctx context.Context, attemptID uuid.UUID) ([]domain.Answer, error) {
	var models []answerModel
	if err := r.db.WithContext(ctx).Where("attempt_id = ?", attemptID).Order("saved_at, question_id").Find(&models).Error; err != nil {
		return nil, err
	}
	answers := make([]domain.Answer, len(models))
	for i, model := range models {
		answers[i] = domain.Answer{QuestionID: model.QuestionID, OptionID: model.OptionID}
	}
	return answers, nil
}
