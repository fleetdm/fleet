//go:build darwin
// +build darwin

package nvram_info

import (
	"github.com/osquery/osquery-go/plugin/table"
	"golang.org/x/net/context"
	"testing"
	"time"
)

func TestGenerate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var tbl table.QueryContext

	table, err := Generate(ctx, tbl)
	if err != nil {
		t.Fatalf(`Expected no error. got %s`, err)
	}
	if table[0]["amfi_enabled"] != "0" && table[0]["amfi_enabled"] != "1" {
		t.Fatalf(`amfiEnabled expected 0 or 1. got %s`, table[0]["amfi_enabled"])
	}
}

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
