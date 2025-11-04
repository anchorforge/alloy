package alloycli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

func otelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "otel [flags] [args...]",
		Short: "Run OpenTelemetry Collector",
		Long: `The otel subcommand executes the OpenTelemetry Collector binary with the provided arguments.
All arguments are passed through to the collector executable.`,
		SilenceUsage:       true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			collectorPath, err := findCollectorBinary()
			if err != nil {
				return fmt.Errorf("failed to find collector binary: %w", err)
			}

			execCmd := exec.Command(collectorPath, args...)
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr
			execCmd.Stdin = os.Stdin
			execCmd.Env = os.Environ()

			if err := execCmd.Run(); err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					os.Exit(exitError.ExitCode())
				}
				return err
			}

			return nil
		},
	}

	return cmd
}

func findCollectorBinary() (string, error) {
	if envPath := os.Getenv("COLLECTOR_BINARY"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
	}

	alloyPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	alloyDir := filepath.Dir(alloyPath)
	alloyName := filepath.Base(alloyPath)

	var collectorName string
	if filepath.Ext(alloyName) == ".exe" {
		collectorName = "collector.exe"
	} else {
		collectorName = "collector"
	}

	sameDirPath := filepath.Join(alloyDir, collectorName)
	if _, err := os.Stat(sameDirPath); err == nil {
		return sameDirPath, nil
	}

	devPath := filepath.Join(alloyDir, "collector", collectorName)
	if _, err := os.Stat(devPath); err == nil {
		return devPath, nil
	}

	parentCollectorPath := filepath.Join(filepath.Dir(alloyDir), "collector", collectorName)
	if _, err := os.Stat(parentCollectorPath); err == nil {
		return parentCollectorPath, nil
	}

	return "", fmt.Errorf("collector binary not found. Tried: %s, %s, %s. Set COLLECTOR_BINARY environment variable to specify the path", sameDirPath, devPath, parentCollectorPath)
}
