package init

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

// Run function performs the copying of specific files and symlink creation
func Run(srcDir, destDir string) error {
	// Copy the orchestrator binary
	orchestratorSrc := filepath.Join(srcDir, binOrchestrator)
	orchestratorDest := filepath.Join(destDir, binOrchestrator)
	if err := copyFile(orchestratorSrc, orchestratorDest); err != nil {
		return err
	}

	// Copy the task agent binary
	agentSrc := filepath.Join(srcDir, binCircleciAgent)
	agentDest := filepath.Join(destDir, binCircleciAgent)
	if err := copyFile(agentSrc, agentDest); err != nil {
		return err
	}
	// Create symbolic link from "circleci-agent" to "circleci"
	if err := os.Symlink(agentDest, filepath.Join(destDir, binCircleci)); err != nil {
		return err
	}

	return nil
}

func copyFile(srcPath, destPath string) (err error) {
	closeFile := func(f *os.File) {
		err = errors.Join(err, f.Close())
	}

	srcFile, err := os.Open(srcPath) //#nosec:G304 // this is trusted input
	if err != nil {
		return err
	}
	defer closeFile(srcFile)

	// Get the file info to preserve the permissions
	info, err := srcFile.Stat()
	if err != nil {
		return err
	}

	//#nosec:G304 // this is trusted output
	destFile, err := os.OpenFile(destPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer closeFile(destFile)

	if _, err = io.Copy(destFile, srcFile); err != nil {
		return err
	}

	return err
}
