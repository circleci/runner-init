package acceptance

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/circleci/ex/testing/compiler"
	"github.com/circleci/ex/testing/testcontext"
)

var (
	orchestratorTestBinary = os.Getenv("ORCHESTRATOR_TEST_BINARY")
	binariesPath           = ""
	taskAgentBinary        = ""
)

func TestMain(m *testing.M) {
	status, err := runTests(m)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	os.Exit(status)
}

func runTests(m *testing.M) (int, error) {
	ctx := testcontext.Background()

	p := compiler.NewParallel(runtime.NumCPU())
	defer p.Cleanup()

	binariesPath = p.Dir()

	// Init binaries
	p.Add(compiler.Work{
		Result: &orchestratorTestBinary,
		Name:   "orchestrator",
		Target: "..",
		Source: "github.com/circleci/runner-init/cmd/orchestrator",
	})

	// Fakes
	p.Add(compiler.Work{
		Result: &taskAgentBinary,
		Name:   "task-agent",
		Target: "..",
		Source: "github.com/circleci/runner-init/internal/testing/faketaskagent",
	})

	err := p.Run(ctx)
	if err != nil {
		return 0, err
	}

	fmt.Printf("Using 'orchestrator' test binary: %q\n", orchestratorTestBinary)
	fmt.Printf("Using fake 'task-agent' test binary: %q\n", taskAgentBinary)

	return m.Run(), nil
}
