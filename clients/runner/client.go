package runner

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"runtime"
	"time"

	"github.com/circleci/ex/config/secret"
	"github.com/circleci/ex/httpclient"
	"github.com/circleci/ex/httpclient/dnscache"
	"github.com/circleci/ex/httpclient/metrics"
	"github.com/circleci/ex/o11y"
)

var ErrExhaustedTaskRetries = o11y.NewWarning("exhausted all task retries")

type Client struct {
	client *httpclient.Client
}

type ClientConfig struct {
	BaseURL   string
	AuthToken secret.String
	Info      Info
	Tracer    *metrics.Metrics
}

type Info struct {
	AgentVersion string
	// Correlation should be a unique-ish string to correlate API requests and logs
	Correlation string
}

func (i Info) userAgent() string {
	return fmt.Sprintf("CircleCI-GOAT/%s (%s; %s; %s)",
		i.AgentVersion, runtime.GOOS, runtime.GOARCH, i.Correlation)
}

func NewClient(c ClientConfig) *Client {
	cfg := httpclient.Config{
		Name:       "orchestrator",
		BaseURL:    c.BaseURL,
		AuthToken:  string(c.AuthToken),
		AcceptType: httpclient.JSON,
		Timeout:    time.Minute * 1,
		UserAgent:  c.Info.userAgent(),
		DialContext: dnscache.DialContext(dnscache.New(dnscache.Config{
			TTL: 30 * time.Second,
		}), nil),
	}

	if c.Tracer != nil {
		cfg.Tracer = c.Tracer
	}

	return &Client{httpclient.New(cfg)}
}

type taskUnclaim struct {
	ID    string `json:"task_id" binding:"required"`
	Token string `json:"task_token" binding:"required"`
}

func (c *Client) UnclaimTask(ctx context.Context, id string, token secret.String) error {
	r := httpclient.NewRequest("POST", "/api/v3/runner/unclaim",
		httpclient.Body(&taskUnclaim{
			ID:    id,
			Token: token.Raw(),
		}))

	err := c.call(ctx, r)

	switch {
	case httpclient.HasStatusCode(err, http.StatusConflict):
		return ErrExhaustedTaskRetries
	case err != nil:
		return err
	default:
		return nil
	}
}

var regexMatchHTMLSpecialChars = regexp.MustCompile(`[<>&'"]`)

type taskEvent struct {
	Allocation     string `json:"allocation"`
	TimestampMilli int64  `json:"timestamp"`
	Message        []byte `json:"message"`
}

func (c *Client) FailTask(ctx context.Context, timestamp time.Time, allocation, message string) error {
	r := httpclient.NewRequest("POST", "/api/v2/task/event/fail",
		httpclient.Body(&taskEvent{
			Allocation:     allocation,
			TimestampMilli: timestamp.UnixMilli(),
			Message:        []byte(regexMatchHTMLSpecialChars.ReplaceAllString(message, "")),
		}))

	return c.call(ctx, r)
}

func (c *Client) call(ctx context.Context, r httpclient.Request) error {
	err := c.client.Call(ctx, r)
	if err != nil && !httpclient.IsNoContent(err) {
		return fmt.Errorf("error calling CircleCI runner API: %w", err)
	}
	return err
}
