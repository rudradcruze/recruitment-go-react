// Package domain holds the rules of the Admin context. It imports no framework, database or cache.
package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// The password has 12 to 72 bytes. bcrypt cannot use more than 72 bytes.
const (
	MinPasswordBytes = 12
	MaxPasswordBytes = 72
)

// Admin is a person who logs in to write Questions and schedule Exams.
type Admin struct {
	ID           uuid.UUID
	EmailAddress EmailAddress
	PasswordHash string
	CreatedAt    time.Time
}

// NewAdmin makes an Admin at the given time. It makes the ID, a UUID of version 7.
func NewAdmin(emailAddress, passwordHash string, now time.Time) (Admin, error) {
	email, err := NewEmailAddress(emailAddress)
	if err != nil {
		return Admin{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return Admin{}, fmt.Errorf("The ID of the Admin is not available. Cause: %w", err)
	}
	return Admin{
		ID:           id,
		EmailAddress: email,
		PasswordHash: passwordHash,
		// Postgres keeps microseconds. Truncate here, so the Admin is the same after it is saved.
		CreatedAt: now.UTC().Truncate(time.Microsecond),
	}, nil
}

// ValidatePassword checks the length of a password for a new Admin.
func ValidatePassword(password string) error {
	if len(password) < MinPasswordBytes || len(password) > MaxPasswordBytes {
		return ErrInvalidPassword
	}
	return nil
}
