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

var _ app.ExamRepository = (*ExamRepository)(nil)

// examModel and examQuestionModel are the rows of the tables exams and exam_questions. The domain does not see them.
type examModel struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	Title            string
	OpensAt          time.Time
	ClosesAt         time.Time
	TimeLimitMinutes int
	LinkToken        string
	CreatedAt        time.Time
}

func (examModel) TableName() string { return "exams" }

type examQuestionModel struct {
	ExamID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	QuestionID uuid.UUID `gorm:"type:uuid"`
	Position   int       `gorm:"primaryKey"`
}

func (examQuestionModel) TableName() string { return "exam_questions" }

func examQuestionModels(exam domain.Exam) []examQuestionModel {
	models := make([]examQuestionModel, len(exam.QuestionIDs))
	for i, id := range exam.QuestionIDs {
		models[i] = examQuestionModel{ExamID: exam.ID, QuestionID: id, Position: i}
	}
	return models
}

func examToDomain(model examModel, questionIDs []uuid.UUID) domain.Exam {
	return domain.Exam{
		ID:               model.ID,
		Title:            model.Title,
		QuestionIDs:      questionIDs,
		OpensAt:          model.OpensAt.UTC(),
		ClosesAt:         model.ClosesAt.UTC(),
		TimeLimitMinutes: model.TimeLimitMinutes,
		LinkToken:        model.LinkToken,
		CreatedAt:        model.CreatedAt.UTC(),
	}
}

// ExamRepository stores Exams in Postgres with GORM.
type ExamRepository struct {
	db *gorm.DB
}

// NewExamRepository builds a repository on a database connection.
func NewExamRepository(db *gorm.DB) *ExamRepository {
	return &ExamRepository{db: db}
}

// Create saves a new Exam and its Question rows in one transaction.
func (r *ExamRepository) Create(ctx context.Context, exam domain.Exam) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		model := examModel{
			ID: exam.ID, Title: exam.Title, OpensAt: exam.OpensAt, ClosesAt: exam.ClosesAt,
			TimeLimitMinutes: exam.TimeLimitMinutes, LinkToken: exam.LinkToken, CreatedAt: exam.CreatedAt,
		}
		if err := tx.Create(&model).Error; err != nil {
			return err
		}
		return tx.Create(examQuestionModels(exam)).Error
	})
	return translateQuestionError(err)
}

// FindByID gives the Exam with this ID. It returns domain.ErrExamNotFound when there is none.
func (r *ExamRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.Exam, error) {
	var model examModel
	result := r.db.WithContext(ctx).Where("id = ?", id).Limit(1).Find(&model)
	if result.Error != nil {
		return domain.Exam{}, result.Error
	}
	if result.RowsAffected == 0 {
		return domain.Exam{}, domain.ErrExamNotFound
	}
	var rows []examQuestionModel
	if err := r.db.WithContext(ctx).Where("exam_id = ?", id).Order("position").Find(&rows).Error; err != nil {
		return domain.Exam{}, err
	}
	return examToDomain(model, questionIDsOf(rows)), nil
}

// FindByToken gives the Exam with this Exam link token. It returns domain.ErrExamLinkNotFound when there is none.
func (r *ExamRepository) FindByToken(ctx context.Context, token string) (domain.Exam, error) {
	var model examModel
	result := r.db.WithContext(ctx).Where("link_token = ?", token).Limit(1).Find(&model)
	if result.Error != nil {
		return domain.Exam{}, result.Error
	}
	if result.RowsAffected == 0 {
		return domain.Exam{}, domain.ErrExamLinkNotFound
	}
	var rows []examQuestionModel
	if err := r.db.WithContext(ctx).Where("exam_id = ?", model.ID).Order("position").Find(&rows).Error; err != nil {
		return domain.Exam{}, err
	}
	return examToDomain(model, questionIDsOf(rows)), nil
}

// List gives all Exams, newest first.
func (r *ExamRepository) List(ctx context.Context) ([]domain.Exam, error) {
	var models []examModel
	if err := r.db.WithContext(ctx).Order("created_at DESC, id DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	var rows []examQuestionModel
	if err := r.db.WithContext(ctx).Order("exam_id, position").Find(&rows).Error; err != nil {
		return nil, err
	}
	byExam := make(map[uuid.UUID][]examQuestionModel, len(models))
	for _, row := range rows {
		byExam[row.ExamID] = append(byExam[row.ExamID], row)
	}
	exams := make([]domain.Exam, len(models))
	for i, model := range models {
		exams[i] = examToDomain(model, questionIDsOf(byExam[model.ID]))
	}
	return exams, nil
}

// Update replaces all values of the Exam in one transaction.
func (r *ExamRepository) Update(ctx context.Context, exam domain.Exam) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&examModel{}).Where("id = ?", exam.ID).Updates(map[string]any{
			"title": exam.Title, "opens_at": exam.OpensAt, "closes_at": exam.ClosesAt,
			"time_limit_minutes": exam.TimeLimitMinutes, "link_token": exam.LinkToken,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.ErrExamNotFound
		}
		if err := tx.Where("exam_id = ?", exam.ID).Delete(&examQuestionModel{}).Error; err != nil {
			return err
		}
		return tx.Create(examQuestionModels(exam)).Error
	})
	return translateQuestionError(err)
}

// Delete removes the Exam. The database removes its Question rows too, and stops the delete
// with domain.ErrExamLocked when the Exam has Attempts.
func (r *ExamRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&examModel{})
	if errors.Is(result.Error, gorm.ErrForeignKeyViolated) {
		return domain.ErrExamLocked
	}
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrExamNotFound
	}
	return nil
}

func questionIDsOf(rows []examQuestionModel) []uuid.UUID {
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.QuestionID
	}
	return ids
}

// translateQuestionError changes a foreign key error on the Question rows into domain.ErrUnknownQuestion.
// The Exam ID is a new UUID, so a foreign key error here is always about a Question that does not exist.
func translateQuestionError(err error) error {
	if errors.Is(err, gorm.ErrForeignKeyViolated) {
		return domain.ErrUnknownQuestion
	}
	return err
}
