package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/circleci/ex/config/secret"
	"github.com/goccy/go-json"
)

type Config struct {
	PreCmd              []string `json:"pre_cmd"`
	Cmd                 []string `json:"cmd"`
	PostCmd             []string `json:"post_cmd"`
	User                string   `json:"user"`
	TaskID              string   `json:"task_id"`
	EnableUnsafeRetries bool     `json:"enable_unsafe_retries"`

	// Task agent configuration
	Token            secret.String `json:"token"`
	TaskAgentPath    string        `json:"task_agent_path"`
	RunnerAPIBaseURL string        `json:"runner_api_base_url"`
	Allocation       string        `json:"allocation"`
	SSHAdvertiseAddr string        `json:"ssh_advertise_addr"`
	MaxRunTime       time.Duration `json:"max_run_time"`
}

func (c *Config) UnmarshalJSON(b []byte) error {
	type tmpConfig Config
	var tc tmpConfig

	if err := json.Unmarshal(b, &tc); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	*c = Config(tc)

	return nil
}

type Agent struct {
	Cmd []string
	Env []string
}

func (c *Config) Agent() Agent {
	args := []string{
		"_internal",
		"agent-runner",
		"--verbose",
		"--runnerAPIBaseURL=" + c.RunnerAPIBaseURL,
		"--allocation=" + c.Allocation,
		"--disableSpinUpStep",
		"--disableIsolatedSSHDir",
		fmt.Sprintf("--maxRunTime=%v", c.MaxRunTime),
	}
	if c.SSHAdvertiseAddr != "" {
		args = append(args, "--sshAdvertiseAddr="+c.SSHAdvertiseAddr)
	}

	cmd := append(strings.Split(c.TaskAgentPath, " "), args...)

	env := []string{fmt.Sprintf("PATH=%s:%s", os.Getenv("PATH"), filepath.Dir(c.TaskAgentPath))}

	return Agent{
		Cmd: cmd,
		Env: env,
	}
}
