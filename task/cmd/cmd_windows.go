package cmd

import (
	"context"
	"os"
	"os/exec"
	"os/signal"

	"github.com/circleci/ex/o11y"
)

func forwardSignals(cmd *exec.Cmd) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	go func() {
		for range ch {
			_ = cmd.Process.Kill()
		}
	}()
}

func switchUser(ctx context.Context, _ *exec.Cmd, user string) {
	o11y.Log(ctx, "switching users is unsupported on windows", o11y.Field("username", user))
}

func additionalSetup(_ context.Context, _ *exec.Cmd) {}
