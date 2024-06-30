package task

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync/atomic"
	"syscall"

	"github.com/circleci/ex/o11y"
)

type Orchestrator struct {
	config Config

	ready     atomic.Bool
	agentPid  atomic.Int64
	cancelCtx context.CancelFunc
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		config:    Config{},
		cancelCtx: func() {},
	}
}

func (o *Orchestrator) Start(ctx context.Context) error {
	ctx, o.cancelCtx = context.WithCancel(o11y.WithProvider(context.Background(), o11y.FromContext(ctx)))

	if err := o.setup(ctx); err != nil {
		return fmt.Errorf("failed setup for task: %w", err)
	}

	if err := o.run(ctx); err != nil {
		return fmt.Errorf("failed to run task: %w", err)
	}

	return nil
}

func (o *Orchestrator) setup(ctx context.Context) error {
	o.ready.Store(true)

	if err := o.config.ReadFromStdin(ctx); err != nil {
		return err
	}

	if len(o.config.Entrypoint) > 0 {
		//#nosec:G204 // this is intentionally running a command
		c := exec.CommandContext(ctx, o.config.Entrypoint[0], o.config.Entrypoint[1:]...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr

		go func() {
			if err := c.Run(); err != nil {
				o11y.LogError(ctx, "error while running custom entrypoint", err)
			}
		}()
	}

	return nil
}

func (o *Orchestrator) run(ctx context.Context) error {
	agent := o.config.Agent()
	//#nosec:G204 // this is intentionally running a command
	c := exec.CommandContext(ctx, agent.Cmd[0], agent.Cmd[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = append(c.Env, agent.Env...)

	if err := c.Start(); err != nil {
		return fmt.Errorf("failed to start task agent: %w", err)
	}

	o.agentPid.Store(int64(c.Process.Pid))

	return c.Wait()
}

func (o *Orchestrator) Cleanup(_ context.Context) error {
	defer o.cancelCtx()

	pid := o.agentPid.Load()
	if pid > 0 {
		if p, err := os.FindProcess(int(pid)); err == nil {
			if err := p.Signal(os.Signal(syscall.Signal(0))); !errors.Is(err, syscall.ESRCH) {
				return fmt.Errorf("task agent is still running on shutdown and may get interrupted: %w", err)
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
