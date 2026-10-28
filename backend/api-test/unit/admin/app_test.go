package admin_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/rudradcruze/go-react-template/backend/api-test/mocks"
	"github.com/rudradcruze/go-react-template/backend/internal/admin/app"
	"github.com/rudradcruze/go-react-template/backend/internal/admin/domain"
)

var ann = domain.Admin{
	ID:           uuid.MustParse("0198f2a0-0000-7000-8000-000000000001"),
	EmailAddress: "ann@example.com",
	PasswordHash: "stored-hash",
}

func TestLimits_AreFiveFailuresIn15MinutesAndASessionOf8Hours(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 5, app.MaxFailedLogins)
	assert.Equal(t, 15*time.Minute, app.FailedLoginWindow)
	assert.Equal(t, 8*time.Hour, app.SessionLifetime)
}

type loginMocks struct {
	admins   *mocks.MockAdminRepository
	hasher   *mocks.MockPasswordHasher
	sessions *mocks.MockSessionStore
	failed   *mocks.MockFailedLogins
}

func newLogin(t *testing.T) (*app.Login, loginMocks) {
	t.Helper()
	m := loginMocks{
		admins:   mocks.NewMockAdminRepository(t),
		hasher:   mocks.NewMockPasswordHasher(t),
		sessions: mocks.NewMockSessionStore(t),
		failed:   mocks.NewMockFailedLogins(t),
	}
	return app.NewLogin(m.admins, m.hasher, m.sessions, m.failed), m
}

func TestLogin_Execute_StartsASessionForTheCorrectPassword(t *testing.T) {
	t.Parallel()
	useCase, m := newLogin(t)
	m.failed.EXPECT().Count(t.Context(), domain.EmailAddress("ann@example.com")).Return(4, time.Minute, nil).Once()
	m.admins.EXPECT().FindByEmail(t.Context(), domain.EmailAddress("ann@example.com")).Return(ann, nil).Once()
	m.hasher.EXPECT().Matches("stored-hash", "the password").Return(true).Once()
	m.sessions.EXPECT().Create(t.Context(), ann.ID, 8*time.Hour).Return("token-1", nil).Once()

	token, err := useCase.Execute(t.Context(), app.LoginInput{EmailAddress: " Ann@Example.COM ", Password: "the password"})

	require.NoError(t, err)
	assert.Equal(t, "token-1", token)
}

func TestLogin_Execute_CountsAFailureForAWrongPassword(t *testing.T) {
	t.Parallel()
	useCase, m := newLogin(t)
	m.failed.EXPECT().Count(t.Context(), domain.EmailAddress("ann@example.com")).Return(0, time.Duration(0), nil).Once()
	m.admins.EXPECT().FindByEmail(t.Context(), domain.EmailAddress("ann@example.com")).Return(ann, nil).Once()
	m.hasher.EXPECT().Matches("stored-hash", "wrong").Return(false).Once()
	m.failed.EXPECT().Add(t.Context(), domain.EmailAddress("ann@example.com"), 15*time.Minute).Return(nil).Once()

	token, err := useCase.Execute(t.Context(), app.LoginInput{EmailAddress: "ann@example.com", Password: "wrong"})

	require.ErrorIs(t, err, domain.ErrInvalidCredentials)
	assert.Empty(t, token)
}

