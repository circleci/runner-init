package task

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/circleci/ex/config/secret"
	"github.com/goccy/go-json"
)

type Config struct {
	Cmd                 []string `json:"cmd"`
	User                string   `json:"user"`
	EnableUnsafeRetries bool     `json:"enable_unsafe_retries"`

	// Task agent configuration
	Token            secret.String `json:"token"`
	TaskAgentPath    string        `json:"task_agent_path"`
	RunnerAPIBaseURL string        `json:"runner_api_base_url"`
	Allocation       string        `json:"allocation"`
	SSHAdvertiseAddr string        `json:"ssh_advertise_addr"`
	MaxRunTime       time.Duration `json:"max_run_time"`

	TokenChecksum string `json:"token_checksum"`
}

func (c *Config) ReadFrom(ctx context.Context, in io.Reader, timeout time.Duration) error {
	bytesCh := make(chan []byte, 1)
	errCh := make(chan error, 1)

	go func() {
		for {
			bytes, err := io.ReadAll(in)
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
	case <-time.After(timeout):
		return fmt.Errorf("timed out reading config from stdin: %w", ctx.Err())
	}

	if !c.validateTokenChecksum() {
		return fmt.Errorf("invalid checksum on config token")
	}

	return nil
}

func (c *Config) validateTokenChecksum() bool {
	hasher := sha256.New()
	hasher.Write([]byte(c.Token.Raw()))

	return c.TokenChecksum == hex.EncodeToString(hasher.Sum(nil))
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

	env := []string{fmt.Sprintf("PATH=$PATH:%s", filepath.Dir(c.TaskAgentPath))}

	return Agent{
		Cmd: cmd,
		Env: env,
	}
}
