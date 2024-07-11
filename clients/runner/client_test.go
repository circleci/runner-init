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
)

func TestClient_UnclaimTask(t *testing.T) {
	type unclaim struct {
		ID    string `json:"task_id"`
		Token string `json:"task_token"`
	}

	tests := []struct {
		name string

		taskID       string
		token        secret.String
		httpResponse func(w http.ResponseWriter)

		wantRequests []httprecorder.Request
		wantErr      string
	}{
		{
			name: "successfully",

			taskID: "5384e98c-f3f0-4228-b6cd-fa105b2c96f2",
			token:  "9968caca-002b-4d57-a66b-44948587f823",
			httpResponse: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusOK)
			},

			wantRequests: []httprecorder.Request{
				{
					Method: "POST",
					URL:    url.URL{Path: "/api/v3/runner/unclaim"},
					Header: http.Header{"Accept": {"application/json; charset=utf-8"}, "Accept-Encoding": {"gzip"},
						"Content-Type": {"application/json; charset=utf-8"},
					},
					Body: jsonMustMarshal(t, unclaim{
						ID:    "5384e98c-f3f0-4228-b6cd-fa105b2c96f2",
						Token: "9968caca-002b-4d57-a66b-44948587f823",
					}),
				},
			},
		},
		{
			name: "exhausted all retries",

			taskID: "f941f7b4-c3f3-4b42-b14d-2b201d60dd8d",
			token:  "fa72e0b5-52cd-49d0-a1fa-342c92729885",
			httpResponse: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusConflict)
			},

			wantRequests: []httprecorder.Request{
				{
					Method: "POST",
					URL:    url.URL{Path: "/api/v3/runner/unclaim"},
					Header: http.Header{"Accept": {"application/json; charset=utf-8"}, "Accept-Encoding": {"gzip"},
						"Content-Type": {"application/json; charset=utf-8"},
					},
					Body: jsonMustMarshal(t, unclaim{
						ID:    "f941f7b4-c3f3-4b42-b14d-2b201d60dd8d",
						Token: "fa72e0b5-52cd-49d0-a1fa-342c92729885",
					}),
				},
			},
			wantErr: ErrExhaustedTaskRetries.Error(),
		},
		{
			name: "some generic error",

			taskID: "93f549d5-aad2-4a7a-ba34-e85f750b50fd",
			token:  "54ad5c4e-331b-4010-9309-f8fc5e042130",
			httpResponse: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusNotFound)
			},

			wantRequests: []httprecorder.Request{
				{
					Method: "POST",
					URL:    url.URL{Path: "/api/v3/runner/unclaim"},
					Header: http.Header{"Accept": {"application/json; charset=utf-8"}, "Accept-Encoding": {"gzip"},
						"Content-Type": {"application/json; charset=utf-8"}},
					Body: jsonMustMarshal(t, unclaim{
						ID:    "93f549d5-aad2-4a7a-ba34-e85f750b50fd",
						Token: "54ad5c4e-331b-4010-9309-f8fc5e042130",
					}),
				},
			},
			wantErr: "the response from POST /api/v3/runner/unclaim was 404 (Not Found) (1 attempts)",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			recorder := httprecorder.New()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := recorder.Record(r); err != nil {
					panic(err)
				}
				w.Header().Add("Content-Type", "application/json")
				tt.httpResponse(w)
			}))
			defer server.Close()

			c := NewClient(ClientConfig{
				BaseURL:   server.URL,
				AuthToken: "",
				Info:      Info{},
			})
			ctx := testcontext.Background()
			err := c.UnclaimTask(ctx, tt.taskID, tt.token)

			if tt.wantErr == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}

			assert.Check(t, cmp.DeepEqual(recorder.AllRequests(), tt.wantRequests,
				ignoreHeaders(t, "Content-Length", "Traceparent", "Tracestate", "User-Agent", "X-Honeycomb-Trace")))
		})
	}
}

func TestClient_FailTask(t *testing.T) {
	tests := []struct {
		name string

		timestamp    time.Time
		message      string
		allocation   string
		httpResponse func(w http.ResponseWriter)

		wantRequests []httprecorder.Request
		wantErr      string
	}{
		{
			name:         "strip html special characters",
			timestamp:    time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
			allocation:   "alloc",
			message:      `<'&error': "something wrong happened here!!!">`,
			httpResponse: func(w http.ResponseWriter) { w.WriteHeader(http.StatusOK) },

			wantRequests: []httprecorder.Request{
				{
					Method: "POST",
					URL:    url.URL{Path: "/api/v2/task/event/fail"},
					Header: http.Header{
						"Accept":          {"application/json; charset=utf-8"},
						"Accept-Encoding": {"gzip"},
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
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			recorder := httprecorder.New()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := recorder.Record(r); err != nil {
					panic(err)
				}
				w.Header().Add("Content-Type", "application/json")
				tt.httpResponse(w)
			}))
			defer server.Close()

			c := NewClient(ClientConfig{
				BaseURL:   server.URL,
				AuthToken: "",
				Info:      Info{},
			})
			ctx := testcontext.Background()
			err := c.FailTask(ctx, tt.timestamp, tt.allocation, tt.message)

			if tt.wantErr == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}

			assert.Check(t, cmp.DeepEqual(recorder.AllRequests(), tt.wantRequests,
				ignoreHeaders(t, "Content-Length", "Traceparent", "Tracestate", "User-Agent", "X-Honeycomb-Trace"),
			))
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
