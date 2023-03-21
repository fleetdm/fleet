//go:build darwin
// +build darwin

package software_update

import (
	"github.com/osquery/osquery-go/plugin/table"
	"golang.org/x/net/context"
	"testing"
	"time"
)

func TestGenerate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var tbl table.QueryContext

	table, err := Generate(ctx, tbl)
	if err != nil {
		t.Fatalf(`Expected no error. got %s`, err)
	}
	if table[0]["new_software_available"] != "0" && table[0]["new_software_available"] != "1" {
		t.Fatalf(`new_software_available expected 0 or 1. got %s`, table[0]["new_software_available"])
	}
}

func TestIsNewSoftwareAvailable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	newSoftwareAvailable, err := isNewSoftwareAvailable(ctx)
	if newSoftwareAvailable != "0" && newSoftwareAvailable != "1" {
		t.Fatalf(`newSoftwareAvailable expected 0 or 1. got %s`, newSoftwareAvailable)
	}
	if err != nil {
		t.Fatalf(`Expected no error. got %s`, err)
	}
}
