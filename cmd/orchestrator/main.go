package main

import (
	"context"
	"fmt"
	"log" //nolint:depguard // a non-O11y log is allowed for a top-level fatal exit
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/circleci/ex/httpserver/healthcheck"
	"github.com/circleci/ex/o11y"
	"github.com/circleci/ex/system"

	"github.com/circleci/runner-init/cmd"
	"github.com/circleci/runner-init/cmd/setup"
	initialize "github.com/circleci/runner-init/init"
	"github.com/circleci/runner-init/task"
)

type cli struct {
	Version kong.VersionFlag `short:"v" help:"Print version information and quit."`

	Init    initCmd    `cmd:"" name:"init" default:"withargs"`
	RunTask runTaskCmd `cmd:"" name:"run-task"`

	ShutdownDelay time.Duration `default:"0s" help:"Delay shutdown by this amount."`
}

type initCmd struct {
	Source      string `arg:"" env:"SOURCE" type:"path" default:"/" help:"Path where to copy the agent binaries from."`
	Destination string `arg:"" env:"DESTINATION" type:"path" default:"/opt/circleci/bin" help:"Path where to copy the agent binaries to."`
}

type runTaskCmd struct {
	TerminationGracePeriod time.Duration `default:"20s" help:"How long the agent will wait for the task to complete if interrupted."`
	HealthCheckAddr        string        `default:"localhost:7623" help:"Address for the health check API to listen on."`

	// Task environment configuration should be injected through a Kubernetes Secret
	Config task.Config `required:"" hidden:"-"`
}

func main() {
	if err := run(cmd.Version, cmd.Date); err != nil {
		log.Fatal(err)
	}
}

func run(version, date string) (err error) {
	cli := cli{}
	kongCtx := kong.Parse(&cli,
		kong.DefaultEnvars("CIRCLECI_GOAT"),
		kong.Vars{
			"version": fmt.Sprintf("%s version %s (built %s)", "runner-init", version, date),
		})

	ctx, o11yCleanup, err := setup.O11y(version)
	if err != nil {
		return err
	}
	defer o11yCleanup(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	o11y.Log(ctx, "starting orchestrator",
		o11y.Field("version", version),
		o11y.Field("date", date),
	)

	sys := system.New()
	defer sys.Cleanup(ctx)

	switch kongCtx.Command() {
	case "init":
		c := cli.Init
		sys.AddService(func(_ context.Context) error {
			defer cancel()
			return initialize.Run(c.Source, c.Destination)
		})

	case "run-task":
		orchestrator, err := runSetup(ctx, cli, sys)
		if err != nil {
			return err
		}

		sys.AddService(func(ctx context.Context) error {
			defer cancel()
			return orchestrator.Run(ctx)
		})
	}

	return sys.Run(ctx, cli.ShutdownDelay)
}

func runSetup(ctx context.Context, cli cli, sys *system.System) (*task.Orchestrator, error) {
	c := cli.RunTask

	_ = os.Unsetenv("CIRCLECI_GOAT_CONFIG")

	if err := cmd.UpdateDefaultTransport(ctx); err != nil {
		return nil, fmt.Errorf("failed to load rootcerts: %w", err)
	}

	o := task.NewOrchestrator(c.Config, c.TerminationGracePeriod)
	sys.AddHealthCheck(o)

	if _, err := healthcheck.Load(ctx, c.HealthCheckAddr, sys); err != nil {
		return nil, fmt.Errorf("failed to load health check API: %w", err)
	}

	return o, nil
}
