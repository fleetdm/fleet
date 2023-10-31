//go:build windows
// +build windows

// based on github.com/kolide/launcher/pkg/osquery/tables
package dev_table_tooling

import "path/filepath"

var allowedCommands = map[string]allowedCommand{
	"echo": {
		binPaths: []string{"echo"},
		args:     []string{"hello"},
	},
	"cb_repcli": {
		binPaths: []string{filepath.Join("Program Files", "Confer", "repcli")},
		args:     []string{"status"},
	},
}
