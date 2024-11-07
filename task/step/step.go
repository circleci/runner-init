package step

import (
	"context"
	"io"
	"sync/atomic"
	"time"

	"github.com/circleci/runner-init/clients/runner"
)

const (
	bufferTimeout = time.Millisecond * 500
	bufferSize    = 8192
)

type step struct {
	ctx         context.Context
	externalAPI *runner.Client
	step        runner.Step
	sequence    atomic.Int64
	newSequence atomic.Int64
}

func newStep(ctx context.Context, externalAPI *runner.Client, s runner.Step) *step {
	return &step{
		ctx:         ctx,
		externalAPI: externalAPI,
		step:        s,
	}
}

func (s *step) next() runner.Step {
	newStep := s.step
	newStep.SequenceNumber = s.sequence.Add(1)
	return newStep
}

func (s *step) stream(in io.ReadCloser) (err error) {
	out := s.out()
	defer func() {
		if closeErr := out.Close(); err == nil {
			err = closeErr
		}
	}()

	_, err = io.Copy(out, in)
	return err
}

func (s *step) out() io.WriteCloser {
	writer := &runnerWriter{
		step: s,
	}
	return NewBufferingWriter(writer, bufferTimeout, bufferSize, nil)
}

type runnerWriter struct {
	step *step
}

func (w *runnerWriter) Write(p []byte) (count int, err error) {
	// I assume we copy here in case data in the slice p changes
	// concurrently. TODO It would be good to avoid a copy so we should
	// confirm this actually needed.
	pc := make([]byte, len(p))
	copy(pc, p)

	ctx := w.step.ctx
	api := w.step.externalAPI
	step := w.step.next()
	step.NewSequenceNumber = w.step.newSequence.Add(1) - 1

	err = api.StepOutput(ctx, &runner.StepOutput{
		Step:    step,
		Message: pc,
	})
	if err != nil {
		return 0, err
	}

	return len(p), nil
}
