package ports

import (
	"context"
	"encoding/csv"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func listWindows(ctx context.Context) ([]Entry, error) {
	pidsToNames, err := windowsProcessNames(ctx)
	if err != nil {
		return nil, err
	}

	tcp, err := listWindowsProtocol(ctx, "tcp", pidsToNames)
	if err != nil {
		return nil, err
	}

	udp, err := listWindowsProtocol(ctx, "udp", pidsToNames)
	if err != nil {
		return nil, err
	}

	return append(tcp, udp...), nil
}

func listWindowsProtocol(ctx context.Context, protocol string, pidsToNames map[int]string) ([]Entry, error) {
	out, err := exec.CommandContext(ctx, "netstat", "-ano", "-p", protocol).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("running netstat for %s: %v: %s", protocol, err, strings.TrimSpace(string(out)))
	}

	var entries []Entry
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 || !strings.EqualFold(fields[0], protocol) {
			continue
		}

		localAddr, localPort, ok := parseEndpoint(fields[1])
		if !ok {
			continue
		}

		var (
			state      string
			pidField   string
			remoteAddr string
			remotePort int
			kind       Kind
		)

		if strings.EqualFold(protocol, "tcp") {
			if len(fields) < 5 {
				continue
			}
			state = strings.ToUpper(fields[3])
			pidField = fields[4]
			remoteAddr, remotePort, _ = parseEndpoint(fields[2])
			kind = inferKind(state, remoteAddr)
		} else {
			state = "UDP"
			pidField = fields[3]
			kind = KindListener
		}

		pid, err := strconv.Atoi(pidField)
		if err != nil {
			continue
		}

		entries = append(entries, Entry{
			Port:          localPort,
			Process:       pidsToNames[pid],
			PID:           pid,
			Protocol:      strings.ToUpper(protocol),
			State:         state,
			Details:       line,
			Kind:          kind,
			LocalAddress:  localAddr,
			RemoteAddress: remoteAddr,
			RemotePort:    remotePort,
		})
	}

	return entries, nil
}

func windowsProcessNames(ctx context.Context) (map[int]string, error) {
	out, err := exec.CommandContext(ctx, "tasklist", "/FO", "CSV", "/NH").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("running tasklist: %v: %s", err, strings.TrimSpace(string(out)))
	}

	r := csv.NewReader(strings.NewReader(string(out)))
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parsing tasklist output: %w", err)
	}

	names := map[int]string{}
	for _, record := range records {
		if len(record) < 2 {
			continue
		}

		pid, err := strconv.Atoi(record[1])
		if err != nil {
			continue
		}

		names[pid] = record[0]
	}

	return names, nil
}
