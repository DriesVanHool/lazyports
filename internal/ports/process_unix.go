//go:build !windows

package ports

import (
	"fmt"
	"syscall"
)

func supportsGracefulTermination() bool {
	return true
}

func gracefulSignalName() string {
	return "TERM"
}

func forceSignalName() string {
	return "KILL"
}

func signalProcess(pid int, force bool) error {
	sig := syscall.SIGTERM
	action := "terminating"
	if force {
		sig = syscall.SIGKILL
		action = "force killing"
	}
	if err := syscall.Kill(pid, sig); err != nil {
		return fmt.Errorf("%s process %d: %w", action, pid, err)
	}
	return nil
}

func processAlive(pid int) (bool, error) {
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true, nil
	}
	if err == syscall.ESRCH {
		return false, nil
	}
	return false, fmt.Errorf("checking process %d: %w", pid, err)
}
