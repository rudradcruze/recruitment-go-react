// Package infra connects the Admin context to Postgres and Redis.
package infra

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rudradcruze/go-react-template/backend/internal/admin/app"
	"github.com/rudradcruze/go-react-template/backend/internal/admin/domain"
)

var _ app.AdminRepository = (*AdminRepository)(nil)

// adminModel is the row of the table admins. The domain does not see it.
type adminModel struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	EmailAddress string
	PasswordHash string
	CreatedAt    time.Time
}

func (adminModel) TableName() string { return "admins" }

// toDomain builds the Admin of a row. The database has only values that the domain accepted, so it does not check them again.
func (m adminModel) toDomain() domain.Admin {
	return domain.Admin{
		ID:           m.ID,
		EmailAddress: domain.EmailAddress(m.EmailAddress),
		PasswordHash: m.PasswordHash,
		CreatedAt:    m.CreatedAt.UTC(),
	}
}

// AdminRepository stores Admins in Postgres with GORM.
type AdminRepository struct {
	db *gorm.DB
}

// NewAdminRepository builds a repository on a database connection.
func NewAdminRepository(db *gorm.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

// Create saves a new Admin. It returns domain.ErrEmailTaken when another Admin has the same email address.
func (r *AdminRepository) Create(ctx context.Context, admin domain.Admin) error {
	model := adminModel{
		ID:           admin.ID,
		EmailAddress: string(admin.EmailAddress),
		PasswordHash: admin.PasswordHash,
		CreatedAt:    admin.CreatedAt,
	}
	err := r.db.WithContext(ctx).Create(&model).Error
	// The email address is the only unique value that a caller chooses. The ID is a new UUID, so a clash of the ID does not occur.
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return domain.ErrEmailTaken
	}
	return err
}

// FindByEmail gives the Admin with this email address. It returns domain.ErrAdminNotFound when there is none.
func (r *AdminRepository) FindByEmail(ctx context.Context, emailAddress domain.EmailAddress) (domain.Admin, error) {
	return r.find(ctx, "email_address = ?", string(emailAddress))
}

// FindByID gives the Admin with this ID. It returns domain.ErrAdminNotFound when there is none.
func (r *AdminRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.Admin, error) {
	return r.find(ctx, "id = ?", id)
}

func (r *AdminRepository) find(ctx context.Context, query string, arg any) (domain.Admin, error) {
	var model adminModel
	err := r.db.WithContext(ctx).Where(query, arg).Take(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Admin{}, domain.ErrAdminNotFound
	}
	if err != nil {
		return domain.Admin{}, err
	}
	return model.toDomain(), nil
}
