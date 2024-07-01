package task

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/circleci/ex/testing/testcontext"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestOrchestrator(t *testing.T) {
	if os.Getenv("BE_TASK_AGENT") == "true" {
		// If BE_TASK_AGENT is set, then this test will run itself as a fake task agent.
		// This provides a way to run checks against the task agent command.
		beTaskAgent(t)
		return
	}

	testPath := os.Args[0]
	scratchDir := t.TempDir()
	defaultConfig := fmt.Sprintf(`
{
	"token": "testtoken",
	"task_agent_path": "%s -test.run=TestOrchestrator",
	"token_checksum": "ada63e98fe50eccb55036d88eda4b2c3709f53c2b65bc0335797067e9a2a5d8b"
}`, testPath)

	tests := []struct {
		name string

		config  string
		env     map[string]string
		timeout time.Duration

		wantRunError     string
		wantCleanupError string
		wantTimeout      bool
		extraChecks      []func(t *testing.T)
	}{
		{
			name:   "happy path",
			config: defaultConfig,
		},
		{
			name: "custom command",
			config: fmt.Sprintf(`
{
	"cmd": ["/bin/sh", "-c", "touch %s/testfile"],
	"token": "testtoken",
	"task_agent_path": "%s -test.run=TestOrchestrator",
	"token_checksum": "ada63e98fe50eccb55036d88eda4b2c3709f53c2b65bc0335797067e9a2a5d8b"
}`, scratchDir, testPath),
			extraChecks: []func(t *testing.T){
				func(t *testing.T) {
					_, err := os.Stat(scratchDir + "/testfile")
					assert.NilError(t, err, "expected custom command to create file")
				},
			},
		},
		{
			name:         "invalid config",
			config:       `a bad config`,
			wantRunError: "failed setup for task: failed to unmarshal config",
		},
		{
			name:   "interrupted task",
			config: defaultConfig,
			env: map[string]string{
				"SIMULATE_RUNNING_A_TASK": "true",
			},
			timeout:          500 * time.Millisecond,
			wantCleanupError: "error on shutdown: task agent process is still running and may interrupt the task",
		},
		{
			name:   "task agent misbehaving",
			config: defaultConfig,
			env: map[string]string{
				"SIMULATE_TASK_AGENT_MISBEHAVING": "true",
			},
			wantRunError: "error while executing task agent: " +
				"task agent command exited with an unexpected error: exit status 123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BE_TASK_AGENT", "true")

			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			r, w, _ := os.Pipe()
			os.Stdin = r

			go func() {
				// Load config in background
				time.Sleep(100 * time.Millisecond)

				_, err := w.Write([]byte(tt.config))
				assert.NilError(t, err)
				assert.NilError(t, w.Close())
			}()

			ctx := testcontext.Background()
			if tt.timeout == 0 {
				tt.timeout = 20 * time.Second
			}
			ctx, cancel := context.WithTimeout(ctx, tt.timeout)
			defer cancel()

			o := NewOrchestrator(r)

			done := make(chan error, 1)
			go func() {
				done <- o.Run(ctx)
			}()

			select {
			case err := <-done:
				if tt.wantRunError != "" {
					assert.Check(t, cmp.ErrorContains(err, tt.wantRunError))
				} else {
					assert.NilError(t, err)
				}
			case <-ctx.Done():
			}

			err := o.Cleanup(ctx)
			if tt.wantCleanupError != "" {
				assert.Check(t, cmp.ErrorContains(err, tt.wantCleanupError))
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
