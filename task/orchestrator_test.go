package task

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/circleci/ex/testing/testcontext"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

var testOnce sync.Once

func TestOrchestrator(t *testing.T) {
	if os.Getenv("BE_TASK_AGENT") == "true" {
		// If BE_TASK_AGENT is set, then this test will run itself as a fake task agent.
		// This provides a way to run checks against the task agent command.
		beTaskAgent(t)
		return
	}

	testOnce.Do(func() {
		// Set the process reap time to 0 to prevent interference between test cases
		reapTime = 0
	})

	testPath := os.Args[0]
	scratchDir := t.TempDir()
	defaultConfig := Config{
		Token:         "testtoken",
		TaskAgentPath: testPath + " -test.run=TestOrchestrator",
	}
	tests := []struct {
		name string

		config      Config
		env         map[string]string
		gracePeriod time.Duration
		timeout     time.Duration

		wantError   string
		wantTimeout bool
		extraChecks []func(t *testing.T)
	}{
		{
			name: "happy path",
			env: map[string]string{
				"CIRCLECI_GOAT_CONFIG": "{}",
			},
			config: defaultConfig,
		},
		{
			name: "custom command",
			config: Config{
				Cmd:           []string{"/bin/sh", "-c", fmt.Sprintf("touch %s/testfile", scratchDir)},
				Token:         "testtoken",
				TaskAgentPath: testPath + " -test.run=TestOrchestrator",
			},
			extraChecks: []func(t *testing.T){
				func(t *testing.T) {
					_, err := os.Stat(scratchDir + "/testfile")
					assert.NilError(t, err, "expected custom command to create file")
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
			name:   "error: interrupted task",
			config: defaultConfig,
			env: map[string]string{
				"SIMULATE_RUNNING_A_TASK": "true",
			},
			timeout:   500 * time.Millisecond,
			wantError: "task agent process is still running and may interrupt the task",
		},
		{
			name:   "error: task agent misbehaving",
			config: defaultConfig,
			env: map[string]string{
				"SIMULATE_TASK_AGENT_MISBEHAVING": "true",
			},
			wantError: "error while executing task agent: " +
				"task agent command exited with an unexpected error: exit status 123",
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

			o := NewOrchestrator(tt.config, tt.gracePeriod)
			err := o.Run(ctx)

			if tt.wantError != "" {
				assert.Check(t, cmp.ErrorContains(err, tt.wantError))
			} else {
				assert.NilError(t, err)
			}

			for _, check := range tt.extraChecks {
				check(t)
			}
		})
	}
}

func beTaskAgent(t *testing.T) {
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

	if os.Getenv("SIMULATE_TASK_AGENT_MISBEHAVING") == "true" {
		os.Exit(123)
	}
}
