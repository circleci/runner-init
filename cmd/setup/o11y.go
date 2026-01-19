package setup

import (
	"context"

	"github.com/circleci/ex/config/o11y"
)

func O11y(version string) (context.Context, func(context.Context), error) {
	cfg := o11y.OtelConfig{
		Version: version,
		Service: "orchestrator",
	}

	return o11y.Otel(context.Background(), cfg)
}
