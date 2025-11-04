package util

import (
	"fmt"
	"os"
	"path/filepath"
)

func FindAlloyBinary() (string, error) {
	// Check environment variable first
	if envPath := os.Getenv("ALLOY_BINARY"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
	}

	// Get the alloy executable path
	extensionPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	extensionDir := filepath.Dir(extensionPath)

	buildAlloyPath := filepath.Join(filepath.Dir(extensionDir), "build", "alloy")
	if _, err := os.Stat(buildAlloyPath); err == nil {
		return buildAlloyPath, nil
	}

	return "", fmt.Errorf("alloy binary not found. Tried: %s, %s, %s. Set ALLOY_BINARY environment variable to specify the path", buildAlloyPath)
}
