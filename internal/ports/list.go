package ports

import (
	"context"
	"fmt"
	"runtime"
	"sort"
)

func List(ctx context.Context, options ListOptions) ([]Entry, error) {
	entries, err := listAll(ctx)
	if err != nil {
		return nil, err
	}

	entries = normalizeEntries(entries)
	if !options.IncludeConnections {
		entries = filterListeners(entries)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Port != entries[j].Port {
			return entries[i].Port < entries[j].Port
		}
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind
		}
		if entries[i].Process != entries[j].Process {
			return entries[i].Process < entries[j].Process
		}
		if entries[i].PID != entries[j].PID {
			return entries[i].PID < entries[j].PID
		}
		return entries[i].Protocol < entries[j].Protocol
	})

	return entries, nil
}

func listAll(ctx context.Context) ([]Entry, error) {
	switch runtime.GOOS {
	case "windows":
		return listWindows(ctx)
	case "darwin":
		return listLsof(ctx)
	default:
		return listUnix(ctx)
	}
}

func listUnix(ctx context.Context) ([]Entry, error) {
	entries, err := listLsof(ctx)
	if err == nil {
		return entries, nil
	}

	if runtime.GOOS == "linux" {
		fallback, fallbackErr := listSS(ctx)
		if fallbackErr == nil {
			return fallback, nil
		}
		return nil, fmt.Errorf("lsof failed: %v; ss failed: %w", err, fallbackErr)
	}

	return nil, err
}

func filterListeners(entries []Entry) []Entry {
	listeners := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.Kind == KindListener {
			listeners = append(listeners, entry)
		}
	}
	return listeners
}
