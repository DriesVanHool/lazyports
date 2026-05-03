package ports

import (
	"context"
	"fmt"
	"os/exec"
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

	return parseWindowsNetstatOutput(protocol, string(out), pidsToNames), nil
}

func windowsProcessNames(ctx context.Context) (map[int]string, error) {
	out, err := exec.CommandContext(ctx, "tasklist", "/FO", "CSV", "/NH").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("running tasklist: %v: %s", err, strings.TrimSpace(string(out)))
	}

	names, err := parseTasklistOutput(string(out))
	if err != nil {
		return nil, fmt.Errorf("parsing tasklist output: %w", err)
	}
	return names, nil
}
