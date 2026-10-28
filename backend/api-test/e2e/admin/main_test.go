//go:build e2e

package admin_test

import (
	"os"
	"testing"

	"github.com/rudradcruze/go-react-template/backend/api-test/e2e/testenv"
)

func TestMain(m *testing.M) {
	os.Exit(testenv.Main(m))
}
