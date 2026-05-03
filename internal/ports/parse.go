package ports

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var portPattern = regexp.MustCompile(`:(\d+)`)

func normalizeEntries(entries []Entry) []Entry {
	seen := map[string]struct{}{}
	result := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.Process == "" {
			entry.Process = "unknown"
		}
		if entry.State == "" {
			entry.State = "-"
		}
		if entry.Kind == "" {
			entry.Kind = inferKind(entry.State, entry.RemoteAddress)
		}
		key := fmt.Sprintf("%d/%s/%d/%s/%s/%s/%s", entry.Port, entry.Process, entry.PID, entry.Protocol, entry.State, entry.LocalAddress, entry.RemoteAddress)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, entry)
	}
	return result
}

func parseEndpoint(value string) (address string, port int, ok bool) {
	value = strings.TrimSpace(value)
	if value == "" || value == "*:*" {
		return "", 0, false
	}

	matches := portPattern.FindAllStringSubmatch(value, -1)
	if len(matches) == 0 {
		return "", 0, false
	}

	port, err := strconv.Atoi(matches[len(matches)-1][1])
	if err != nil {
		return "", 0, false
	}

	idx := strings.LastIndex(value, ":")
	if idx <= 0 {
		return value, port, true
	}

	return strings.Trim(value[:idx], "[]"), port, true
}

func parseLsofName(value string) (local, remote string, localPort, remotePort int, kind Kind, ok bool) {
	parts := strings.Split(value, "->")
	if len(parts) == 1 {
		local, localPort, ok = parseEndpoint(strings.TrimSpace(parts[0]))
		if !ok {
			return "", "", 0, 0, "", false
		}
		return local, "", localPort, 0, KindListener, true
	}
	if len(parts) != 2 {
		return "", "", 0, 0, "", false
	}

	local, localPort, ok = parseEndpoint(strings.TrimSpace(parts[0]))
	if !ok {
		return "", "", 0, 0, "", false
	}
	remote, remotePort, ok = parseEndpoint(strings.TrimSpace(parts[1]))
	if !ok {
		return "", "", 0, 0, "", false
	}

	return local, remote, localPort, remotePort, KindConnection, true
}

func extractState(value string) string {
	start := strings.LastIndex(value, "(")
	end := strings.LastIndex(value, ")")
	if start >= 0 && end > start {
		return strings.TrimSpace(value[start+1 : end])
	}
	return "-"
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

func inferKind(state string, remoteAddress string) Kind {
	upper := strings.ToUpper(state)
	if upper == "LISTEN" || upper == "LISTENING" || upper == "UNCONN" || upper == "UDP" || remoteAddress == "" {
		return KindListener
	}
	return KindConnection
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
