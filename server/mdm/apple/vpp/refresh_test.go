package vpp

import (
	"os"
	"testing"
)

func TestRefreshVersions(t *testing.T) {
	if os.Getenv("NETWORK_TEST") == "" {
		t.Skip("set environment variable NETWORK_TEST=1 to run")
	}
}