func TestLogin_Execute_ChecksTheHashOfNoAdminForAnUnknownEmailAddressAndCountsAFailure(t *testing.T) {
	t.Parallel()
	useCase, m := newLogin(t)
	m.failed.EXPECT().Count(t.Context(), domain.EmailAddress("bob@example.com")).Return(0, time.Duration(0), nil).Once()
	m.admins.EXPECT().FindByEmail(t.Context(), domain.EmailAddress("bob@example.com")).Return(domain.Admin{}, domain.ErrAdminNotFound).Once()
	// An empty hash makes the hasher spend the same time as for a real hash. The answer is the same as for a wrong password.
	m.hasher.EXPECT().Matches("", "any password").Return(false).Once()
	m.failed.EXPECT().Add(t.Context(), domain.EmailAddress("bob@example.com"), 15*time.Minute).Return(nil).Once()

	_, err := useCase.Execute(t.Context(), app.LoginInput{EmailAddress: "bob@example.com", Password: "any password"})

	require.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestLogin_Execute_RefusesAfterFiveFailuresAndTellsWhenToTryAgain(t *testing.T) {
	t.Parallel()
	useCase, m := newLogin(t) // The other mocks have no expected call, so a call fails the test. The password is not checked.
	m.failed.EXPECT().Count(t.Context(), domain.EmailAddress("ann@example.com")).Return(5, 9*time.Minute, nil).Once()

	_, err := useCase.Execute(t.Context(), app.LoginInput{EmailAddress: "ann@example.com", Password: "the password"})

	var blocked *app.LoginBlockedError
	require.ErrorAs(t, err, &blocked)
	assert.Equal(t, 9*time.Minute, blocked.RetryAfter)
}

func TestLogin_Execute_ReturnsInvalidCredentialsWithoutAnyCallWhenTheInputCannotBeAnAdmin(t *testing.T) {
	t.Parallel()
	tests := map[string]app.LoginInput{
		"email address is not valid": {EmailAddress: "ann", Password: "the password"},
		"password is empty":          {EmailAddress: "ann@example.com", Password: ""},
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			useCase, _ := newLogin(t) // The mocks have no expected call, so a call fails the test.

			_, err := useCase.Execute(t.Context(), input)

			require.ErrorIs(t, err, domain.ErrInvalidCredentials)
		})
	}
}

func TestLogin_Execute_ReturnsTheErrorWhenTheCountOfFailuresIsNotAvailable(t *testing.T) {
	t.Parallel()
	useCase, m := newLogin(t)
	cause := errors.New("Redis is down.")
	m.failed.EXPECT().Count(t.Context(), mock.Anything).Return(0, time.Duration(0), cause).Once()

	_, err := useCase.Execute(t.Context(), app.LoginInput{EmailAddress: "ann@example.com", Password: "the password"})

	require.ErrorIs(t, err, cause)
}

func TestCurrentAdmin_Execute_GivesTheAdminOfTheSession(t *testing.T) {
	t.Parallel()
	sessions, admins := mocks.NewMockSessionStore(t), mocks.NewMockAdminRepository(t)
	sessions.EXPECT().Get(t.Context(), "token-1").Return(ann.ID, true, nil).Once()
	admins.EXPECT().FindByID(t.Context(), ann.ID).Return(ann, nil).Once()

	got, err := app.NewCurrentAdmin(sessions, admins).Execute(t.Context(), "token-1")

	require.NoError(t, err)
	assert.Equal(t, ann, got)
}

func TestCurrentAdmin_Execute_ReturnsNotLoggedInForAnUnknownSessionOrAnAdminThatIsGone(t *testing.T) {
	t.Parallel()
	t.Run("unknown session", func(t *testing.T) {
		t.Parallel()
		sessions, admins := mocks.NewMockSessionStore(t), mocks.NewMockAdminRepository(t)
		sessions.EXPECT().Get(t.Context(), "token-1").Return(uuid.Nil, false, nil).Once()

		_, err := app.NewCurrentAdmin(sessions, admins).Execute(t.Context(), "token-1")

		require.ErrorIs(t, err, domain.ErrNotLoggedIn)
	})
	t.Run("admin that is gone", func(t *testing.T) {
		t.Parallel()
		sessions, admins := mocks.NewMockSessionStore(t), mocks.NewMockAdminRepository(t)
		sessions.EXPECT().Get(t.Context(), "token-1").Return(ann.ID, true, nil).Once()
		admins.EXPECT().FindByID(t.Context(), ann.ID).Return(domain.Admin{}, domain.ErrAdminNotFound).Once()

		_, err := app.NewCurrentAdmin(sessions, admins).Execute(t.Context(), "token-1")

		require.ErrorIs(t, err, domain.ErrNotLoggedIn)
	})
}

