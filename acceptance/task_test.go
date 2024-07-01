package acceptance

import (
	"fmt"
	"testing"
	"time"

	"github.com/circleci/ex/testing/runner"
	"gotest.tools/v3/assert"
)

func TestRunTask(t *testing.T) {
	goodConfig := fmt.Sprintf(`
{
	"cmd": [],
	"enable_unsafe_retries": false,
	"token": "testtoken",
	"task_agent_path": "%v",
	"runner_api_base_url": "https://runner.circleci.com",
	"allocation": "testallocation",
	"max_run_time": 60000000000
}`, taskAgentBinary)

	r := runner.New(
		"CIRCLECI_GOAT_SHUTDOWN_DELAY=10s",
		"CIRCLECI_GOAT_CONFIG="+goodConfig,
	)
	res, err := r.Start(orchestratorTestBinaryRunTask)
	assert.NilError(t, err)

	t.Run("Probe for readiness", func(t *testing.T) {
		assert.NilError(t, res.Ready("admin", time.Second*20))
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
