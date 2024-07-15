package runner

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/circleci/ex/config/secret"
	"github.com/circleci/ex/testing/httprecorder"
	"github.com/circleci/ex/testing/testcontext"
	gocmp "github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/circleci/runner-init/internal/testing/fakerunnerapi"
)

func TestClient_UnclaimTask(t *testing.T) {
	type unclaim struct {
		ID    string `json:"task_id"`
		Token string `json:"task_token"`
	}

	var (
		goodTask = fakerunnerapi.Task{
			Token:        "testtoken",
			ID:           "good",
			UnclaimCount: 0,
		}
		exhaustedTask = fakerunnerapi.Task{
			Token:        "anothertesttoken",
			ID:           "exhausted",
			UnclaimCount: 3,
		}
	)

	tests := []struct {
		name string

		taskID string
		token  secret.String

		wantRequests []httprecorder.Request
		wantError    string
	}{
		{
			name: "success",

			taskID: goodTask.ID,
			token:  goodTask.Token,
			wantRequests: []httprecorder.Request{
				{
					Method: "POST",
					URL:    url.URL{Path: "/api/v3/runner/unclaim"},
					Header: http.Header{"Accept": {"application/json; charset=utf-8"}, "Accept-Encoding": {"gzip"},
						"Content-Type": {"application/json; charset=utf-8"},
					},
					Body: jsonMustMarshal(t, unclaim{
						ID:    goodTask.ID,
						Token: goodTask.Token.Raw(),
					}),
				},
			},
		},
		{
			name: "exhausted all retries",

			taskID: exhaustedTask.ID,
			token:  exhaustedTask.Token,

			wantRequests: []httprecorder.Request{
				{
					Method: "POST",
					URL:    url.URL{Path: "/api/v3/runner/unclaim"},
					Header: http.Header{"Accept": {"application/json; charset=utf-8"}, "Accept-Encoding": {"gzip"},
						"Content-Type": {"application/json; charset=utf-8"},
					},
					Body: jsonMustMarshal(t, unclaim{
						ID:    exhaustedTask.ID,
						Token: exhaustedTask.Token.Raw(),
					}),
				},
			},
			wantError: ErrExhaustedTaskRetries.Error(),
		},
		{
			name: "not found",

			taskID: "notfound",
			token:  "notfound",

			wantRequests: []httprecorder.Request{
				{
					Method: "POST",
					URL:    url.URL{Path: "/api/v3/runner/unclaim"},
					Header: http.Header{"Accept": {"application/json; charset=utf-8"}, "Accept-Encoding": {"gzip"},
						"Content-Type": {"application/json; charset=utf-8"}},
					Body: jsonMustMarshal(t, unclaim{
						ID:    "notfound",
						Token: "notfound",
					}),
				},
			},
			wantError: "404 (Not Found)",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx := testcontext.Background()
			runnerAPI := fakerunnerapi.New(ctx, []fakerunnerapi.Task{goodTask, exhaustedTask})
			server := httptest.NewServer(runnerAPI)
			defer server.Close()

			c := NewClient(ClientConfig{
				BaseURL:   server.URL,
				AuthToken: tt.token,
				Info:      Info{},
			})

			err := c.UnclaimTask(ctx, tt.taskID, tt.token)

			if tt.wantError == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantError)
			}

			assert.Check(t, cmp.DeepEqual(runnerAPI.AllRequests(), tt.wantRequests, ignoreHeaders(t,
				"Authorization", "Content-Length", "Traceparent", "Tracestate", "User-Agent", "X-Honeycomb-Trace")))
		})
	}
}

func TestClient_FailTask(t *testing.T) {
	var goodTask = fakerunnerapi.Task{
		Token:      secret.String("testtoken"),
		Allocation: "alloc",
	}

	tests := []struct {
		name string

		token      secret.String
		timestamp  time.Time
		message    string
		allocation string

		wantRequests []httprecorder.Request
		wantError    string
	}{
		{
			name:       "strip html special characters",
			token:      goodTask.Token,
			timestamp:  time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
			allocation: goodTask.Allocation,
			message:    `<'&error': "something wrong happened here!!!">`,

			wantRequests: []httprecorder.Request{
				{
					Method: "POST",
					URL:    url.URL{Path: "/api/v2/task/event/fail"},
					Header: http.Header{
						"Accept":          {"application/json; charset=utf-8"},
						"Accept-Encoding": {"gzip"},
						"Authorization":   {"Bearer testtoken"},
						"Content-Type":    {"application/json; charset=utf-8"},
					},
					Body: jsonMustMarshal(t, struct {
						Allocation string `json:"allocation"`
						Timestamp  int64  `json:"timestamp"` // milliseconds
						Message    []byte `json:"message"`
					}{
						Allocation: "alloc",
						Timestamp:  1257894000000,
						Message:    []byte("error: something wrong happened here!!!"),
					}),
				},
			},
		},
		{
			name:      "not found",
			token:     "badtoken",
			wantError: "404 (Not Found)",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx := testcontext.Background()
			runnerAPI := fakerunnerapi.New(ctx, []fakerunnerapi.Task{goodTask})
			server := httptest.NewServer(runnerAPI)
			defer server.Close()

			c := NewClient(ClientConfig{
				BaseURL:   server.URL,
				AuthToken: tt.token,
				Info:      Info{},
			})

			err := c.FailTask(ctx, tt.timestamp, tt.allocation, tt.message)

			if tt.wantError == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantError)
			}

			if tt.wantRequests != nil {
				assert.Check(t, cmp.DeepEqual(runnerAPI.AllRequests(), tt.wantRequests,
					ignoreHeaders(t, "Content-Length", "Traceparent", "Tracestate", "User-Agent", "X-Honeycomb-Trace"),
				))
			}
		})
	}
}

func jsonMustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()

	b, err := json.Marshal(v)
	assert.NilError(t, err)

	return append(b, []byte("\n")...)
}

func ignoreHeaders(t *testing.T, headers ...string) gocmp.Option {
	t.Helper()

	return cmpopts.IgnoreMapEntries(func(h string, _ []string) bool {
		for _, header := range headers {
			if header == h {
				return true
			}
		}
		return false
	})
}
