package entrypoint

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/circleci/ex/o11y"
)

type Entrypoint struct {
	args []string
}

func New(args []string) Entrypoint {
	return Entrypoint{args}
}

func (e Entrypoint) Run(ctx context.Context) (err error) {
	ctx, span := o11y.StartSpan(ctx, "override-entrypoint")
	defer o11y.End(span, &err)

	//#nosec:G204 // this is intentionally setting up a command
	if err := syscall.Exec(e.args[0], append(e.args[1:], os.Args...), os.Environ()); err != nil {
		return fmt.Errorf("error execing entrypoint overide: %w", err)
	}

	return nil
}
