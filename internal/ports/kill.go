package ports

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

func KillByPort(ctx context.Context, port int) (int, error) {
	entries, err := List(ctx, ListOptions{})
	if err != nil {
		return 0, err
	}

	pids := targetPIDsForPort(entries, port)
	if len(pids) == 0 {
		return 0, fmt.Errorf("no listening process found on port %d", port)
	}

	killed := 0
	var errs []error
	for _, pid := range pids {
		if err := KillPID(pid); err != nil {
			errs = append(errs, err)
			continue
		}
		killed++
	}

	if len(errs) > 0 {
		return killed, errors.Join(errs...)
	}

	return killed, nil
}

func KillPID(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("locating process %d: %w", pid, err)
	}

	if err := proc.Kill(); err != nil {
		return fmt.Errorf("killing process %d: %w", pid, err)
	}

	return nil
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
