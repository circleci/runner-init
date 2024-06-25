package task

import (
	"context"
	"fmt"
	"sync/atomic"
)

type Orchestrator struct {
	config Config

	ready atomic.Bool
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		config: Config{},
	}
}

func (o *Orchestrator) Setup(ctx context.Context) error {
	o.ready.Store(true)

	if err := o.config.ReadFromStdin(ctx); err != nil {
		return err
	}

	// TODO: Execute o.config.Entrypoint in the background

	return nil
}

func (o *Orchestrator) Run(_ context.Context) (err error) {
	cmd := o.config.TaskAgentCmd()

	println(cmd)

	// TODO: Execute the task agent command

	return nil
}

func (o *Orchestrator) Cleanup(_ context.Context) error {
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
