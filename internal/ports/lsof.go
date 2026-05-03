package ports

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func listLsof(ctx context.Context) ([]Entry, error) {
	if _, err := exec.LookPath("lsof"); err != nil {
		return nil, errors.New("lsof is required to list ports on this OS")
	}

	out, err := exec.CommandContext(ctx, "lsof", "-nP", "-i").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("running lsof: %v: %s", err, strings.TrimSpace(string(out)))
	}

	lines := strings.Split(string(out), "\n")
	if len(lines) <= 1 {
		return nil, nil
	}

	entries := make([]Entry, 0, len(lines)-1)
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		protocolIdx := indexLsofProtocol(fields)
		if protocolIdx < 0 || protocolIdx+1 >= len(fields) {
			continue
		}

		protocol := strings.ToUpper(fields[protocolIdx])
		nameField := fields[protocolIdx+1]
		localAddr, remoteAddr, localPort, remotePort, kind, ok := parseLsofName(nameField)
		if !ok {
			continue
		}

		details := strings.Join(fields[protocolIdx+1:], " ")
		entries = append(entries, Entry{
			Port:          localPort,
			Process:       fields[0],
			PID:           pid,
			Protocol:      protocol,
			State:         extractState(details),
			Details:       details,
			Kind:          kind,
			LocalAddress:  localAddr,
			RemoteAddress: remoteAddr,
			RemotePort:    remotePort,
		})
	}

	return entries, nil
}
