//go:build darwin
// +build darwin

package csrutil_info

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
	if table[0]["ssv_enabled"] != "0" && table[0]["ssv_enabled"] != "1" {
		t.Fatalf(`ssvEnabled expected 0 or 1. got %s`, table[0]["ssvEnabled"])
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
