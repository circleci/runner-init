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
	_, span := o11y.StartSpan(ctx, "override-entrypoint")
	defer o11y.End(span, &err)

	args := os.Args
	if len(e.args) > 1 {
		args = append(e.args[1:], os.Args...)
	}

	//#nosec:G204 // this is intentionally setting up a command
	if err := syscall.Exec(e.args[0], args, os.Environ()); err != nil {
		return fmt.Errorf("error execing entrypoint overide: %w", err)
	}

	return nil
}
