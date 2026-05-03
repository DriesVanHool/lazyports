package ports

import (
	"encoding/csv"
	"strconv"
	"strings"
)

func parseLsofOutput(output string) []Entry {
	lines := strings.Split(output, "\n")
	if len(lines) <= 1 {
		return nil
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

	return entries
}

func parseSSOutput(output string) []Entry {
	var entries []Entry
	for _, line := range strings.Split(output, "\n") {
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
	return entries
}

func parseWindowsNetstatOutput(protocol string, output string, pidsToNames map[int]string) []Entry {
	var entries []Entry
	for _, line := range strings.Split(output, "\n") {
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
	return entries
}

func parseTasklistOutput(output string) (map[int]string, error) {
	r := csv.NewReader(strings.NewReader(output))
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
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