func TestLogout_Execute_DeletesTheSession(t *testing.T) {
	t.Parallel()
	sessions := mocks.NewMockSessionStore(t)
	sessions.EXPECT().Delete(t.Context(), "token-1").Return(nil).Once()

	require.NoError(t, app.NewLogout(sessions).Execute(t.Context(), "token-1"))
}

var seedTime = time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)

func newSeedAdmin(t *testing.T) (*app.SeedAdmin, *mocks.MockAdminRepository, *mocks.MockPasswordHasher) {
	t.Helper()
	admins, hasher := mocks.NewMockAdminRepository(t), mocks.NewMockPasswordHasher(t)
	return app.NewSeedAdmin(admins, hasher, func() time.Time { return seedTime }), admins, hasher
}

func TestSeedAdmin_Execute_CreatesTheAdminWithAHashOfThePassword(t *testing.T) {
	t.Parallel()
	useCase, admins, hasher := newSeedAdmin(t)
	admins.EXPECT().FindByEmail(t.Context(), domain.EmailAddress("ann@example.com")).Return(domain.Admin{}, domain.ErrAdminNotFound).Once()
	hasher.EXPECT().Hash("a password of 12+").Return("new-hash", nil).Once()
	admins.EXPECT().
		Create(t.Context(), mock.MatchedBy(func(a domain.Admin) bool {
			return a.EmailAddress == "ann@example.com" && a.PasswordHash == "new-hash" && a.CreatedAt.Equal(seedTime)
		})).
		Return(nil).
		Once()

	created, err := useCase.Execute(t.Context(), " Ann@Example.COM ", "a password of 12+")

	require.NoError(t, err)
	assert.True(t, created)
}

func TestSeedAdmin_Execute_DoesNothingWhenTheAdminExists(t *testing.T) {
	t.Parallel()
	useCase, admins, _ := newSeedAdmin(t) // The hasher has no expected call. The password of the Admin does not change.
	admins.EXPECT().FindByEmail(t.Context(), domain.EmailAddress("ann@example.com")).Return(ann, nil).Once()

	created, err := useCase.Execute(t.Context(), "ann@example.com", "another password")

	require.NoError(t, err)
	assert.False(t, created)
}

func TestSeedAdmin_Execute_DoesNothingWhenAnotherProcessCreatesTheAdminFirst(t *testing.T) {
	t.Parallel()
	useCase, admins, hasher := newSeedAdmin(t)
	admins.EXPECT().FindByEmail(t.Context(), mock.Anything).Return(domain.Admin{}, domain.ErrAdminNotFound).Once()
	hasher.EXPECT().Hash(mock.Anything).Return("new-hash", nil).Once()
	admins.EXPECT().Create(t.Context(), mock.Anything).Return(domain.ErrEmailTaken).Once()

	created, err := useCase.Execute(t.Context(), "ann@example.com", "a password of 12+")

	require.NoError(t, err)
	assert.False(t, created)
}

func TestSeedAdmin_Execute_RejectsAnEmailAddressOrAPasswordThatBreaksARule(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		emailAddress, password string
		want                   error
	}{
		"email address is empty": {"", "a password of 12+", domain.ErrInvalidEmailAddress},
		"password is too short":  {"ann@example.com", "short", domain.ErrInvalidPassword},
		"password is too long":   {"ann@example.com", string(make([]byte, 73)), domain.ErrInvalidPassword},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			useCase, _, _ := newSeedAdmin(t) // The mocks have no expected call, so a call fails the test.

			created, err := useCase.Execute(context.Background(), tt.emailAddress, tt.password)

			require.ErrorIs(t, err, tt.want)
			assert.False(t, created)
		})
	}
}
