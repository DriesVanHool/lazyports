package ports

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

type Entry struct {
	Port     int
	Process  string
	PID      int
	Protocol string
	State    string
	Details  string
}

var portPattern = regexp.MustCompile(`:(\d+)`)

func List(ctx context.Context) ([]Entry, error) {
	var (
		entries []Entry
		err     error
	)

	switch runtime.GOOS {
	case "windows":
		entries, err = listWindows(ctx)
	case "darwin":
		entries, err = listLsof(ctx)
	default:
		entries, err = listUnix(ctx)
	}
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Port != entries[j].Port {
			return entries[i].Port < entries[j].Port
		}
		if entries[i].Process != entries[j].Process {
			return entries[i].Process < entries[j].Process
		}
		if entries[i].PID != entries[j].PID {
			return entries[i].PID < entries[j].PID
		}
		return entries[i].Protocol < entries[j].Protocol
	})

	return dedupe(entries), nil
}

func KillByPort(ctx context.Context, port int) (int, error) {
	entries, err := List(ctx)
	if err != nil {
		return 0, err
	}

	pids := map[int]struct{}{}
	for _, entry := range entries {
		if entry.Port == port {
			pids[entry.PID] = struct{}{}
		}
	}

	if len(pids) == 0 {
		return 0, fmt.Errorf("no process found on port %d", port)
	}

	killed := 0
	for pid := range pids {
		if err := KillPID(pid); err != nil {
			return killed, err
		}
		killed++
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
		endpoint := fields[protocolIdx+1]
		port, ok := extractPort(endpoint)
		if !ok {
			continue
		}

		details := strings.Join(fields[protocolIdx+1:], " ")
		state := extractState(details)
		entries = append(entries, Entry{
			Port:     port,
			Process:  fields[0],
			PID:      pid,
			Protocol: protocol,
			State:    state,
			Details:  details,
		})
	}

	return entries, nil
}

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
		if len(fields) < 5 {
			continue
		}

		port, ok := extractPort(fields[4])
		if !ok {
			continue
		}

		processName, pid := extractSSProcess(line)
		if pid == 0 {
			continue
		}

		state := strings.ToUpper(fields[1])
		entries = append(entries, Entry{
			Port:     port,
			Process:  processName,
			PID:      pid,
			Protocol: strings.ToUpper(fields[0]),
			State:    state,
			Details:  line,
		})
	}

	return entries, nil
}

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

		var state string
		var pidField string
		if strings.EqualFold(protocol, "tcp") {
			if len(fields) < 5 {
				continue
			}
			state = strings.ToUpper(fields[3])
			pidField = fields[4]
		} else {
			state = "UDP"
			pidField = fields[3]
		}

		pid, err := strconv.Atoi(pidField)
		if err != nil {
			continue
		}

		port, ok := extractPort(fields[1])
		if !ok {
			continue
		}

		entries = append(entries, Entry{
			Port:     port,
			Process:  pidsToNames[pid],
			PID:      pid,
			Protocol: strings.ToUpper(protocol),
			State:    state,
			Details:  line,
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

func dedupe(entries []Entry) []Entry {
	seen := map[string]struct{}{}
	result := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		key := fmt.Sprintf("%d/%s/%d/%s/%s", entry.Port, entry.Process, entry.PID, entry.Protocol, entry.State)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if entry.Process == "" {
			entry.Process = "unknown"
		}
		if entry.State == "" {
			entry.State = "-"
		}
		result = append(result, entry)
	}

	return result
}

func extractPort(value string) (int, bool) {
	matches := portPattern.FindAllStringSubmatch(value, -1)
	if len(matches) == 0 {
		return 0, false
	}

	port, err := strconv.Atoi(matches[len(matches)-1][1])
	if err != nil {
		return 0, false
	}

	return port, true
}

func extractState(value string) string {
	start := strings.LastIndex(value, "(")
	end := strings.LastIndex(value, ")")
	if start >= 0 && end > start {
		return strings.TrimSpace(value[start+1 : end])
	}
	return "-"
}

func firstField(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return "-"
	}
	return fields[0]
}

func indexLsofProtocol(fields []string) int {
	for i, field := range fields {
		upper := strings.ToUpper(field)
		if upper == "TCP" || upper == "UDP" {
			return i
		}
	}
	return -1
}

func extractSSProcess(line string) (string, int) {
	pidIdx := strings.Index(line, "pid=")
	if pidIdx < 0 {
		return "", 0
	}

	pidStart := pidIdx + len("pid=")
	pidEnd := pidStart
	for pidEnd < len(line) && line[pidEnd] >= '0' && line[pidEnd] <= '9' {
		pidEnd++
	}

	pid, err := strconv.Atoi(line[pidStart:pidEnd])
	if err != nil {
		return "", 0
	}

	nameEnd := strings.LastIndex(line[:pidIdx], `"`)
	if nameEnd < 0 {
		return "unknown", pid
	}
	nameStart := strings.LastIndex(line[:nameEnd], `"`)
	if nameStart < 0 {
		return "unknown", pid
	}

	return line[nameStart+1 : nameEnd], pid
}
