//go:build e2e

package admin_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudradcruze/go-react-template/backend/api-test/e2e/testapi"
	"github.com/rudradcruze/go-react-template/backend/internal/admin/domain"
)

func countAdmins(t *testing.T, api *testapi.API) int {
	t.Helper()
	var n int
	require.NoError(t, api.DB.QueryRowContext(t.Context(), "SELECT count(*) FROM admins").Scan(&n))
	return n
}

func TestSeedAdmin_Execute_MakesAnAdminWhoCanLogIn(t *testing.T) {
	api := testapi.Start(t)

	created, err := api.SeedAdmin.Execute(t.Context(), " Owner@Example.COM ", "a password of 12+")

	require.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, 1, countAdmins(t, api))
	logIn(t, api, "owner@example.com", "a password of 12+")
}

func TestSeedAdmin_Execute_DoesNotChangeAnAdminThatExists(t *testing.T) {
	api := testapi.Start(t)
	_, err := api.SeedAdmin.Execute(t.Context(), "owner@example.com", "the first password")
	require.NoError(t, err)

	created, err := api.SeedAdmin.Execute(t.Context(), "OWNER@example.com", "the second password")

	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, 1, countAdmins(t, api))
	logIn(t, api, "owner@example.com", "the first password")
}

func TestSeedAdmin_Execute_MakesNoAdminForAPasswordThatBreaksTheRule(t *testing.T) {
	api := testapi.Start(t)

	created, err := api.SeedAdmin.Execute(t.Context(), "owner@example.com", "short")

	require.ErrorIs(t, err, domain.ErrInvalidPassword)
	assert.False(t, created)
	assert.Zero(t, countAdmins(t, api))
}

func TestAdminTable_RejectsAnEmailAddressInOtherLetterCase(t *testing.T) {
	api := testapi.Start(t)

	_, err := api.DB.ExecContext(t.Context(),
		"INSERT INTO admins (id, email_address, password_hash, created_at) VALUES (gen_random_uuid(), 'Owner@Example.com', 'x', now())")

	require.Error(t, err)
}
