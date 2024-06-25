package setup

import (
	"context"

	"github.com/circleci/ex/config/o11y"
)

func O11y(version string) (context.Context, func(context.Context), error) {
	cfg := o11y.Config{
		HoneycombEnabled: false,
		// Set `HoneycombKey` to something to suppress the "WARN: Missing API Key." log on startup.
		// If `HoneycombEnabled` is false, this doesn't matter. Without context,
		// this log can be misleading and has been confused with the runner API token.
		HoneycombKey: "-",
		Format:       "text",
		Version:      version,
		Service:      "orchestrator",
	}

	return o11y.Setup(context.Background(), cfg)
}
