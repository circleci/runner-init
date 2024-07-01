package task

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/circleci/ex/config/secret"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func Test_UnmarshalJSON(t *testing.T) {
	goodConfig := `
{
	"cmd": [],
	"enable_unsafe_retries": false,
	"token": "testtoken",
	"task_agent_path": "/path/to/agent",
	"runner_api_base_url": "https://example.com/api",
	"allocation": "testallocation",
	"ssh_advertise_addr": "192.168.1.1",
	"max_run_time": 60000000000
}`
	tests := []struct {
		name string

		rawConfig string

		wantConfig Config
		wantError  string
	}{
		{
			name:      "valid",
			rawConfig: goodConfig,
			wantConfig: Config{
				Cmd:              []string{},
				Token:            secret.String("testtoken"),
				TaskAgentPath:    "/path/to/agent",
				RunnerAPIBaseURL: "https://example.com/api",
				Allocation:       "testallocation",
				SSHAdvertiseAddr: "192.168.1.1",
				MaxRunTime:       time.Duration(60000000000),
			},
		},
		{
			name:      "invalid",
			rawConfig: `not a valid JSON string`,
			wantError: "failed to unmarshal config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{}
			err := config.UnmarshalJSON([]byte(tt.rawConfig))

			if tt.wantError == "" {
				assert.NilError(t, err)
			} else {
				assert.Check(t, cmp.ErrorContains(err, tt.wantError))
			}
		})
	}
}

func Test_Agent(t *testing.T) {
	config := &Config{
		TaskAgentPath:    "/path/to/agent",
		RunnerAPIBaseURL: "https://example.com/api",
		Allocation:       "testallocation",
		MaxRunTime:       60 * time.Minute,
		SSHAdvertiseAddr: "192.168.1.1",
	}

	expectedCmd := []string{
		"/path/to/agent",
		"_internal",
		"agent-runner",
		"--verbose",
		"--runnerAPIBaseURL=https://example.com/api",
		"--allocation=testallocation",
		"--disableSpinUpStep",
		"--disableIsolatedSSHDir",
		"--maxRunTime=1h0m0s",
		"--sshAdvertiseAddr=192.168.1.1",
	}

	expectedEnv := []string{
		fmt.Sprintf("PATH=%s:/path/to", os.Getenv("PATH")),
	}

	expectedAgent := Agent{
		Cmd: expectedCmd,
		Env: expectedEnv,
	}

	agent := config.Agent()

	assert.Check(t, cmp.DeepEqual(agent, expectedAgent))
}
