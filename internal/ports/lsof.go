package ports

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
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

	return parseLsofOutput(string(out)), nil
}
