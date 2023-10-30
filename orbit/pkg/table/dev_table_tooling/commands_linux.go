//go:build linux
// +build linux

package dev_table_tooling

var allowedCommands = map[string]allowedCommand{
	"echo": {
		binPaths: []string{"echo"},
		args:     []string{"hello"},
	},
	"cb_repcli": {
		binPaths: []string{"/opt/carbonblack/psc/bin/repcli"},
		args:     []string{"status"},
	},
}
