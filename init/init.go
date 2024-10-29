package init

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pierrec/lz4"
)

// Run function performs the copying of specific files and symlink creation
func Run(srcDir, destDir string) error {
	// Copy the orchestrator binary
	orchestratorSrc := filepath.Join(srcDir, "orchestrator")
	orchestratorDest := filepath.Join(destDir, "orchestrator")
	if err := copyFile(orchestratorSrc, orchestratorDest); err != nil {
		return err
	}

	// Check and decompress the task agent binary if needed
	agentSrc := filepath.Join(srcDir, "circleci-agent")
	if strings.HasSuffix(agentSrc, ".lz4") {
		decompressedAgentSrc, err := decompressLZ4(agentSrc)
		if err != nil {
			return err
		}
		agentSrc = decompressedAgentSrc
	}
	// Copy the task agent binary
	agentDest := filepath.Join(destDir, "circleci-agent")
	if err := copyFile(agentSrc, agentDest); err != nil {
		return err
	}
	// Create symbolic link from "circleci-agent" to "circleci"
	if err := os.Symlink(agentDest, filepath.Join(destDir, "circleci")); err != nil {
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

func decompressLZ4(filePath string) (string, error) {
	inFile, err := os.Open(filePath) //#nosec:G304
	if err != nil {
		return "", fmt.Errorf("failed to open input file %s: %w", filePath, err)
	}
	defer inFile.Close()

	outPath := strings.TrimSuffix(filePath, ".lz4")
	outFile, err := os.Create(outPath) //#nosec:G304
	if err != nil {
		return "", fmt.Errorf("failed to create output file %s: %w", outPath, err)
	}
	defer outFile.Close()

	lz4Reader := lz4.NewReader(inFile)
	if _, err := io.Copy(outFile, lz4Reader); err != nil {
		return "", fmt.Errorf("failed to decompress %s: %w", filePath, err)
	}

	return outPath, nil
}
