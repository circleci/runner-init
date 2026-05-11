//go:build !windows

package testing

import (
	"syscall"
	"testing"

	"gotest.tools/v3/assert"
)

// ReparentChildren any child processes to the current process
func ReparentChildren(t *testing.T) {
	t.Helper()

	const PrSetChildSubreaper = 36
	_, _, errno := syscall.Syscall(syscall.SYS_PRCTL, PrSetChildSubreaper, uintptr(1), 0)
	assert.Check(t, errno == 0)
}
