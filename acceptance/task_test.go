package acceptance

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/circleci/ex/testing/runner"
	"gotest.tools/v3/assert"
)

func TestRunTask(t *testing.T) {
	in := stdin(t)
	r := runner.New(
		"SHUTDOWN_DELAY=10s",
		"STDIN="+in.Name(),
	)
	res, err := r.Start(orchestratorTestBinaryRunTask)
	assert.NilError(t, err)

	goodConfig := fmt.Sprintf(`
{
	"cmd": [],
	"enable_unsafe_retries": false,
	"token": "testtoken",
	"task_agent_path": "%v",
	"runner_api_base_url": "https://runner.circleci.com",
	"allocation": "testallocation",
	"max_run_time": 60000000000,
	"token_checksum": "ada63e98fe50eccb55036d88eda4b2c3709f53c2b65bc0335797067e9a2a5d8b"
}`, taskAgentBinary)

	t.Run("Probe for readiness", func(t *testing.T) {
		assert.NilError(t, res.Ready("admin", time.Second*20))
	})

	t.Run("Load config", func(t *testing.T) {
		_, err := in.Write([]byte(goodConfig))
		assert.NilError(t, err)
	})

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

func stdin(t *testing.T) *os.File {
	t.Helper()

	f, err := os.Create(filepath.Join(t.TempDir(), "fakestdin")) //#nosec:G304 // this is just for testing
	assert.NilError(t, err)
	t.Cleanup(func() {
		assert.NilError(t, f.Close())
	})

	return f
}
