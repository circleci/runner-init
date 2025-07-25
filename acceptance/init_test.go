package acceptance

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/circleci/ex/testing/runner"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestInit(t *testing.T) {
	srcDir := createMockSourceFiles(t)
	destDir := t.TempDir()
	orchSrc := path(t, srcDir, "orchestrator")
	orchDest := path(t, destDir, "orchestrator")
	agentSrc := path(t, srcDir, "circleci-agent")
	agentDest := path(t, destDir, "circleci-agent")
	circleciDest := path(t, destDir, "circleci")

	r := runner.New(
		"SOURCE="+srcDir,
		"DESTINATION="+destDir,
		"CIRCLECI_GOAT_SHUTDOWN_DELAY=0",
	)
	res, err := r.Start(orchestratorTestBinary)
	assert.NilError(t, err)

	t.Run("Run init", func(t *testing.T) {
		select {
		case err = <-res.Wait():
			assert.NilError(t, err)
		case <-time.After(time.Second * 5):
			assert.NilError(t, res.Stop())
			t.Fatal(t, "timeout before process stopped")
		}
	})

	t.Run("Files were copied and symlink created", func(t *testing.T) {
		assertFileIsCopied(t, orchSrc, orchDest)
		assertFileIsCopied(t, agentSrc, agentDest)

		if runtime.GOOS == "windows" {
			assertFileIsCopied(t, agentSrc, circleciDest)
		} else {
			agentLink, err := os.Readlink(circleciDest)
			assert.NilError(t, err)
			assert.Check(t, cmp.DeepEqual(agentLink, agentDest))
		}
	})
}

// Mock source file creation for testing purposes
func createMockSourceFiles(t *testing.T) string {
	t.Helper()

	srcDir := t.TempDir()

	err := os.WriteFile(path(t, srcDir, "orchestrator"), []byte("mock orchestrator data"), 0600)
	assert.NilError(t, err)

	err = os.WriteFile(path(t, srcDir, "circleci-agent"), []byte("mock agent data"), 0600)
	assert.NilError(t, err)

	return srcDir
}

func assertFileIsCopied(t *testing.T, src, dest string) {
	t.Helper()

	srcInfo, err := os.Stat(src)
	assert.NilError(t, err)
	destInfo, err := os.Stat(dest)
	assert.NilError(t, err)
	assert.Check(t, cmp.DeepEqual(srcInfo.Mode().Perm(), destInfo.Mode().Perm()), "files should have same permissions")

	srcContents, err := os.ReadFile(src) //#nosec:G304 // this is trusted input
	assert.NilError(t, err)
	destContents, err := os.ReadFile(dest) //#nosec:G304 // this is trusted input
	assert.NilError(t, err)
	assert.Check(t, cmp.DeepEqual(srcContents, destContents), "files should have same contents")
}

func path(t *testing.T, a, b string) string {
	t.Helper()

	p := filepath.Join(a, b)

	if runtime.GOOS == "windows" {
		return p + ".exe"
	}
	return p
}
