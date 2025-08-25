package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"syscall"
)

type Command struct {
	cmd            *exec.Cmd
	stderrSaver    *prefixSuffixSaver
	isStarted      atomic.Bool
	isCompleted    atomic.Bool
	forwardSignals bool
	waitCh         chan error
}

func New(ctx context.Context, cmd []string, forwardSignals bool, user string, env ...string) Command {
	s := &prefixSuffixSaver{N: 160}
	return Command{
		cmd:            newCmd(ctx, cmd, user, s, env...),
		stderrSaver:    s,
		forwardSignals: forwardSignals,
		waitCh:         make(chan error, 1),
	}
}

func (c *Command) Start() error {
	cmd := c.cmd

	if err := c.start(); err != nil {
		return err
	}

	if cmd.Process == nil {
		return fmt.Errorf("no underlying process")
	}

	go func() {
		c.waitCh <- c.wait()
	}()

	if c.forwardSignals {
		forwardSignals(cmd)
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
	return <-c.waitCh
}

func (c *Command) wait() error {
	cmd := c.cmd
	defer func() {
		_ = cmd.Cancel()

		c.isCompleted.Store(cmd.ProcessState != nil)
	}()

	err := cmd.Wait()
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

	return !c.isCompleted.Load(), nil
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

	cmd.SysProcAttr = &syscall.SysProcAttr{}

	if user != "" {
		switchUser(ctx, cmd, user)
	}

	additionalSetup(ctx, cmd)

	return cmd
}
