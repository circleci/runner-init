package task

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/circleci/ex/config/secret"
)

var configReadTimeout = 2 * time.Minute

type Config struct {
	Entrypoint []string `json:"entrypoint"`

	// Task agent configuration
	Token            secret.String `json:"token"`
	User             string        `json:"user"`
	TaskAgentPath    string        `json:"task_agent_path"`
	RunnerAPIBaseURL string        `json:"runner_api_base_url"`
	Allocation       string        `json:"allocation"`
	SSHAdvertiseAddr string        `json:"ssh_advertise_addr"`
	MaxRunTime       time.Duration `json:"max_run_time"`
}

func (c *Config) ReadFromStdin(ctx context.Context) error {
	bytesCh := make(chan []byte, 1)
	errCh := make(chan error, 1)

	go func() {
		for {
			bytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				errCh <- fmt.Errorf("failed to read config from stdin: %w", err)
				return
			}
			if len(bytes) == 0 {
				continue
			}
			bytesCh <- bytes
			return
		}
	}()

	select {
	case err := <-errCh:
		return err
	case bytes := <-bytesCh:
		if err := json.Unmarshal(bytes, c); err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}
	case <-time.After(configReadTimeout):
		return fmt.Errorf("timed out reading config from stdin: %w", ctx.Err())
	}

	return nil
}

func (c *Config) TaskAgentCmd() string {
	args := []string{
		"_internal",
		"agent-runner",
		"--verbose",
		"--runnerAPIBaseURL=" + c.RunnerAPIBaseURL,
		"--allocation=" + c.Allocation,
		"--disableSpinUpStep",
		"--disableIsolatedSSHDir",
		fmt.Sprintf("--maxRunTime=%v", c.MaxRunTime.Seconds()),
	}
	if c.SSHAdvertiseAddr != "" {
		args = append(args, "--sshAdvertiseAddr="+c.SSHAdvertiseAddr)
	}
	cmd := fmt.Sprintf("PATH=$PATH:%s %s %s", filepath.Dir(c.TaskAgentPath), c.TaskAgentPath, strings.Join(args, " "))
	return cmd
}
