package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/rudradcruze/go-react-template/backend/internal/admin/domain"
)

// CurrentAdmin is the use case that finds the Admin of a session.
type CurrentAdmin struct {
	sessions SessionStore
	admins   AdminRepository
}

// NewCurrentAdmin builds the use case.
func NewCurrentAdmin(sessions SessionStore, admins AdminRepository) *CurrentAdmin {
	return &CurrentAdmin{sessions: sessions, admins: admins}
}

// Execute gives the Admin of the session. It returns domain.ErrNotLoggedIn when the token
// is not known, the session ended, or the Admin does not exist.
func (c *CurrentAdmin) Execute(ctx context.Context, token string) (domain.Admin, error) {
	adminID, found, err := c.sessions.Get(ctx, token)
	if err != nil {
		return domain.Admin{}, fmt.Errorf("The application did not read the session. Cause: %w", err)
	}
	if !found {
		return domain.Admin{}, domain.ErrNotLoggedIn
	}
	admin, err := c.admins.FindByID(ctx, adminID)
	if errors.Is(err, domain.ErrAdminNotFound) {
		return domain.Admin{}, domain.ErrNotLoggedIn
	}
	if err != nil {
		return domain.Admin{}, fmt.Errorf("The application did not read the Admin. Cause: %w", err)
	}
	return admin, nil
}

// Logout is the use case that ends a session.
type Logout struct {
	sessions SessionStore
}

// NewLogout builds the use case.
func NewLogout(sessions SessionStore) *Logout {
	return &Logout{sessions: sessions}
}

// Execute ends the session.
func (l *Logout) Execute(ctx context.Context, token string) error {
	if err := l.sessions.Delete(ctx, token); err != nil {
		return fmt.Errorf("The application did not end the session. Cause: %w", err)
	}
	return nil
}
