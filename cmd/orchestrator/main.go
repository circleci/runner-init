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

	ShutdownDelay time.Duration `env:"SHUTDOWN_DELAY" default:"0s" help:"Delay shutdown by this amount."`
}

type initCmd struct {
	Source      string `arg:"" env:"SOURCE" type:"path" default:"/" help:"Path where to copy the agent binaries from."`
	Destination string `arg:"" env:"DESTINATION" type:"path" default:"/opt/circleci/bin" help:"Path where to copy the agent binaries to."`
}

type runTaskCmd struct {
	Stdin           *os.File `env:"STDIN" default:"-" hidden:""`
	HealthCheckAddr string   `env:"HEALTH_CHECK_ADDR" default:"localhost:7623" help:"Address for the health check API to listen on."`
}

func main() {
	if err := run(cmd.Version, cmd.Date); err != nil {
		log.Fatal(err)
	}
}

func run(version, date string) (err error) {
	cli := cli{}
	kongCtx := kong.Parse(&cli,
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
			return runTask(ctx, orchestrator)
		})
	}

	return sys.Run(ctx, cli.ShutdownDelay)
}

func runSetup(ctx context.Context, cli cli, sys *system.System) (*task.Orchestrator, error) {
	c := cli.RunTask

	if err := cmd.UpdateDefaultTransport(ctx); err != nil {
		return nil, fmt.Errorf("failed to load rootcerts: %w", err)
	}

	os.Stdin = c.Stdin

	o := task.NewOrchestrator()
	sys.AddService(o.Cleanup)
	sys.AddHealthCheck(o)

	if _, err := healthcheck.Load(ctx, cli.RunTask.HealthCheckAddr, sys); err != nil {
		return nil, fmt.Errorf("failed to load health check API: %w", err)
	}

	return o, nil
}

func runTask(ctx context.Context, o *task.Orchestrator) error {
	if err := o.Setup(ctx); err != nil {
		return fmt.Errorf("failed setup for task: %w", err)
	}

	return o.Run(ctx)
}
