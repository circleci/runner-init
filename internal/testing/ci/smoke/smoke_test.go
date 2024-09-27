//go:build smoke

package smoke

import (
	"os"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/circleci/ex/config/secret"
)

type CLI struct {
	CircleHost    string        `name:"circle-host" env:"CIRCLE_HOST" help:"The URL of your CircleCI host. This setting takes precedence over individual driver configurations and applies universally to all test cases."`
	CircleToken   secret.String `name:"circle-api-token" env:"CIRCLE_API_TOKEN" help:"An API token to authenticate with the CircleCI API. This setting overrides individual driver configurations and is applied to all test cases."`
	TriggerSource string        `name:"trigger-source" env:"CIRCLE_BUILD_URL" default:"dev" help:"Where this pipeline was triggered from."`
	T             string        `name:"smoke.test" short:"t"`

	Tests struct {
		Branch   string `name:"branch" env:"BRANCH" default:"main" help:"Which branch to run the tests on."`
		Version  string `name:"version" env:"VERSION" default:"edge" help:"The runner agent version to use in the tests."`
		IsCanary bool   `name:"is-canary" env:"IS_CANARY" default:"false" help:"Whether this is a canary or not. Some things like the Docker image repositories may differ for canaries."`

		// Driver-specific parameters
		Machine    `prefix:"machine-" envprefix:"MACHINE_"`
		Kubernetes `prefix:"kubernetes-" envprefix:"KUBERNETES_"`
	} `envprefix:"SMOKE_TESTS_" embed:""`
}

type Machine struct {
	CircleHost  string        `name:"circle-host" env:"CIRCLE_HOST" default:"https://circleci.com" help:"URL to your CircleCI host for the machine tests."`
	CircleToken secret.String `name:"circle-api-token" env:"CIRCLE_API_TOKEN" required:"true" help:"An API token to authenticate with the CircleCI API for the machine tests."`

	Skip bool `env:"SKIP" help:"Skip tests for the machine driver."`
}

type Kubernetes struct {
	CircleHost  string        `name:"circle-host" env:"CIRCLE_HOST" default:"https://k9s.sphereci.com" help:"URL to your CircleCI host for the Kubernetes tests."`
	CircleToken secret.String `name:"circle-api-token" env:"CIRCLE_API_TOKEN" required:"true" help:"An API token to authenticate with the CircleCI API for the Kubernetes tests."`

	Skip            bool   `env:"SKIP" help:"Skip tests for the Kubernetes driver."`
	RunnerInitTag   string `env:"RUNNER_INIT_TAG" default:"" help:"The runner-init image tag to use in the smoke tests."`
	HelmChartBranch string `env:"HELM_CHART_BRANCH" default:"" help:"An optional branch name on the CircleCI-Public/container-runner-helm-chart repository. This can be used for testing a pre-release Helm chart version."`
}

var cli *CLI

func TestMain(m *testing.M) {
	cli = &CLI{}
	_ = kong.Parse(cli, kong.Resolvers(
		kong.ResolverFunc(func(context *kong.Context, parent *kong.Path, flag *kong.Flag) (interface{}, error) {
			if strings.HasSuffix(flag.Name, "circle-host") && cli.CircleHost != "" {
				return cli.CircleHost, nil
			}
			if strings.HasSuffix(flag.Name, "circle-api-token") && cli.CircleToken.Raw() != "" {
				return cli.CircleToken.Raw(), nil
			}
			return nil, nil
		}),
	))
	os.Exit(m.Run())
}

func TestSmoke(t *testing.T) {
	var tests = []struct {
		name string

		driver      string
		circleHost  string
		circleToken secret.String
		skip        bool
		cases       []TestCase
	}{
		{
			name:        "machine success",
			driver:      "machine",
			circleHost:  cli.Tests.Machine.CircleHost,
			circleToken: cli.Tests.Machine.CircleToken,
			skip:        cli.Tests.Machine.Skip,
			cases: []TestCase{
				{
					WorkflowName:       "machine",
					WantWorkflowStatus: "success",
					CheckJobs:          nil,
				},
			},
		},
		{
			name:        "kubernetes success",
			driver:      "kubernetes",
			circleHost:  cli.Tests.Kubernetes.CircleHost,
			circleToken: cli.Tests.Kubernetes.CircleToken,
			skip:        cli.Tests.Kubernetes.Skip,
			cases: []TestCase{
				{
					WorkflowName:       "kubernetes",
					WantWorkflowStatus: "success",
					CheckJobs:          nil,
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.skip {
				t.Skipf("Tests for driver %q are disabled this run", tt.driver)
			}

			st := Tester{
				AgentDriver:   tt.driver,
				CircleHost:    tt.circleHost,
				CircleToken:   tt.circleToken,
				TriggerSource: cli.TriggerSource,
				Branch:        cli.Tests.Branch,
				AgentVersion:  cli.Tests.Version,
				IsCanary:      cli.Tests.IsCanary,
				ExtraPipelineParameters: map[string]any{
					"kubernetes_helm_chart_branch": cli.Tests.Kubernetes.HelmChartBranch,
					"kubernetes_runner_init_tag":   cli.Tests.Kubernetes.RunnerInitTag,
				},
			}
			st.Setup(t)

			for _, c := range tt.cases {
				t.Run(c.WorkflowName, func(t *testing.T) {
					st.Execute(t, c)
				})
			}
		})
	}
}
