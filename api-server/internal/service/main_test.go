package service

import (
	"os"
	"testing"

	"github.com/bison/api-server/pkg/logger"
)

// TestMain initializes the package-level logger so service tests that log do not
// hit a nil SugaredLogger.
func TestMain(m *testing.M) {
	logger.Init(false)
	os.Exit(m.Run())
}
