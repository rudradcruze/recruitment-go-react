// Package app holds the use cases of the Admin context, and the ports that the use cases call.
package app

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/rudradcruze/go-react-template/backend/internal/admin/domain"
)

// AdminRepository stores Admins.
type AdminRepository interface {
	// Create saves a new Admin. It returns domain.ErrEmailTaken when another Admin has the same email address.
	Create(ctx context.Context, admin domain.Admin) error
	// FindByEmail gives the Admin with this email address. It returns domain.ErrAdminNotFound when there is none.
	FindByEmail(ctx context.Context, emailAddress domain.EmailAddress) (domain.Admin, error)
	// FindByID gives the Admin with this ID. It returns domain.ErrAdminNotFound when there is none.
	FindByID(ctx context.Context, id uuid.UUID) (domain.Admin, error)
}

// PasswordHasher makes and checks the hashes of passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
	// Matches tells if the password has this hash. An empty hash never matches, and the call takes
	// the same time as for a real hash, so the time does not show that an email address is unknown.
	Matches(hash, password string) bool
}

// SessionStore keeps the sessions of the Admins.
type SessionStore interface {
	// Create starts a session for the Admin and gives its token. The session ends after the lifetime.
	Create(ctx context.Context, adminID uuid.UUID, lifetime time.Duration) (token string, err error)
	// Get gives the Admin of the session. found is false when the token is not known or the session ended.
	Get(ctx context.Context, token string) (adminID uuid.UUID, found bool, err error)
	Delete(ctx context.Context, token string) error
}

// FailedLogins counts the failed logins of each email address.
type FailedLogins interface {
	// Count gives the failed logins in the window and the time until the count expires.
	Count(ctx context.Context, emailAddress domain.EmailAddress) (count int, expiresIn time.Duration, err error)
	// Add counts one failed login. The count expires after the window that starts at the first failure.
	Add(ctx context.Context, emailAddress domain.EmailAddress, window time.Duration) error
}
