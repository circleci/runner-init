package init

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestRun(t *testing.T) {
	srcDir := createMockSourceFiles(t)
	destDir := t.TempDir()
	orchSrc := filepath.Join(srcDir, binOrchestrator)
	orchDest := filepath.Join(destDir, binOrchestrator)
	agentSrc := filepath.Join(srcDir, binCircleciAgent)
	agentDest := filepath.Join(destDir, binCircleciAgent)
	circleciDest := filepath.Join(destDir, binCircleci)

	t.Run("Copy files and create symlink", func(t *testing.T) {
		err := Run(srcDir, destDir)
		assert.NilError(t, err)

		assertFileIsCopied(t, orchSrc, orchDest)
		assertFileIsCopied(t, agentSrc, agentDest)

		agentLink, errLink := os.Readlink(circleciDest)
		assert.NilError(t, errLink)
		assert.Check(t, cmp.DeepEqual(agentLink, agentDest))
	})

	t.Run("Fail when source files not present", func(t *testing.T) {
		err := Run(srcDir, "non-existent-dir")
		if runtime.GOOS == "windows" {
			assert.Check(t, cmp.ErrorContains(err, "The system cannot find the path specified"))
		} else {
			assert.Check(t, cmp.ErrorContains(err, "no such file or directory"))
		}
	})
}

// Mock source file creation for testing purposes
func createMockSourceFiles(t *testing.T) string {
	t.Helper()

	srcDir := t.TempDir()

	err := os.WriteFile(filepath.Join(srcDir, binOrchestrator), []byte("mock orchestrator data"), 0600)
	assert.NilError(t, err)

	err = os.WriteFile(filepath.Join(srcDir, binCircleciAgent), []byte("mock agent data"), 0600)
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
