package main

import (
	"context"
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
	Init initCmd `cmd:"" name:"init"`

	ShutdownDelay time.Duration `env:"SHUTDOWN_DELAY" default:"5s" help:"Delay shutdown by this amount."`
}

type initCmd struct {
	Source      string `arg:"" type:"path" default:"/" help:"Path where to copy the agent binaries from."`
	Destination string `arg:"" type:"path" default:"/opt/circleci/bin" help:"Path where to copy the agent binaries to."`
}

func main() {
	if err := run(cmd.Version, cmd.Date); err != nil {
		log.Fatal(err)
	}
}

func run(version, date string) (err error) {
	cli := cli{}
	kongCtx := kong.Parse(&cli)

	ctx, o11yCleanup, err := setup.O11y(version)
	if err != nil {
		return err
	}
	defer o11yCleanup(ctx)

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
			return initialize.Run(c.Source, c.Destination)
		})
	}

	return sys.Run(ctx, cli.ShutdownDelay)
}
