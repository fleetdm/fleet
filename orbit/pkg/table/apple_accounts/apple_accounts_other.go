//go:build unix && !darwin

package apple_accounts

import (
	"context"
	"syscall"
)

// The table is only registered on macOS. This file lets the package and its
// tests build on the Linux CI runners, where O_NOFOLLOW_ANY doesn't exist and
// the owner check on the opened file is what stops a swapped path.

const noFollowFlags = syscall.O_NOFOLLOW

func localUsers(context.Context) ([]localUser, error) { return nil, nil }
