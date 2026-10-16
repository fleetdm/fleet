//go:build !windows

package amcache

import (
	"context"
	"errors"
	"time"
)

// The tables are only registered on Windows; these stubs keep the portable parts of the
// package buildable and testable elsewhere.

func readHive(context.Context) (hiveResult, error) {
	return hiveResult{}, errors.ErrUnsupported
}

func statHive() (time.Time, int64, error) {
	return time.Time{}, 0, errors.ErrUnsupported
}
