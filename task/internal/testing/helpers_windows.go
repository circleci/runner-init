package testing

import (
	"testing"
)

// ReparentChildren is a no-op on Windows
func ReparentChildren(t *testing.T) {
	t.Helper()
}
