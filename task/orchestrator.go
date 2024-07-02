package task

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/circleci/ex/o11y"
)

type Orchestrator struct {
	config      Config
	gracePeriod time.Duration

	ready      atomic.Bool
	agentPid   atomic.Int64
	cancelTask context.CancelFunc
	cancelMu   sync.RWMutex
}

func NewOrchestrator(config Config, gracePeriod time.Duration) *Orchestrator {
	return &Orchestrator{
		config:      config,
		gracePeriod: gracePeriod,
		cancelTask:  func() {},
	}
}

func (o *Orchestrator) Run(parentCtx context.Context) (err error) {
	ctx := o.taskContext(parentCtx)

	defer func() {
		err = errors.Join(err, o.cleanup(ctx))
	}()

	o.setup(ctx)

	errCh := make(chan error, 1)
	go func() {
		if err := o.executeAgent(ctx); err != nil {
			errCh <- fmt.Errorf("error while executing task agent: %w", err)
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-parentCtx.Done():
		// If the parent context is canceled, wait for the termination grace period before shutting down.
		// This is in case the task completes within that period.
		select {
		case err := <-errCh:
			return err
		case <-time.After(o.gracePeriod):
			o11y.Log(ctx, "termination grace period is over")
			return err
		}
	}
}

func (o *Orchestrator) taskContext(ctx context.Context) context.Context {
	o.cancelMu.Lock()
	defer o.cancelMu.Unlock()

	// Copy the O11y provider to a new context that can be separately cancelled.
	// This ensures we can drain the task on shutdown of the agent even if the parent context was cancelled,
	// but still make sure any task resources are released.
	ctx, o.cancelTask = context.WithCancel(o11y.WithProvider(context.Background(), o11y.FromContext(ctx)))
	return ctx
}

func (o *Orchestrator) setup(ctx context.Context) {
	if len(o.config.Cmd) > 0 {
		// If a custom command is specified, execute it in the background
		go o.executeCmd(ctx)
	}

	// Signal the orchestrator is ready and will start the task agent process
	o.ready.Store(true)
}

func (o *Orchestrator) executeCmd(ctx context.Context) {
	cmd := o.config.Cmd
	c := o.newCmd(ctx, cmd)
	if err := c.Run(); err != nil {
		o11y.LogError(ctx, "error running custom command", err, o11y.Field("cmd", cmd))
	}
}

func (o *Orchestrator) executeAgent(ctx context.Context) error {
	agent := o.config.Agent()
	c := o.newCmd(ctx, agent.Cmd, agent.Env...)

	if err := o.loadToken(c); err != nil {
		return retryableErrorf("failed to load task token: %w", err)
	}

	// Start and wait for the task agent process to exit
	if err := c.Start(); err != nil {
		return retryableErrorf("failed to start task agent command: %w", err)
	}
	if c.Process == nil {
		return retryableErrorf("no process associated with task agent command")
	}
	// Store the task agent PID so that we can inspect the process later on cleanup
	o.agentPid.Store(int64(c.Process.Pid))
	if err := c.Wait(); err != nil {
		return fmt.Errorf("task agent command exited with an unexpected error: %w", err)
	}

	return nil
}

func (o *Orchestrator) loadToken(c *exec.Cmd) error {
	// Pass the task token to the task agent process through its stdin pipe
	w, err := c.StdinPipe()
	if err != nil {
		return fmt.Errorf("unexpected error on stdin pipe for task agent command: %w", err)
	}
	defer func() {
		_ = w.Close()
	}()

	_, err = w.Write([]byte(o.config.Token.Raw()))
	if err != nil {
		return fmt.Errorf("failed to write task token to stdin pipe for task agent command: %w", err)
	}

	return nil
}

func (o *Orchestrator) newCmd(ctx context.Context, cmd []string, env ...string) *exec.Cmd {
	//#nosec:G204 // this is intentionally setting up a command
	c := exec.CommandContext(ctx, cmd[0], cmd[1:]...)

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "CIRCLECI_GOAT") {
			// Prevent orchestrator configuration from being injected in the task environment
			continue
		}
		c.Env = append(c.Env, env)
	}
	if env != nil {
		c.Env = append(c.Env, env...)
	}

	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	o.maybeSwitchUser(ctx, c)

	return c
}

func (o *Orchestrator) maybeSwitchUser(ctx context.Context, c *exec.Cmd) {
	username := o.config.User
	if username == "" {
		return
	}

	usr, err := user.Lookup(username)
	if err == nil {
		uid, _ := strconv.Atoi(usr.Uid)
		gid, _ := strconv.Atoi(usr.Gid)
		c.SysProcAttr = &syscall.SysProcAttr{}
		c.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}
	} else {
		o11y.LogError(ctx, "failed to lookup user", err, o11y.Field("username", username))
	}
}

func (o *Orchestrator) cleanup(_ context.Context) error {
	defer func() {
		// Cancelling the context terminates the task agent and custom entrypoint commands
		o.cancelMu.RLock()
		defer o.cancelMu.RUnlock()
		o.cancelTask()
	}()

	pid := o.agentPid.Load()
	if pid > 0 {
		if p, err := os.FindProcess(int(pid)); err == nil {
			if err := p.Signal(os.Signal(syscall.Signal(0))); err == nil {
				return errors.New("task agent process is still running and may interrupt the task")
			} else if !errors.Is(err, os.ErrProcessDone) {
				return fmt.Errorf("unexpected error while signaling task agent process; %w", err)
			}
		}
	}

	return nil
}

func (o *Orchestrator) HealthChecks() (_ string, ready, live func(ctx context.Context) error) {
	return "orchestrator",
		func(ctx context.Context) error {
			if !o.ready.Load() {
				return fmt.Errorf("not ready")
			}
			return nil
		}, nil
}

type retryableError struct {
	error
}

func retryableErrorf(format string, a ...any) retryableError {
	return retryableError{fmt.Errorf(format, a...)}
}
