package cmd

import (
	"context"
	"fmt"
	"github.com/circleci/ex/o11y"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync/atomic"
)

type Command struct {
	cmd            *exec.Cmd
	stderrSaver    *prefixSuffixSaver
	isStarted      atomic.Bool
	isDone         atomic.Bool
	forwardSignals bool
}

func New(ctx context.Context, cmd []string, forwardSignals bool, user string, env ...string) Command {
	s := &prefixSuffixSaver{N: 160}
	return Command{
		cmd:            newCmd(ctx, cmd, user, s, env...),
		stderrSaver:    s,
		forwardSignals: forwardSignals,
	}
}

func (c *Command) Start() error {
	cmd := c.cmd

	if err := cmd.Start(); err != nil {
		return err
	}

	if cmd.Process == nil {
		return fmt.Errorf("no underlying process")
	}

	if c.forwardSignals {
		notifySignals(cmd)
	}

	c.isStarted.Store(true)

	return nil
}

func (c *Command) StartWithStdin(b []byte) error {
	w, err := c.cmd.StdinPipe()

	if err != nil {
		return fmt.Errorf("unexpected error on stdin pipe: %w", err)
	}
	defer func() {
		_ = w.Close()
	}()

	_, err = w.Write(b)
	if err != nil {
		return fmt.Errorf("failed to write to stdin pipe: %w", err)
	}

	return c.Start()
}

func (c *Command) Wait() error {
	cmd := c.cmd
	defer func() {
		_ = cmd.Cancel()
	}()

	err := cmd.Wait()
	c.isDone.Store(cmd.ProcessState != nil)
	if err != nil {
		stderr := c.stderrSaver.Bytes()
		if len(stderr) > 0 {
			return fmt.Errorf("%w: %s", err, string(stderr))
		}
	}
	return err
}

func (c *Command) IsRunning() (bool, error) {
	if !c.isStarted.Load() {
		return false, nil
	}

	return !c.isDone.Load(), nil
}

func newCmd(ctx context.Context, argv []string, user string, stderrSaver *prefixSuffixSaver, env ...string) *exec.Cmd {
	//#nosec:G204 // this is intentionally setting up a command
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "CIRCLECI_GOAT") {
			// Prevent internal configuration from being injected in the command environment
			continue
		}
		cmd.Env = append(cmd.Env, env)
	}
	if env != nil {
		cmd.Env = append(cmd.Env, env...)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, stderrSaver)

	if user != "" {
		o11y.Log(ctx, "switching users is unsupported on windows", o11y.Field("user", user))
	}

	return cmd
}

func notifySignals(cmd *exec.Cmd) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	go func() {
		for range ch {
			_ = cmd.Process.Kill()
		}
	}()
}
