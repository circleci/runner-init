package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/circleci/ex/o11y"
)

type Command struct {
	cmd            *exec.Cmd
	stderrSaver    *prefixSuffixSaver
	isStarted      atomic.Bool
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

	if err := c.cmd.Process.Signal(syscall.Signal(0)); err == nil {
		return true, nil

	} else if !errors.Is(err, os.ErrProcessDone) {
		return false, fmt.Errorf("unexpected error from signaling process: %w", err)
	}

	return false, nil
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

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: syscall.SIGKILL,
	}

	cmd.Cancel = func() error {
		// Kill the child process group
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	if user != "" {
		maybeSwitchUser(ctx, cmd, user)
	}

	return cmd
}

func maybeSwitchUser(ctx context.Context, cmd *exec.Cmd, username string) {
	usr, err := user.Lookup(username)

	if err == nil {
		cmd.Env = append(cmd.Env, "HOME="+usr.HomeDir)

		uid, _ := strconv.Atoi(usr.Uid)
		gid, _ := strconv.Atoi(usr.Gid)
		//nolint:gosec // G115: we only support POSIX right now, so this won't overflow
		cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}
	} else {
		o11y.LogError(ctx, "failed to lookup user", err, o11y.Field("username", username))
	}
}

func notifySignals(cmd *exec.Cmd) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		for sig := range ch {
			_ = cmd.Process.Signal(sig)
		}
	}()
}
