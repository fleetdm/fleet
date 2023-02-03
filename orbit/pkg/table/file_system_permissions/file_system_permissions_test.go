//go:build darwin
// +build darwin

package file_system_permissions

import (
	"golang.org/x/net/context"
	"testing"
	"time"
)

func TestGetAMFIEnabled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	amfiEnabled, err := getAMFIEnabled(ctx)
	if amfiEnabled != "0" && amfiEnabled != "1" {
		t.Fatalf(`amfiEnabled expected some answer. got %s`, amfiEnabled)
	}
	if err != nil {
		t.Fatalf(`Expected no error. got %s`, err)
	}
}
