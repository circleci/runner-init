package task

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/circleci/ex/o11y"
	"github.com/fsnotify/fsnotify"

	"github.com/circleci/runner-init/clients/runner"
	"github.com/circleci/runner-init/task/cmd"
	"github.com/circleci/runner-init/task/taskerrors"
)

type Orchestrator struct {
	config       Config
	runnerClient *runner.Client
	gracePeriod  time.Duration

	ready      atomic.Bool
	entrypoint cmd.Command
	taskAgent  cmd.Command
	reaper     cmd.Reaper
	cancelTask context.CancelFunc
}

var (
	// These can be overridden in tests
	reapTimeout             = 2 * time.Second
	waitForReadinessTimeout = 10 * time.Minute
)

func NewOrchestrator(config Config, runnerClient *runner.Client, gracePeriod time.Duration) *Orchestrator {
	if runnerClient == nil {
		panic("runner API client is unset")
	}

	return &Orchestrator{
		config:       config,
		runnerClient: runnerClient,
		gracePeriod:  gracePeriod,
		reaper:       cmd.NewReaper(reapTimeout),
	}
}

func (o *Orchestrator) Run(parentCtx context.Context) (err error) {
	parentCtx, span := o11y.StartSpan(parentCtx, "run-task")

	ctx := o.taskContext(parentCtx)
	o.reaper.Enable(ctx)

	defer func() {
		err = o.shutdown(ctx, err)
		o11y.End(span, &err)
	}()

	if len(o.config.Cmd) > 0 {
		// If a custom entrypoint is specified, execute it in the background
		if err := o.executeEntrypoint(ctx); err != nil {
			return err
		}
	}

	// Signal the orchestrator is ready and will start the task agent process
	o.ready.Store(true)

	if len(o.config.ReadinessFilePath) > 0 {
		// Wait for readiness from the other containers before starting the task agent process
		if err := o.waitForReadiness(ctx); err != nil {
			return err
		}
	}

	errCh := make(chan error, 1)
	go func() {
		// Start process reaping once the task agent process has completed
		defer o.reaper.Start()

		if err := o.executeAgent(ctx); err != nil {
			errCh <- fmt.Errorf("error while executing task agent: %w", err)
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-parentCtx.Done():
		// If the parent context is cancelled, wait for the termination grace period before shutting down.
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
	// Copy the O11y provider to a new context that can be separately cancelled.
	// This ensures we can drain the task on shutdown of the agent even if the parent context was cancelled,
	// but still make sure any task resources are released.
	ctx, o.cancelTask = context.WithCancel(o11y.WithProvider(context.Background(), o11y.FromContext(ctx)))
	return ctx
}

func (o *Orchestrator) waitForReadiness(ctx context.Context) (err error) {
	ctx, span := o11y.StartSpan(ctx, "orchestrator: wait-for-readiness")
	defer func() {
		span.AddField("ready", err == nil)
		o11y.End(span, &err)
	}()
	ctx, cancel := context.WithTimeout(ctx, waitForReadinessTimeout) // so we don't wait indefinitely if there's a problem
	defer cancel()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer func() {
		_ = watcher.Close()
	}()

	readinessFilePath := o.config.ReadinessFilePath
	if err := watcher.Add(filepath.Dir(readinessFilePath)); err != nil {
		return err
	}

	// Check if the readiness file already exists
	if _, err := os.Stat(readinessFilePath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	// Otherwise, wait for it to be created
	for {
		select {
		case event, ok := <-watcher.Events:
			if ok && event.Has(fsnotify.Create) && event.Name == readinessFilePath {
				return nil
			}
		case err, ok := <-watcher.Errors:
			if ok && err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (o *Orchestrator) executeEntrypoint(ctx context.Context) error {
	c := o.config.Cmd
	o.entrypoint = cmd.New(ctx, c, true, "")

	if err := o.entrypoint.Start(); err != nil {
		return fmt.Errorf("error starting custom entrypoint %s: %w", c, err)
	}
	return nil
}

func (o *Orchestrator) executeAgent(ctx context.Context) error {
	cfg := o.config
	agent := cfg.Agent()

	o.taskAgent = cmd.New(ctx, agent.Cmd, false, cfg.User, agent.Env...)

	if err := o.taskAgent.StartWithStdin([]byte(cfg.Token.Raw())); err != nil {
		return taskerrors.RetryableErrorf("failed to start task agent command: %w", err)
	}

	if err := o.taskAgent.Wait(); err != nil {
		return fmt.Errorf("task agent command exited with an unexpected error: %w", err)
	}

	return nil
}

func (o *Orchestrator) shutdown(ctx context.Context, runErr error) (err error) {
	isRunning, err := o.taskAgent.IsRunning()
	if isRunning {
		err = fmt.Errorf("task agent process is still running, which could interrupt the task. " +
			"Possible reasons include the Pod being evicted or deleted")
	}
	if err != nil {
		err = fmt.Errorf("error on shutdown: %w", err)
	}

	err = errors.Join(err, runErr)
	if err != nil {
		err = o.handleErrors(ctx, err)
	}

	o.cancelTask()

	<-o.reaper.Done()

	return err
}

func (o *Orchestrator) handleErrors(ctx context.Context, err error) error {
	ctx = o11y.WithProvider(context.Background(), o11y.FromContext(ctx))
	c := o.config

	if err != nil {
		err = fmt.Errorf("%w: Check container logs for more details", err)
	}

	var unclaimErr error
	if errors.As(err, &taskerrors.RetryableError{}) || c.EnableUnsafeRetries {
		unclaimErr = o.runnerClient.UnclaimTask(ctx, c.TaskID, c.Token)
		if unclaimErr == nil {
			o11y.LogError(ctx, "retrying task after encountering a retryable error", err)
			return nil
		}
	}

	if unclaimErr != nil {
		unclaimErr = fmt.Errorf("failed to retry task: %w", unclaimErr)
	}

	failErr := o.runnerClient.FailTask(ctx, time.Now(), c.Allocation, err.Error())
	if failErr != nil {
		failErr = fmt.Errorf("failed to send fail event for task: %w", failErr)
		return errors.Join(failErr, unclaimErr, err)
	}

	return taskerrors.NewHandledError(errors.Join(unclaimErr, err))
}

func (o *Orchestrator) HealthChecks() (_ string, ready, live func(ctx context.Context) error) {
	return "orchestrator",
		func(_ context.Context) error {
			if !o.ready.Load() {
				return fmt.Errorf("not ready")
			}
			return nil
		}, nil
}
