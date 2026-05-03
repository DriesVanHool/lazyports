//go:build !linux

package ports

import (
	"context"
	"errors"
)

func listSS(context.Context) ([]Entry, error) {
	return nil, errors.New("ss fallback is only available on Linux")
}
