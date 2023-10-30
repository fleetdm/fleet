//go:build windows
// +build windows

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
