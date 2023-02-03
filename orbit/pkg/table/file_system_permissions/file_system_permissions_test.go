//go:build darwin
// +build darwin

package file_system_permissions

import (
	"golang.org/x/net/context"
	"testing"
	"time"
)

func TestGetAMFIEnabled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	amfiEnabled, err := getAMFIEnabled(ctx)
	if amfiEnabled != "0" && amfiEnabled != "1" {
		t.Fatalf(`amfiEnabled expected 0 or 1. got %s`, amfiEnabled)
	}
	if err != nil {
		t.Fatalf(`Expected no error. got %s`, err)
	}
}

func TestGetSSVEnabled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ssvEnabled, err := getSSVEnabled(ctx)
	if ssvEnabled != "0" && ssvEnabled != "1" {
		t.Fatalf(`ssvEnabled expected 0 or 1. got %s`, ssvEnabled)
	}
	if err != nil {
		t.Fatalf(`Expected no error. got %s`, err)
	}
}
