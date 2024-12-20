package smoke

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/circleci/ex/config/secret"
	"github.com/circleci/ex/httpclient"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/poll"
)

const projectSlug = "github/circleci/runner-smoke-tests"

type Tester struct {
	Branch          string
	CircleHost      string
	RunnerAPIURL    string
	CircleToken     secret.String
	RunnerNamespace string

	TriggerSource           string
	AgentDriver             string
	AgentVersion            string
	IsCanary                bool
	ExtraPipelineParameters map[string]any

	client       *httpclient.Client
	pipelineResp pipelineResponse
}

func (st *Tester) Setup(t *testing.T) {
	st.client = httpclient.New(httpclient.Config{
		Name:       "runner-smoke-tests",
		BaseURL:    st.CircleHost + "/api/v2",
		AuthHeader: "Circle-Token",
		AuthToken:  st.CircleToken.Raw(),
	})

	t.Logf("Triggering pipeline for project %q on branch %q on host %q", projectSlug, st.Branch, st.CircleHost)
	t.Logf("Agent driver %q", st.AgentDriver)
	t.Logf("Agent version %q", st.AgentVersion)
	t.Logf("Extra pipeline parameters: %v", st.ExtraPipelineParameters)
	t.Logf("Is this a canary? %t", st.IsCanary)

	pipelineResp, err := st.triggerPipeline()
	assert.NilError(t, err)
	st.pipelineResp = pipelineResp

	t.Logf("Pipeline number %d was created; checking workflows...", pipelineResp.Number)
	t.Logf("Workflows URL: %s/api/v2/pipeline/%s/workflow", st.CircleHost, st.pipelineResp.ID)
}

type TestCase struct {
	WorkflowName       string
	WantWorkflowStatus string
	CheckJobs          func(t *testing.T, jobs []Job)
}

func (st *Tester) Execute(t *testing.T, tt TestCase) {
	var workflow *workflow
	isFound := false
	i := 0
	poll.WaitOn(t, func(t poll.LogT) poll.Result {
		i++
		workflows, err := st.allWorkflows(st.pipelineResp.ID)
		if err != nil {
			return poll.Continue("Could not get workflows for pipeline number %d: %s", st.pipelineResp.Number, err)
		}

		workflow = findWorkflow(workflows, tt.WorkflowName)
		if workflow == nil {
			return poll.Continue("Could not find workflow %q", tt.WorkflowName)
		}

		if !isFound {
			t.Logf("Found workflow %q: %s", tt.WorkflowName, st.workflowURL(workflow))
			isFound = true
		}

		if workflow.isStillRunning() {
			if i%300 == 0 {
				t.Logf("Workflow %q is still running: %s", tt.WorkflowName, st.workflowURL(workflow))
			}
			return poll.Continue("Workflow %q is still running: %s", tt.WorkflowName, st.workflowURL(workflow))
		}

		if workflow.Status != tt.WantWorkflowStatus {
			return poll.Error(fmt.Errorf("workflow %q does not have status %q\n%#v",
				tt.WorkflowName, tt.WantWorkflowStatus, workflow))
		}
		return poll.Success()
	}, poll.WithTimeout(20*time.Minute), poll.WithDelay(time.Second))

	if workflow.isStillRunning() {
		// If the workflow is still running after the tests have timed out,
		// cancel the workflow so that we don't have wasteful concurrency consumption on tasks that will never get claimed.
		assert.NilError(t, st.cancel(workflow.ID))
	}

	jobs, err := st.allJobs(workflow.ID)
	assert.NilError(t, err)

	if tt.CheckJobs != nil {
		tt.CheckJobs(t, jobs.Items)
	}
}

func (st *Tester) workflowURL(wf *workflow) string {
	u, _ := url.Parse(st.CircleHost)
	r, _ := url.JoinPath(u.Scheme+"://app."+u.Host,
		"pipelines", projectSlug, strconv.Itoa(wf.PipelineNumber), "workflows", wf.ID)
	return r
}

func findWorkflow(workflows workflowsResponse, name string) *workflow {
	// Get the earliest matching workflow so that we can ignore any reran workflows of the same name
	var earliestWorkflow *workflow
	for i := range workflows.Items {
		workflow := workflows.Items[i]
		if workflow.Name == name {
			if earliestWorkflow == nil || workflow.CreatedAt.Before(earliestWorkflow.CreatedAt) {
				earliestWorkflow = &workflow
			}
		}
	}
	return earliestWorkflow
}

type pipelineRequest struct {
	Branch     string         `json:"branch"`
	Parameters map[string]any `json:"parameters"`
}

type pipelineResponse struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	Number int    `json:"number"`
}

