package admin_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudradcruze/go-react-template/backend/internal/admin/domain"
)

func TestEmailAddress_New_RemovesTheSpacesAndChangesTheLettersToLowercase(t *testing.T) {
	t.Parallel()

	got, err := domain.NewEmailAddress("  Admin@Example.COM \t")

	require.NoError(t, err)
	assert.Equal(t, domain.EmailAddress("admin@example.com"), got)
}

func TestEmailAddress_New_RejectsAnAddressThatIsNotValid(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"empty":                  "",
		"only spaces":            "   ",
		"no at sign":             "admin.example.com",
		"no local part":          "@example.com",
		"no domain":              "admin@",
		"domain without a dot":   "admin@example",
		"space in the address":   "ad min@example.com",
		"display name included":  "Admin <admin@example.com>",
		"more than 254 chars":    strings.Repeat("a", 250) + "@example.com",
		"non-ASCII local part":   "admäin@example.com",
		"Kelvin sign":            "admin@exampleK.com",
		"label starts with dash": "admin@-example.com",
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.NewEmailAddress(input)

			require.ErrorIs(t, err, domain.ErrInvalidEmailAddress)
		})
	}
}

func TestPassword_Validate_AcceptsFrom12To72BytesAndRejectsOtherLengths(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		password string
		valid    bool
	}{
		"11 bytes":                           {strings.Repeat("a", 11), false},
		"12 bytes":                           {strings.Repeat("a", 12), true},
		"72 bytes":                           {strings.Repeat("a", 72), true},
		"73 bytes":                           {strings.Repeat("a", 73), false},
		"6 characters of 2 bytes, 12 bytes":  {strings.Repeat("é", 6), true},
		"37 characters of 2 bytes, 74 bytes": {strings.Repeat("é", 37), false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := domain.ValidatePassword(tt.password)

			if tt.valid {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, domain.ErrInvalidPassword)
			}
		})
	}
}

func TestAdmin_New_MakesAnAdminWithAVersion7IDAndANormalizedEmailAddress(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 27, 10, 0, 0, 123456789, time.UTC)

	got, err := domain.NewAdmin(" Admin@Example.COM ", "hash", now)

	require.NoError(t, err)
	assert.Equal(t, uuid.Version(7), got.ID.Version())
	assert.Equal(t, domain.EmailAddress("admin@example.com"), got.EmailAddress)
	assert.Equal(t, "hash", got.PasswordHash)
	assert.Equal(t, time.Date(2026, 10, 27, 10, 0, 0, 123456000, time.UTC), got.CreatedAt)
}

func TestAdmin_New_RejectsAnEmailAddressThatIsNotValid(t *testing.T) {
	t.Parallel()

	_, err := domain.NewAdmin("admin", "hash", time.Now())

	require.ErrorIs(t, err, domain.ErrInvalidEmailAddress)
}
