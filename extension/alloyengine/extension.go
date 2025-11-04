package alloyengine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/grafana/alloy/extension/alloyengine/util"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.uber.org/zap"
)

var _ extension.Extension = (*alloyEngineExtension)(nil)

type state int

var (
	stateNotStarted   state = 0
	stateRunning      state = 1
	stateShuttingDown state = 2
	stateTerminated   state = 3
)

func (s state) String() string {
	switch s {
	case stateNotStarted:
		return "not_started"
	case stateRunning:
		return "running"
	case stateShuttingDown:
		return "shutting_down"
	case stateTerminated:
		return "terminated"
	}
	return fmt.Sprintf("unknown_state_%d", s)
}

// alloyEngineExtension implements the alloyengine extension.
type alloyEngineExtension struct {
	config    *Config
	settings  component.TelemetrySettings
	runExited chan struct{}
	alloyPath string

	stateMutex sync.Mutex
	state      state
	runCancel  context.CancelFunc
	runCmd     *exec.Cmd
}

// newAlloyEngineExtension creates a new alloyEngine extension instance.
func newAlloyEngineExtension(config *Config, settings component.TelemetrySettings) (*alloyEngineExtension, error) {
	alloyPath, err := util.FindAlloyBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to find alloy binary: %w", err)
	}

	return &alloyEngineExtension{
		config:    config,
		settings:  settings,
		state:     stateNotStarted,
		alloyPath: alloyPath,
	}, nil
}

// Start is called when the extension is started.
func (e *alloyEngineExtension) Start(ctx context.Context, host component.Host) error {
	e.stateMutex.Lock()
	defer e.stateMutex.Unlock()

	switch e.state {
	case stateNotStarted:
		break
	default:
		return fmt.Errorf("cannot start alloyengine extension in current state: %s", e.state.String())
	}

	// Build command arguments: "run" + config path + flags
	args := []string{"run", e.config.ConfigPath}
	args = append(args, e.config.flagsAsSlice()...)

	runCtx, runCancel := context.WithCancel(context.Background())
	e.runCancel = runCancel
	e.runExited = make(chan struct{})

	// Create the command
	cmd := exec.CommandContext(runCtx, e.alloyPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()
	e.runCmd = cmd

	go func() {
		defer close(e.runExited)
		err := cmd.Run()

		e.stateMutex.Lock()
		previousState := e.state
		e.state = stateTerminated
		e.stateMutex.Unlock()

		if err == nil {
			e.settings.Logger.Debug("run command exited successfully without error")
		} else if previousState == stateShuttingDown {
			e.settings.Logger.Warn("run command exited with an error during shutdown", zap.Error(err))
		} else {
			e.settings.Logger.Error("run command exited unexpectedly with an error", zap.Error(err))
		}
	}()

	e.state = stateRunning
	e.settings.Logger.Info("alloyengine extension started successfully")
	return nil
}

// Shutdown is called when the extension is being stopped.
func (e *alloyEngineExtension) Shutdown(ctx context.Context) error {
	e.stateMutex.Lock()
	switch e.state {
	case stateRunning:
		e.settings.Logger.Info("alloyengine extension shutting down")
		e.state = stateShuttingDown
		// guaranteed to be non-nil since we are in stateRunning
		e.runCancel()
		// unlock so that the run goroutine can transition to terminated
		e.stateMutex.Unlock()

		select {
		case <-e.runExited:
			e.settings.Logger.Info("alloyengine extension shut down successfully")
		case <-ctx.Done():
			e.settings.Logger.Warn("alloyengine extension shutdown interrupted by context", zap.Error(ctx.Err()))
		}
		return nil
	case stateNotStarted:
		e.settings.Logger.Info("alloyengine extension shutdown completed (not started)")
		e.stateMutex.Unlock()
		return nil
	default:
		e.settings.Logger.Warn("alloyengine extension shutdown in current state is a no-op", zap.String("state", e.state.String()))
		e.stateMutex.Unlock()
		return nil
	}
}

// Ready returns nil when the extension is ready to process data.
func (e *alloyEngineExtension) Ready() error {
	e.stateMutex.Lock()
	defer e.stateMutex.Unlock()

	switch e.state {
	case stateRunning:
		return nil
	default:
		return fmt.Errorf("alloyengine extension not ready in current state: %s", e.state.String())
	}
}

// NotReady returns an error when the extension is not ready to process data.
func (e *alloyEngineExtension) NotReady() error {
	e.stateMutex.Lock()
	defer e.stateMutex.Unlock()

	switch e.state {
	case stateRunning:
		return nil
	default:
		return fmt.Errorf("alloyengine extension not ready in current state: %s", e.state.String())
	}
}