func (st *Tester) triggerPipeline() (resp pipelineResponse, err error) {
	parameters := st.ExtraPipelineParameters
	parameters["driver"] = st.AgentDriver
	parameters["trigger_source"] = st.TriggerSource
	parameters["version"] = st.AgentVersion

	u, err := url.Parse(st.CircleHost)
	if err != nil {
		return resp, err
	}
	if u.Host != "circleci.com" {
		parameters["circleci_domain"] = u.Host
		parameters["runner_api_url"] = st.CircleHost
		parameters["namespace"] = st.RunnerNamespace
	}

	req := pipelineRequest{
		Branch:     st.Branch,
		Parameters: parameters,
	}
	if st.IsCanary {
		req.Parameters["is_canary"] = true
	}
	route := fmt.Sprintf("/project/%s/pipeline", projectSlug)
	err = st.client.Call(context.Background(),
		httpclient.NewRequest("POST", route, httpclient.Body(req), httpclient.JSONDecoder(&resp)))
	if err != nil {
		return resp, fmt.Errorf("error triggering pipeline: %w", err)
	}
	return resp, nil
}

func (st *Tester) cancel(workflowID string) error {
	route := fmt.Sprintf("/workflow/%s/cancel", workflowID)
	err := st.client.Call(context.Background(), httpclient.NewRequest("POST", route))
	if err != nil {
		return fmt.Errorf("error cancelling workflow: %w", err)
	}
	return nil
}

type workflow struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Status         string    `json:"status"`
	PipelineID     string    `json:"pipeline_id"`
	PipelineNumber int       `json:"pipeline_number"`
	CreatedAt      time.Time `json:"created_at"`
	ProjectSlug    string    `json:"project_slug"`
}

var workflowStatuses = map[string]bool{
	"error":        true,
	"failed":       true,
	"success":      true,
	"canceled":     true,
	"unauthorized": true,
}

func (w *workflow) isStillRunning() bool {
	_, isNotRunning := workflowStatuses[w.Status]
	return !isNotRunning
}

type workflowsResponse struct {
	Items         []workflow `json:"items"`
	NextPageToken string     `json:"next_page_token"`
}

func (st *Tester) allWorkflows(pipelineID string) (all workflowsResponse, err error) {
	nextPageToken := ""
	for {
		resp, err := st.getWorkflows(pipelineID, nextPageToken)
		if err != nil {
			return all, err
		}
		all.Items = append(all.Items, resp.Items...)
		if resp.NextPageToken == "" {
			break
		}
		nextPageToken = resp.NextPageToken
	}
	return all, nil
}

func (st *Tester) getWorkflows(pipelineID string, nextPageToken string) (resp workflowsResponse, err error) {
	route := fmt.Sprintf("/pipeline/%s/workflow", pipelineID)
	if nextPageToken != "" {
		route += fmt.Sprintf("?page-token=%s", nextPageToken)
	}

	err = st.client.Call(context.Background(), httpclient.NewRequest("GET", route, httpclient.JSONDecoder(&resp)))
	if err != nil {
		return resp, fmt.Errorf("error getting workflows: %w", err)
	}
	return resp, nil
}

type Job struct {
	CanceledBy   string   `json:"canceled_by"`
	Dependencies []string `json:"dependencies"`
	JobNumber    int      `json:"job_number"`
	ID           string   `json:"id"`
	StartedAt    string   `json:"started_at"`
	Name         string   `json:"name"`
	ApprovedBy   string   `json:"approved_by"`
	ProjectSlug  string   `json:"project_slug"`
	Status       string   `json:"status"`
	Type         string   `json:"type"`
	StoppedAt    string   `json:"stopped_at"`
}

type jobsResponse struct {
	Items         []Job  `json:"items"`
	NextPageToken string `json:"next_page_token"`
}

func (st *Tester) allJobs(workflowID string) (all jobsResponse, err error) {
	nextPageToken := ""
	for {
		resp, err := st.getJobs(workflowID, nextPageToken)
		if err != nil {
			return all, err
		}
		all.Items = append(all.Items, resp.Items...)
		if resp.NextPageToken == "" {
			break
		}
		nextPageToken = resp.NextPageToken
	}
	return all, nil
}

func (st *Tester) getJobs(workflowID, nextPageToken string) (resp jobsResponse, err error) {
	route := fmt.Sprintf("/workflow/%s/job", workflowID)
	if nextPageToken != "" {
		route += fmt.Sprintf("?page-token=%s", nextPageToken)
	}
	err = st.client.Call(context.Background(), httpclient.NewRequest("GET", route, httpclient.JSONDecoder(&resp)))
	if err != nil {
		return resp, fmt.Errorf("error getting jobs: %w", err)
	}
	return resp, nil
}
