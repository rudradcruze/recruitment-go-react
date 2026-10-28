package infra

import (
	"golang.org/x/crypto/bcrypt"

	"github.com/rudradcruze/go-react-template/backend/internal/admin/app"
)

var _ app.PasswordHasher = (*BcryptHasher)(nil)

// absentHash is a bcrypt hash, at the default cost, of a password that nobody knows.
// Matches checks it when it has no hash to check.
var absentHash = []byte("$2a$10$eRKDi//osld.koqaf3SjX.Qn/Yrqcp9zJ3evPJJ640ASSm/NE5l/K")

// BcryptHasher makes and checks password hashes with bcrypt.
type BcryptHasher struct{}

// NewBcryptHasher builds a hasher.
func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{}
}

// Hash makes the bcrypt hash of the password. The password has 72 bytes or fewer.
func (h *BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

// Matches tells if the password has this hash. For an empty hash, it checks the password against
// a hash that never matches, so the call takes the same time as for a real hash.
func (h *BcryptHasher) Matches(hash, password string) bool {
	if hash == "" {
		_ = bcrypt.CompareHashAndPassword(absentHash, []byte(password))
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
