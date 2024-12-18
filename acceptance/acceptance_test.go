package acceptance

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/circleci/ex/testing/compiler"
	"github.com/circleci/ex/testing/testcontext"
)

var (
	orchestratorTestBinary        = os.Getenv("ORCHESTRATOR_TEST_BINARY")
	orchestratorTestBinaryRunTask = ""

	binariesPath    = ""
	taskAgentBinary = ""
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

	if err := createRunTaskScript(); err != nil {
		return 0, err
	}
	fmt.Printf("Using 'orchestrator run-task' script: %q\n", orchestratorTestBinaryRunTask)

	return m.Run(), nil
}

// A little hack to get around limitations of the test runner on positional arguments
func createRunTaskScript() error {
	var script string
	var scriptPath string

	if runtime.GOOS == "windows" {
		script = "@echo off\n" + orchestratorTestBinary + " run-task"
		scriptPath = filepath.Join(binariesPath, "orchestratorRunTask.bat")
	} else {
		script = "#!/bin/bash\nexec " + orchestratorTestBinary + " run-task"
		scriptPath = filepath.Join(binariesPath, "orchestratorRunTask.sh")
	}

	if err := os.WriteFile(scriptPath, []byte(script), 0750); err != nil { //nolint:gosec
		return err
	}

	orchestratorTestBinaryRunTask = scriptPath

	return nil
}
