package ports

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func listSS(ctx context.Context) ([]Entry, error) {
	if _, err := exec.LookPath("ss"); err != nil {
		return nil, errors.New("ss is required as a fallback on Linux")
	}

	out, err := exec.CommandContext(ctx, "ss", "-H", "-tunap").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("running ss: %v: %s", err, strings.TrimSpace(string(out)))
	}

	var entries []Entry
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		localAddr, localPort, ok := parseEndpoint(fields[4])
		if !ok {
			continue
		}
		remoteAddr, remotePort, _ := parseEndpoint(fields[5])

		processName, pid := extractSSProcess(line)
		if pid == 0 {
			continue
		}

		state := strings.ToUpper(fields[1])
		entries = append(entries, Entry{
			Port:          localPort,
			Process:       processName,
			PID:           pid,
			Protocol:      strings.ToUpper(fields[0]),
			State:         state,
			Details:       line,
			Kind:          inferKind(state, remoteAddr),
			LocalAddress:  localAddr,
			RemoteAddress: remoteAddr,
			RemotePort:    remotePort,
		})
	}

	return entries, nil
}
