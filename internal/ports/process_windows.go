//go:build windows

package ports

import (
	"fmt"
	"os"
)

func supportsGracefulTermination() bool {
	return false
}

func gracefulSignalName() string {
	return "KILL"
}

func forceSignalName() string {
	return "KILL"
}

func signalProcess(pid int, force bool) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("locating process %d: %w", pid, err)
	}
	if err := proc.Kill(); err != nil {
		return fmt.Errorf("killing process %d: %w", pid, err)
	}
	return nil
}

func processAlive(pid int) (bool, error) {
	return false, nil
}
