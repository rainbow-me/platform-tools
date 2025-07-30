package test

import (
	"testing"

	"go.uber.org/zap/zaptest"

	"github.com/rainbow-me/platform-tools/common/logger"
)

// NewLogger returns a logger that only prints if a test fails
func NewLogger(t *testing.T) *logger.Logger {
	return logger.NewLogger(zaptest.NewLogger(t))
}
