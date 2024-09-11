package task

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/circleci/ex/testing/testcontext"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/circleci/runner-init/clients/runner"
	"github.com/circleci/runner-init/internal/testing/fakerunnerapi"
)

var testOnce sync.Once

func TestOrchestrator(t *testing.T) {
	if os.Getenv("BE_TASK_AGENT") == "true" {
		// If BE_TASK_AGENT is set, then this test will run itself as a fake task agent.
		// This provides a way to run checks against the task agent command.
		beFakeTaskAgent(t)
		return
	}

	testOnce.Do(func() {
		// Reduce the process reap timeout to speed up the tests
		reapTimeout = 500 * time.Millisecond
	})

	// Re-parent any child processes to us to simulate the orchestrator being init
	const PrSetChildSubreaper = 36
	_, _, err := syscall.Syscall(syscall.SYS_PRCTL, PrSetChildSubreaper, uintptr(1), 0)
	assert.Check(t, err == 0)

	testPath := os.Args[0]
	scratchDir := t.TempDir()
	defaultConfig := Config{
		Token:         "testtoken",
		TaskAgentPath: testPath + " -test.run=TestOrchestrator",
		Allocation:    "testalloc",
	}
	tests := []struct {
		name string

		config          Config
		env             map[string]string
		gracePeriod     time.Duration
		timeout         time.Duration
		additionalTasks []fakerunnerapi.Task

		wantError        string
		wantTimeout      bool
		wantTaskUnclaims []fakerunnerapi.TaskUnclaim
		wantTaskEvents   []fakerunnerapi.TaskEvent
		extraChecks      []func(t *testing.T)
	}{
		{
			name: "happy path",
			env: map[string]string{
				"CIRCLECI_GOAT_CONFIG": "{}",
			},
			config: defaultConfig,
		},
		{
			name: "custom entrypoint",
			config: Config{
				Cmd:           []string{"/bin/sh", "-c", fmt.Sprintf("touch %s/testfile", scratchDir)},
				Token:         "testtoken",
				TaskAgentPath: testPath + " -test.run=TestOrchestrator",
			},
			extraChecks: []func(t *testing.T){
				func(t *testing.T) {
					_, err := os.Stat(scratchDir + "/testfile")
					assert.NilError(t, err, "expected custom entrypoint to create file")
				},
			},
		},
		{
			name:        "finish within grace period",
			config:      defaultConfig,
			timeout:     500 * time.Millisecond,
			gracePeriod: 2 * time.Second,
			wantError:   "",
		},
		{
			name:   "zombie processes are reaped",
			config: defaultConfig,
			env: map[string]string{
				"SIMULATE_A_ZOMBIE_PROCESS": scratchDir + "/task.pid",
			},
			extraChecks: []func(t *testing.T){
				func(t *testing.T) {
					b, err := os.ReadFile(scratchDir + "/task.pid") //nolint:gosec // this is a test
					assert.NilError(t, err)

					pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
					assert.NilError(t, err)

					time.Sleep(500 * time.Millisecond)

					p, err := os.FindProcess(pid)
					assert.NilError(t, err)

					err = p.Signal(syscall.Signal(0))
					assert.Check(t, cmp.ErrorIs(err, os.ErrProcessDone))
				},
			},
		},
		{
			name:   "error: interrupted task",
			config: defaultConfig,
			env: map[string]string{
				"SIMULATE_RUNNING_A_TASK": "true",
			},
			timeout: 500 * time.Millisecond,
			wantError: "error on shutdown: task agent process is still running, " +
				"which could interrupt the task. Possible reasons include the Pod being evicted or deleted",
			wantTaskEvents: []fakerunnerapi.TaskEvent{
				{
					Allocation:     defaultConfig.Allocation,
					TimestampMilli: time.Now().UnixMilli(),
					Message: []byte("error on shutdown: task agent process is still running, " +
						"which could interrupt the task. Possible reasons include the Pod being evicted or deleted: " +
						"Check container logs for more details"),
				},
			},
		},
		{
			name:   "error: task agent panicking",
			config: defaultConfig,
			env: map[string]string{
				"SIMULATE_TASK_AGENT_PANICKING": "true",
			},
			wantError: "error while executing task agent: " +
				"task agent command exited with an unexpected error: exit status 2",
			wantTaskEvents: []fakerunnerapi.TaskEvent{
				{
					Allocation:     defaultConfig.Allocation,
					TimestampMilli: time.Now().UnixMilli(),
					Message: []byte("error while executing task agent: " +
						"task agent command exited with an unexpected error: exit status 2: " +
						"Check container logs for more details"),
				},
			},
		},
		{
			name: "retryable error: task agent failed to start",
			config: Config{
				TaskID:        "retry",
				Token:         "retry-token",
				TaskAgentPath: "thiswontstart",
			},
			wantError: "",
			wantTaskUnclaims: []fakerunnerapi.TaskUnclaim{
				{
					ID:    "retry",
					Token: "retry-token",
				},
			},
		},
		{
			name: "retryable error: an unsafe retry",
			config: Config{
				TaskID:              "retry",
				EnableUnsafeRetries: true,
				Token:               "retry-token",
				TaskAgentPath:       defaultConfig.TaskAgentPath,
			},
			env: map[string]string{
				"SIMULATE_TASK_AGENT_PANICKING": "true",
			},
			wantError: "",
			wantTaskUnclaims: []fakerunnerapi.TaskUnclaim{
				{
					ID:    "retry",
					Token: "retry-token",
				},
			},
		},
		{
			name: "error: retryable, but exhausted all retries",
			config: Config{
				TaskID:        "no-retry",
				Token:         "no-retry-token",
				TaskAgentPath: "thiswontstart",
			},
			additionalTasks: []fakerunnerapi.Task{
				{
					ID:           "no-retry",
					Token:        "no-retry-token",
					UnclaimCount: 3,
				},
			},
			wantError: "failed to retry task: exhausted all task retries",
			wantTaskUnclaims: []fakerunnerapi.TaskUnclaim{
				{
					ID:    "no-retry",
					Token: "no-retry-token",
				},
			},
			wantTaskEvents: []fakerunnerapi.TaskEvent{
				{
					TimestampMilli: time.Now().UnixMilli(),
					Message: []byte("error while executing task agent: " +
						"failed to start task agent command: " +
						"exec: thiswontstart: executable file not found in $PATH: " +
						"Check container logs for more details"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BE_TASK_AGENT", "true")

			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			ctx := testcontext.Background()
			if tt.timeout == 0 {
				tt.timeout = 20 * time.Second
			}
			ctx, cancel := context.WithTimeout(ctx, tt.timeout)
			defer cancel()

			c := tt.config

			defaultTask := fakerunnerapi.Task{
				ID:         c.TaskID,
				Token:      c.Token,
				Allocation: c.Allocation,
			}
			runnerAPI := fakerunnerapi.New(ctx, append(tt.additionalTasks, defaultTask))
			server := httptest.NewServer(runnerAPI)
			defer server.Close()

			r := runner.NewClient(runner.ClientConfig{
				BaseURL:   server.URL,
				AuthToken: c.Token,
			})

			o := NewOrchestrator(tt.config, r, tt.gracePeriod)
			err := o.Run(ctx)

			if tt.wantError != "" {
				assert.Check(t, cmp.ErrorContains(err, tt.wantError))
			} else {
				assert.NilError(t, err)
			}

			assert.Check(t, cmp.DeepEqual(runnerAPI.TaskUnclaims(), tt.wantTaskUnclaims))
			assert.Check(t, cmp.DeepEqual(runnerAPI.TaskEvents(), tt.wantTaskEvents, fakerunnerapi.CmpTaskEvent))

			for _, check := range tt.extraChecks {
				check(t)
			}
		})
	}
}

func beFakeTaskAgent(t *testing.T) {
	t.Helper()

	assert.Check(t, cmp.Equal(os.Args[2], "_internal"))
	assert.Check(t, cmp.Equal(os.Args[3], "agent-runner"))

	for _, env := range os.Environ() {
		assert.Check(t, !strings.Contains(env, "CIRCLECI_GOAT"),
			"orchestrator configuration shouldn't be in the task environment")
	}

	b, err := io.ReadAll(os.Stdin)
	assert.NilError(t, err)
	assert.Check(t, cmp.Equal(string(b), "testtoken"), "expected the task token on stdin")

	if os.Getenv("SIMULATE_RUNNING_A_TASK") == "true" {
		time.Sleep(30 * time.Second)
	}

	if os.Getenv("SIMULATE_TASK_AGENT_PANICKING") == "true" {
		panic("I'm intentionally panicking!!!")
	}

	if pidfile := os.Getenv("SIMULATE_A_ZOMBIE_PROCESS"); pidfile != "" {
		c := exec.Command("/bin/sh", "-c", "echo $$ >"+pidfile+" && sleep 300") //nolint:gosec // this is a test
		assert.NilError(t, c.Start())
	}
}
