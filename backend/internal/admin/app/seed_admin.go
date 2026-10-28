package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rudradcruze/go-react-template/backend/internal/admin/domain"
)

// SeedAdmin is the use case that makes the first Admin. An operator runs it, not a person on the web.
type SeedAdmin struct {
	admins AdminRepository
	hasher PasswordHasher
	now    func() time.Time
}

// NewSeedAdmin builds the use case. now gives the time of the creation.
func NewSeedAdmin(admins AdminRepository, hasher PasswordHasher, now func() time.Time) *SeedAdmin {
	return &SeedAdmin{admins: admins, hasher: hasher, now: now}
}

// Execute makes an Admin. It does nothing, and returns false, when an Admin with this email address exists.
// It never changes the password of an Admin that exists. It returns an error of the domain when the
// email address or the password breaks a rule.
func (s *SeedAdmin) Execute(ctx context.Context, emailAddress, password string) (created bool, err error) {
	email, err := domain.NewEmailAddress(emailAddress)
	if err != nil {
		return false, err
	}
	if err := domain.ValidatePassword(password); err != nil {
		return false, err
	}

	_, err = s.admins.FindByEmail(ctx, email)
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, domain.ErrAdminNotFound) {
		return false, fmt.Errorf("The application did not read the Admin. Cause: %w", err)
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return false, fmt.Errorf("The application did not hash the password. Cause: %w", err)
	}
	admin, err := domain.NewAdmin(string(email), hash, s.now())
	if err != nil {
		return false, err
	}
	err = s.admins.Create(ctx, admin)
	if errors.Is(err, domain.ErrEmailTaken) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("The application did not save the Admin. Cause: %w", err)
	}
	return true, nil
}
