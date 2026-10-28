// Package infra connects the Exam context to Postgres.
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

var _ app.QuestionRepository = (*QuestionRepository)(nil)

// questionModel and optionModel are the rows of the tables questions and options. The domain does not see them.
type questionModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Text      string
	CreatedAt time.Time
}

func (questionModel) TableName() string { return "questions" }

type optionModel struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	QuestionID uuid.UUID `gorm:"type:uuid"`
	Position   int
	Text       string
	Correct    bool
}

func (optionModel) TableName() string { return "options" }

func optionModels(question domain.Question) []optionModel {
	models := make([]optionModel, len(question.Options))
	for i, option := range question.Options {
		models[i] = optionModel{ID: option.ID, QuestionID: question.ID, Position: i, Text: option.Text, Correct: option.Correct}
	}
	return models
}

// toDomain builds the Question of its rows. The database has only values that the domain accepted, so it does not check them again.
func toDomain(model questionModel, options []optionModel) domain.Question {
	question := domain.Question{ID: model.ID, Text: model.Text, CreatedAt: model.CreatedAt.UTC(), Options: make([]domain.Option, len(options))}
	for i, option := range options {
		question.Options[i] = domain.Option{ID: option.ID, Text: option.Text, Correct: option.Correct}
	}
	return question
}

// QuestionRepository stores Questions in Postgres with GORM.
type QuestionRepository struct {
	db *gorm.DB
}

// NewQuestionRepository builds a repository on a database connection.
func NewQuestionRepository(db *gorm.DB) *QuestionRepository {
	return &QuestionRepository{db: db}
}

// Create saves a new Question and its Options in one transaction.
func (r *QuestionRepository) Create(ctx context.Context, question domain.Question) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		model := questionModel{ID: question.ID, Text: question.Text, CreatedAt: question.CreatedAt}
		if err := tx.Create(&model).Error; err != nil {
			return err
		}
		return tx.Create(optionModels(question)).Error
	})
}

// FindByID gives the Question with this ID. It returns domain.ErrQuestionNotFound when there is none.
func (r *QuestionRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.Question, error) {
	var model questionModel
	result := r.db.WithContext(ctx).Where("id = ?", id).Limit(1).Find(&model)
	if result.Error != nil {
		return domain.Question{}, result.Error
	}
	if result.RowsAffected == 0 {
		return domain.Question{}, domain.ErrQuestionNotFound
	}
	var options []optionModel
	if err := r.db.WithContext(ctx).Where("question_id = ?", id).Order("position").Find(&options).Error; err != nil {
		return domain.Question{}, err
	}
	return toDomain(model, options), nil
}

// FindByIDs gives the Questions with these IDs, in the order of the IDs. It skips an ID that has no Question.
func (r *QuestionRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Question, error) {
	var models []questionModel
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&models).Error; err != nil {
		return nil, err
	}
	var options []optionModel
	if err := r.db.WithContext(ctx).Where("question_id IN ?", ids).Order("question_id, position").Find(&options).Error; err != nil {
		return nil, err
	}
	modelByID := make(map[uuid.UUID]questionModel, len(models))
	for _, model := range models {
		modelByID[model.ID] = model
	}
	optionsByQuestion := make(map[uuid.UUID][]optionModel, len(models))
	for _, option := range options {
		optionsByQuestion[option.QuestionID] = append(optionsByQuestion[option.QuestionID], option)
	}
	questions := make([]domain.Question, 0, len(ids))
	for _, id := range ids {
		if model, ok := modelByID[id]; ok {
			questions = append(questions, toDomain(model, optionsByQuestion[id]))
		}
	}
	return questions, nil
}

// UsedByAttempt tells if an Attempt of an Exam that has this Question exists.
func (r *QuestionRepository) UsedByAttempt(ctx context.Context, id uuid.UUID) (bool, error) {
	var used bool
	err := r.db.WithContext(ctx).Raw(`SELECT EXISTS (
		SELECT 1 FROM attempts a JOIN exam_questions eq ON eq.exam_id = a.exam_id WHERE eq.question_id = ?)`, id).
		Scan(&used).Error
	return used, err
}

// List gives all Questions, newest first.
func (r *QuestionRepository) List(ctx context.Context) ([]domain.Question, error) {
	var models []questionModel
	if err := r.db.WithContext(ctx).Order("created_at DESC, id DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	var options []optionModel
	if err := r.db.WithContext(ctx).Order("question_id, position").Find(&options).Error; err != nil {
		return nil, err
	}
	byQuestion := make(map[uuid.UUID][]optionModel, len(models))
	for _, option := range options {
		byQuestion[option.QuestionID] = append(byQuestion[option.QuestionID], option)
	}
	questions := make([]domain.Question, len(models))
	for i, model := range models {
		questions[i] = toDomain(model, byQuestion[model.ID])
	}
	return questions, nil
}

// Update replaces the text and the Options of the Question in one transaction.
func (r *QuestionRepository) Update(ctx context.Context, question domain.Question) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&questionModel{}).Where("id = ?", question.ID).Update("text", question.Text)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.ErrQuestionNotFound
		}
		if err := tx.Where("question_id = ?", question.ID).Delete(&optionModel{}).Error; err != nil {
			return err
		}
		return tx.Create(optionModels(question)).Error
	})
}

// Delete removes the Question. The database removes its Options too, and stops the delete
// with domain.ErrQuestionInUse when an Exam uses the Question.
func (r *QuestionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&questionModel{})
	if errors.Is(result.Error, gorm.ErrForeignKeyViolated) {
		return domain.ErrQuestionInUse
	}
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrQuestionNotFound
	}
	return nil
}
