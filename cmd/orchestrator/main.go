package main

import (
	"context"
	"fmt"
	"log" //nolint:depguard // a non-O11y log is allowed for a top-level fatal exit
	"time"

	"github.com/alecthomas/kong"
	"github.com/circleci/ex/o11y"
	"github.com/circleci/ex/system"

	"github.com/circleci/runner-init/cmd"
	"github.com/circleci/runner-init/cmd/setup"
	initialize "github.com/circleci/runner-init/init"
)

type cli struct {
	Version kong.VersionFlag `short:"v" help:"Print version information and quit."`

	Init initCmd `cmd:"" name:"init" default:"withargs"`

	ShutdownDelay time.Duration `env:"SHUTDOWN_DELAY" default:"0s" help:"Delay shutdown by this amount."`
}

type initCmd struct {
	Source      string `arg:"" env:"SOURCE" type:"path" default:"/" help:"Path where to copy the agent binaries from."`
	Destination string `arg:"" env:"DESTINATION" type:"path" default:"/opt/circleci/bin" help:"Path where to copy the agent binaries to."`
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
			err := initialize.Run(c.Source, c.Destination)
			cancel()
			return err
		})
	}

	return sys.Run(ctx, cli.ShutdownDelay)
}
