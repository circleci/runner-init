package step

import (
	"context"
	"io"
	"time"

	"github.com/circleci/ex/o11y"

	"github.com/circleci/runner-init/clients/runner"
)

type Streamer struct {
	runnerAPI *runner.Client

	step       *step
	out        io.WriteCloser
	startEvent runner.StepStart
	endEvent   runner.StepEnd
	ended      chan bool
	cancelCtx  func()
}

func NewStreamer(ctx context.Context, api *runner.Client, index int32) *Streamer {
	ctx, cancel := contextWithNewCancel(ctx)

	s := newStep(ctx, api, runner.Step{
		Allocation:     "unused",
		StepID:         index,
		Timestamp:      nowMillis(),
		SequenceNumber: int64(0),
	})

	e := runner.StepStart{
		Step: s.step,
	}
	e.Attributes.Type = "service_container"
	e.Attributes.Name = "Prerun"
	e.Attributes.Background = false
	e.Attributes.Insignificant = true

	se := runner.StepEnd{
		Step: s.next(),
	}
	se.Attributes.Result = "success" // TODO: we should probably set this based on the exit code of the container

	streamer := &Streamer{
		runnerAPI:  api,
		step:       s,
		startEvent: e,
		endEvent:   se,
		cancelCtx:  cancel,
	}

	streamer.start()

	return streamer
}

// contextWithNewCancel copies the O11y provider to a new context that can be separately cancelled.
// This ensures we can send all step events, even on shutdown of the agent and the parent context was cancelled
func contextWithNewCancel(ctx context.Context) (context.Context, func()) {
	return context.WithCancel(o11y.WithProvider(context.Background(), o11y.FromContext(ctx)))
}

func (s *Streamer) start() {
	s.out = s.step.out()

	s.ended = make(chan bool, 1)

	err := s.runnerAPI.StepStart(s.step.ctx, &s.startEvent)
	if err != nil {
		o11y.LogError(s.step.ctx, "step start", err)
	}
}

func (s *Streamer) Stream(in io.ReadCloser) {
	err := s.step.stream(in)
	if err != nil {
		o11y.LogError(s.step.ctx, "step output", err)
	}
}

func (s *Streamer) Out() io.Writer {
	return s.out
}

func (s *Streamer) End() {
	s.ended <- true

	if err := s.out.Close(); err != nil {
		o11y.LogError(s.step.ctx, "step output close", err)
	}

	s.endEvent.Step.Timestamp = nowMillis()
	err := s.runnerAPI.StepEnd(s.step.ctx, &s.endEvent)
	if err != nil {
		o11y.LogError(s.step.ctx, "step end", err)
	}

	s.cancelCtx()
}

func nowMillis() int64 {
	return time.Now().UnixNano() / 1000000
}
