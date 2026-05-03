package ports

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	defaultGracePeriod = 1500 * time.Millisecond
	defaultWaitInterval = 100 * time.Millisecond
)

type TerminateOptions struct {
	GracePeriod  time.Duration
	Force        bool
	WaitInterval time.Duration
	GracefulOnly bool
}

type TerminateResult struct {
	Signal string
	Forced bool
}

type NeedsForceError struct {
	PID         int
	GracePeriod time.Duration
}

func (e NeedsForceError) Error() string {
	return fmt.Sprintf("process %d did not exit after %s; force kill required", e.PID, e.GracePeriod)
}

func KillByPort(ctx context.Context, port int) (int, error) {
	terminated, _, err := TerminateByPort(ctx, port, TerminateOptions{})
	return terminated, err
}

func KillPID(pid int) error {
	_, err := TerminatePID(context.Background(), pid, TerminateOptions{Force: true})
	return err
}

func TerminateByPort(ctx context.Context, port int, opts TerminateOptions) (terminated int, forced int, err error) {
	entries, err := List(ctx, ListOptions{})
	if err != nil {
		return 0, 0, err
	}

	pids := targetPIDsForPort(entries, port)
	if len(pids) == 0 {
		return 0, 0, fmt.Errorf("no listening process found on port %d", port)
	}

	var errs []error
	for _, pid := range pids {
		result, err := terminateWithFallback(ctx, pid, opts)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		terminated++
		if result.Forced {
			forced++
		}
	}

	if len(errs) > 0 {
		return terminated, forced, errors.Join(errs...)
	}
	return terminated, forced, nil
}

func terminateWithFallback(ctx context.Context, pid int, opts TerminateOptions) (TerminateResult, error) {
	result, err := TerminatePID(ctx, pid, opts)
	if err == nil {
		return result, nil
	}

	var needsForce NeedsForceError
	if opts.Force || opts.GracefulOnly || !errors.As(err, &needsForce) {
		return TerminateResult{}, err
	}

	return TerminatePID(ctx, pid, TerminateOptions{Force: true})
}

func TerminatePID(ctx context.Context, pid int, opts TerminateOptions) (TerminateResult, error) {
	if opts.Force || !supportsGracefulTermination() {
		return sendSignal(pid, true)
	}

	gracePeriod := opts.GracePeriod
	if gracePeriod <= 0 {
		gracePeriod = defaultGracePeriod
	}
	waitInterval := opts.WaitInterval
	if waitInterval <= 0 {
		waitInterval = defaultWaitInterval
	}

	if _, err := sendSignal(pid, false); err != nil {
		return TerminateResult{}, err
	}

	exited, err := waitForExit(ctx, pid, gracePeriod, waitInterval)
	if err != nil {
		return TerminateResult{}, err
	}
	if !exited {
		return TerminateResult{}, NeedsForceError{PID: pid, GracePeriod: gracePeriod}
	}

	return TerminateResult{Signal: gracefulSignalName()}, nil
}

func sendSignal(pid int, force bool) (TerminateResult, error) {
	if err := signalProcess(pid, force); err != nil {
		return TerminateResult{}, err
	}
	return TerminateResult{Signal: signalName(force), Forced: force}, nil
}

func signalName(force bool) string {
	if force {
		return forceSignalName()
	}
	return gracefulSignalName()
}

func waitForExit(ctx context.Context, pid int, gracePeriod, waitInterval time.Duration) (bool, error) {
	deadline := time.NewTimer(gracePeriod)
	defer deadline.Stop()

	ticker := time.NewTicker(waitInterval)
	defer ticker.Stop()

	for {
		alive, err := processAlive(pid)
		if err != nil {
			return false, err
		}
		if !alive {
			return true, nil
		}

		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-deadline.C:
			alive, err := processAlive(pid)
			if err != nil {
				return false, err
			}
			return !alive, nil
		case <-ticker.C:
		}
	}
}

func targetPIDsForPort(entries []Entry, port int) []int {
	pids := map[int]struct{}{}
	for _, entry := range entries {
		if entry.Port != port || entry.Kind != KindListener || entry.PID <= 0 {
			continue
		}
		pids[entry.PID] = struct{}{}
	}

	result := make([]int, 0, len(pids))
	for pid := range pids {
		result = append(result, pid)
	}
	sort.Ints(result)
	return result
}

func SummarizeKillError(err error) string {
	if err == nil {
		return ""
	}
	parts := strings.Split(err.Error(), "\n")
	if len(parts) == 0 {
		return "kill failed"
	}
	return parts[0]
}
