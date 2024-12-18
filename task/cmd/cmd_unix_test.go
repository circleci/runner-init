//go:build !windows

package cmd

import (
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/circleci/ex/testing/testcontext"
	"gotest.tools/v3/assert"
)

func TestCommand_notifySignals(t *testing.T) {
	scratchDir := t.TempDir()
	ctx := testcontext.Background()
	cmd := New(ctx, []string{"/bin/sh", "-c", fmt.Sprintf("trap 'touch %s/sighup' HUP; sleep 1", scratchDir)}, true, "")

	err := cmd.Start()
	assert.NilError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Send a SIGHUP to ourselves
	err = syscall.Kill(os.Getpid(), syscall.SIGHUP)
	assert.NilError(t, err)

	err = cmd.Wait()
	assert.NilError(t, err)

	// Check that the SIGHUP was forwarded and handled
	_, err = os.Stat(scratchDir + "/sighup")
	assert.NilError(t, err)
}
