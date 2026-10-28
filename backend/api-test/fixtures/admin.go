// Package fixtures builds test data. A builder gives a valid object with default values,
// and each call gives unique values for the unique fields.
package fixtures

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rudradcruze/go-react-template/backend/internal/admin/app"
	"github.com/rudradcruze/go-react-template/backend/internal/admin/domain"
)

var adminCounter atomic.Uint64

type adminValues struct {
	emailAddress string
	password     string
	createdAt    time.Time
}

func newAdminValues() adminValues {
	n := adminCounter.Add(1)
	return adminValues{
		emailAddress: fmt.Sprintf("admin%d@example.com", n),
		password:     fmt.Sprintf("password-of-admin-%d", n),
		createdAt:    time.Now(),
	}
}

// AdminOption changes a value of the Admin that Admin builds.
type AdminOption func(*adminValues)

// WithAdminEmailAddress sets the email address.
func WithAdminEmailAddress(emailAddress string) AdminOption {
	return func(v *adminValues) { v.emailAddress = emailAddress }
}

// WithAdminPassword sets the password. SaveAdmin hashes it.
func WithAdminPassword(password string) AdminOption {
	return func(v *adminValues) { v.password = password }
}

func applyAdminOptions(options []AdminOption) adminValues {
	values := newAdminValues()
	for _, option := range options {
		option(&values)
	}
	return values
}

// Admin builds a valid Admin. Its password hash is a fixed text, not a real hash: use SaveAdmin to log in.
// It panics when an option gives a value that the domain does not accept, because that is a mistake in the test.
func Admin(options ...AdminOption) domain.Admin {
	values := applyAdminOptions(options)
	admin, err := domain.NewAdmin(values.emailAddress, "hash-of-"+values.password, values.createdAt)
	if err != nil {
		panic(fmt.Sprintf("The fixture Admin is not valid. Cause: %v", err))
	}
	return admin
}

// SaveAdmin builds an Admin with a real hash of its password, and saves it through the repository.
// It gives the Admin and the password.
func SaveAdmin(t *testing.T, admins app.AdminRepository, hasher app.PasswordHasher, options ...AdminOption) (domain.Admin, string) {
	t.Helper()
	values := applyAdminOptions(options)
	hash, err := hasher.Hash(values.password)
	if err != nil {
		t.Fatalf("The fixture password did not hash. Cause: %v", err)
	}
	admin, err := domain.NewAdmin(values.emailAddress, hash, values.createdAt)
	if err != nil {
		t.Fatalf("The fixture Admin is not valid. Cause: %v", err)
	}
	if err := admins.Create(t.Context(), admin); err != nil {
		t.Fatalf("The fixture Admin did not save. Cause: %v", err)
	}
	return admin, values.password
}
