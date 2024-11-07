package runner

import (
	"context"

	"github.com/circleci/ex/httpclient"
)

type Step struct {
	Allocation        string `json:"allocation"`
	StepID            int32  `json:"step_id"`
	Timestamp         int64  `json:"timestamp"`
	SequenceNumber    int64  `json:"sequence_number"`
	NewSequenceNumber int64  `json:"-"`
}

type StepEnd struct {
	Step       Step `json:"step"`
	Attributes struct {
		Result   string `json:"result"`
		ExitCode string `json:"exit_code"`
	} `json:"attributes"`
}

func (c *Client) StepEnd(ctx context.Context, e *StepEnd) error {
	return c.client.Call(ctx, httpclient.NewRequest("POST", "/api/v2/step/end",
		httpclient.Body(StepEnd{
			Step:       replaceSequenceNumber(e.Step),
			Attributes: e.Attributes,
		}),
	))
}

type StepStart struct {
	Step       Step `json:"step"`
	Attributes struct {
		Type          string `json:"type"`
		Name          string `json:"name"`
		Parallel      bool   `json:"parallel"`
		Phase         string `json:"phase"`
		Command       string `json:"command"`
		Background    bool   `json:"background"`
		Insignificant bool   `json:"insignificant"`
	} `json:"attributes"`
}

func (c *Client) StepStart(ctx context.Context, e *StepStart) error {
	return c.client.Call(ctx, httpclient.NewRequest("POST", "/api/v2/step/start",
		httpclient.Body(StepStart{
			Step:       replaceSequenceNumber(e.Step),
			Attributes: e.Attributes,
		}),
	))
}

type StepOutput struct {
	Step    Step   `json:"step"`
	Message []byte `json:"out"`
}

func (c *Client) StepOutput(ctx context.Context, e *StepOutput) error {
	return c.client.Call(ctx, httpclient.NewRequest("POST", "/api/v2/step/output",
		httpclient.Body(StepOutput{
			Step:    replaceSequenceNumber(e.Step),
			Message: e.Message,
		}),
	))
}

func replaceSequenceNumber(s Step) Step {
	return Step{
		Allocation:     "unused",
		StepID:         s.StepID,
		Timestamp:      s.Timestamp,
		SequenceNumber: s.NewSequenceNumber,
	}
}
