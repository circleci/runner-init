package acceptance

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/circleci/ex/testing/runner"
	"github.com/circleci/ex/testing/testcontext"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/circleci/runner-init/internal/testing/fakerunnerapi"
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

	t.Run("Good run-task", func(t *testing.T) {
		r := runner.New(
			"CIRCLECI_GOAT_SHUTDOWN_DELAY=10s",
			"CIRCLECI_GOAT_CONFIG="+goodConfig,
			"CIRCLECI_GOAT_HEALTH_CHECK_ADDR=:7624",
		)
		res, err := r.Start(orchestratorTestBinaryRunTask)
		assert.NilError(t, err)
		defer func() { t.Log(res.Logs()) }()

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
	})

	t.Run("Good entrypoint override", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Not supported on Windows")
		}

		entrypointPath := filepath.ToSlash(filepath.Join(t.TempDir(), "entrypoint.sh"))

		//nolint:gosec
		err := os.WriteFile(entrypointPath, []byte(`#!/bin/bash
	echo "Executing custom entrypoint"
	exec "$@"`), 0750)
		assert.NilError(t, err)

		r := runner.New(
			"CIRCLECI_GOAT_ENTRYPOINT="+entrypointPath,
			"CIRCLECI_GOAT_SHUTDOWN_DELAY=10s",
			"CIRCLECI_GOAT_CONFIG="+goodConfig,
			"CIRCLECI_GOAT_HEALTH_CHECK_ADDR=:7624",
		)
		res, err := r.Start(orchestratorTestBinaryOverride)
		defer func() { t.Log(res.Logs()) }()

		t.Run("Probe for readiness", func(t *testing.T) {
			assert.NilError(t, res.Ready("admin", time.Second*20))
		})

		t.Run("Custom entrypoint ran", func(t *testing.T) {
			assert.Check(t, cmp.Contains(res.Logs(), "Executing custom entrypoint"))
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
	})

	t.Run("task-agent exits with an error", func(t *testing.T) {
		ctx := testcontext.Background()
		runnerAPI := fakerunnerapi.New(ctx, []fakerunnerapi.Task{
			{
				Token:      "testtoken",
				Allocation: "testallocation",
			},
		})
		s := httptest.NewServer(runnerAPI)
		t.Cleanup(s.Close)

		badConfig := fmt.Sprintf(`
	{
		"cmd": [],
		"enable_unsafe_retries": false,
		"token": "testtoken",
		"task_agent_path": "%v --bad-flag",
		"runner_api_base_url": "%v",
		"allocation": "testallocation",
		"max_run_time": 60000000000
	}`, strings.ReplaceAll(taskAgentBinary, `\`, `\\`), s.URL)

		r := runner.New(
			"CIRCLECI_GOAT_SHUTDOWN_DELAY=10s",
			"CIRCLECI_GOAT_CONFIG="+badConfig,
			"CIRCLECI_GOAT_HEALTH_CHECK_ADDR=:7624",
		)
		res, err := r.Start(orchestratorTestBinaryRunTask)
		assert.NilError(t, err)
		defer func() { t.Log(res.Logs()) }()

		t.Run("Probe for readiness", func(t *testing.T) {
			assert.NilError(t, res.Ready("admin", time.Second*20))
		})

		t.Run("Run task", func(t *testing.T) {
			select {
			case err = <-res.Wait():
				assert.NilError(t, err, "handled errors should have a clean exit")
			case <-time.After(time.Second * 40):
				assert.NilError(t, res.Stop())
				t.Fatal(t, "timeout before process stopped")
			}
		})

		t.Run("Task failed", func(t *testing.T) {
			assert.Check(t, cmp.Len(runnerAPI.TaskUnclaims(), 0))
			assert.Check(t, cmp.DeepEqual(runnerAPI.TaskEvents(), []fakerunnerapi.TaskEvent{
				{
					Allocation:     "testallocation",
					TimestampMilli: time.Now().UnixMilli(),
					Message: []byte("error while executing task agent: " +
						"task agent command exited with an unexpected error: " +
						"exit status 80: circleci-agent: error: unknown flag --bad-flag\n: " +
						"Check container logs for more details"),
				},
			}, fakerunnerapi.CmpTaskEvent))
		})
	})

	t.Run("task-agent fails to start", func(t *testing.T) {
		ctx := testcontext.Background()
		runnerAPI := fakerunnerapi.New(ctx, []fakerunnerapi.Task{
			{
				ID:         "testid",
				Token:      "testtoken",
				Allocation: "testallocation",
			},
		})
		s := httptest.NewServer(runnerAPI)
		t.Cleanup(s.Close)

		badConfig := fmt.Sprintf(`
	{
		"cmd": [],
		"enable_unsafe_retries": false,
		"task_id": "testid",
		"token": "testtoken",
		"task_agent_path": "thiswillfailtostart",
		"runner_api_base_url": "%v",
		"allocation": "testallocation",
		"max_run_time": 60000000000
	}`, s.URL)

		r := runner.New(
			"CIRCLECI_GOAT_SHUTDOWN_DELAY=10s",
			"CIRCLECI_GOAT_CONFIG="+badConfig,
			"CIRCLECI_GOAT_HEALTH_CHECK_ADDR=:7624",
		)
		res, err := r.Start(orchestratorTestBinaryRunTask)
		assert.NilError(t, err)
		defer func() { t.Log(res.Logs()) }()

		t.Run("Probe for readiness", func(t *testing.T) {
			assert.NilError(t, res.Ready("admin", time.Second*20))
		})

		t.Run("Run task", func(t *testing.T) {
			select {
			case err = <-res.Wait():
				assert.NilError(t, err, "handled errors should have a clean exit")
			case <-time.After(time.Second * 40):
				assert.NilError(t, res.Stop())
				t.Fatal(t, "timeout before process stopped")
			}
		})

		t.Run("Task unclaimed - this is a safe error to retry", func(t *testing.T) {
			assert.Check(t, cmp.Len(runnerAPI.TaskUnclaims(), 1))
			assert.Check(t, cmp.Len(runnerAPI.TaskEvents(), 0))
		})
	})
}
