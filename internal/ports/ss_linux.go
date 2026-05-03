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

	return parseSSOutput(string(out)), nil
}
