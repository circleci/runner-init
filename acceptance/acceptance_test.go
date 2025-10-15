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
	orchestratorTestBinary         = os.Getenv("ORCHESTRATOR_TEST_BINARY")
	orchestratorTestBinaryRunTask  = ""
	orchestratorTestBinaryOverride = ""

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

	if orchestratorTestBinaryRunTask, err = createRunnerScript("run-task"); err != nil {
		return 0, err
	}
	fmt.Printf("Using 'orchestrator run-task' script: %q\n", orchestratorTestBinaryRunTask)

	if orchestratorTestBinaryOverride, err = createRunnerScript("override"); err != nil {
		return 0, err
	}
	fmt.Printf("Using 'orchestrator override' script: %q\n", orchestratorTestBinaryOverride)

	return m.Run(), nil
}

// A little hack to get around limitations of the test runner on positional arguments
func createRunnerScript(cmd string) (string, error) {
	var script string
	var scriptPath string

	if runtime.GOOS == "windows" {
		script = "@echo off\n" + orchestratorTestBinary + " " + cmd
		scriptPath = filepath.Join(binariesPath, fmt.Sprintf("orchestrator-%s.bat", cmd))
	} else {
		script = "#!/bin/bash\nexec " + orchestratorTestBinary + " " + cmd
		scriptPath = filepath.Join(binariesPath, fmt.Sprintf("orchestrator-%s.sh", cmd))
	}

	if err := os.WriteFile(scriptPath, []byte(script), 0750); err != nil { //nolint:gosec
		return "", err
	}

	return scriptPath, nil
}
