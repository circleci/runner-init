package init

import (
	"os"
	"path/filepath"
)

// Run function performs the hard linking of specific files and symlink creation on the ephemeral volume
func Run(srcDir, destDir string) error {
	// Hard link the orchestrator binary
	orchestratorSrc := filepath.Join(srcDir, "orchestrator")
	orchestratorDest := filepath.Join(destDir, "orchestrator")
	if err := os.Link(orchestratorSrc, orchestratorDest); err != nil {
		return err
	}

	// Hard link the task agent binary
	agentSrc := filepath.Join(srcDir, "circleci-agent")
	agentDest := filepath.Join(destDir, "circleci-agent")
	if err := os.Link(agentSrc, agentDest); err != nil {
		return err
	}
	// Create symbolic link from "circleci-agent" to "circleci"
	if err := os.Symlink(agentDest, filepath.Join(destDir, "circleci")); err != nil {
		return err
	}

	return nil
}
