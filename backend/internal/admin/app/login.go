package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rudradcruze/go-react-template/backend/internal/admin/domain"
)

const (
	// MaxFailedLogins is the number of failed logins of one email address that the window allows.
	MaxFailedLogins = 5
	// FailedLoginWindow is the time that a failed login counts.
	FailedLoginWindow = 15 * time.Minute
	// SessionLifetime is the time that a session lives.
	SessionLifetime = 8 * time.Hour
)

// LoginBlockedError tells that there were too many failed logins for the email address.
type LoginBlockedError struct {
	RetryAfter time.Duration
}

func (e *LoginBlockedError) Error() string {
	return fmt.Sprintf("There were too many failed logins. Try again in %s.", e.RetryAfter.Round(time.Second))
}

// LoginInput is the data of a login, as the caller gives it.
type LoginInput struct {
	EmailAddress string
	Password     string
}

// Login is the use case that starts a session.
type Login struct {
	admins   AdminRepository
	hasher   PasswordHasher
	sessions SessionStore
	failed   FailedLogins
}

// NewLogin builds the use case.
func NewLogin(admins AdminRepository, hasher PasswordHasher, sessions SessionStore, failed FailedLogins) *Login {
	return &Login{admins: admins, hasher: hasher, sessions: sessions, failed: failed}
}

// Execute starts a session and gives its token. It returns domain.ErrInvalidCredentials for an input
// that is not correct, with the same error for an unknown email address and for a wrong password.
// It returns *LoginBlockedError, without a check of the password, after too many failed logins.
func (l *Login) Execute(ctx context.Context, input LoginInput) (string, error) {
	emailAddress, err := domain.NewEmailAddress(input.EmailAddress)
	if err != nil || input.Password == "" {
		return "", domain.ErrInvalidCredentials
	}

	count, expiresIn, err := l.failed.Count(ctx, emailAddress)
	if err != nil {
		return "", fmt.Errorf("The application did not read the failed logins. Cause: %w", err)
	}
	if count >= MaxFailedLogins {
		return "", &LoginBlockedError{RetryAfter: expiresIn}
	}

	hash := ""
	admin, err := l.admins.FindByEmail(ctx, emailAddress)
	switch {
	case err == nil:
		hash = admin.PasswordHash
	case !errors.Is(err, domain.ErrAdminNotFound):
		return "", fmt.Errorf("The application did not read the Admin. Cause: %w", err)
	}
	if !l.hasher.Matches(hash, input.Password) {
		if err := l.failed.Add(ctx, emailAddress, FailedLoginWindow); err != nil {
			return "", fmt.Errorf("The application did not count the failed login. Cause: %w", err)
		}
		return "", domain.ErrInvalidCredentials
	}

	token, err := l.sessions.Create(ctx, admin.ID, SessionLifetime)
	if err != nil {
		return "", fmt.Errorf("The application did not start the session. Cause: %w", err)
	}
	return token, nil
}
