package acceptance

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/circleci/ex/testing/runner"
	"gotest.tools/v3/assert"
)

func TestRunTask(t *testing.T) {
	readinessFilePath := filepath.Join(t.TempDir(), "ready")
	goodConfig := fmt.Sprintf(`
{
	"cmd": [],
	"enable_unsafe_retries": false,
	"token": "testtoken",
	"readiness_file_path": "%v",
	"task_agent_path": "%v",
	"runner_api_base_url": "https://runner.circleci.com",
	"allocation": "testallocation",
	"max_run_time": 60000000000
}`, strings.ReplaceAll(readinessFilePath, `\`, `\\`), strings.ReplaceAll(taskAgentBinary, `\`, `\\`))

	r := runner.New(
		"CIRCLECI_GOAT_SHUTDOWN_DELAY=10s",
		"CIRCLECI_GOAT_CONFIG="+goodConfig,
		"CIRCLECI_GOAT_HEALTH_CHECK_ADDR=:7624",
	)
	res, err := r.Start(orchestratorTestBinaryRunTask)
	assert.NilError(t, err)

	t.Run("Probe for readiness", func(t *testing.T) {
		assert.NilError(t, res.Ready("admin", time.Second*20))
	})

	go func() {
		f, err := os.Create(readinessFilePath) //nolint:gosec
		defer func() { assert.NilError(t, f.Close()) }()
		assert.NilError(t, err)
	}()

	t.Run("Run task", func(t *testing.T) {
		select {
		case err = <-res.Wait():
			assert.NilError(t, err)
		case <-time.After(time.Second * 40):
			assert.NilError(t, res.Stop())
			t.Fatal(t, "timeout before process stopped")
		}
	})

	// TODO: Add more test cases...
}
