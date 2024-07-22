package cmd

import (
	"context"
	"sync"
	"time"

	"github.com/circleci/ex/o11y"
	"github.com/hashicorp/go-reap"
)

type Reaper struct {
	reapTimeout time.Duration
	reapMu      sync.RWMutex
	done        chan struct{}
}

func NewReaper(reapTimeout time.Duration) Reaper {
	return Reaper{
		reapTimeout: reapTimeout,
		done:        make(chan struct{}),
	}
}

func (r *Reaper) Enable(ctx context.Context) {
	// Take the reap lock so the process reaper doesn't immediately steal the return value from Go exec
	r.reapMu.RLock()

	go r.reapChildProcesses(ctx)
}

func (r *Reaper) Start() {
	r.reapMu.RUnlock()
}

func (r *Reaper) Done() chan struct{} {
	return r.done
}

func (r *Reaper) reapChildProcesses(ctx context.Context) {
	defer close(r.done)

	if !reap.IsSupported() {
		o11y.Log(ctx, "child process reaping is unsupported - this may result in zombie processes")
		return
	}

	reaped := make(reap.PidCh)

	go reap.ReapChildren(reaped, nil, r.done, &r.reapMu)

	<-ctx.Done() // block until the task is completed

	timer := time.NewTimer(r.reapTimeout)
	defer timer.Stop()

	for {
		// Time out if we don't reap any processes within 2 seconds
		select {
		case <-timer.C:
			return
		case <-reaped:
			timer.Stop()
			timer.Reset(r.reapTimeout)
		}
	}
}
