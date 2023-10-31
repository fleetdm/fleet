//go:build darwin
// +build darwin

// based on github.com/kolide/launcher/pkg/osquery/tables
package dev_table_tooling

var allowedCommands = map[string]allowedCommand{
	"echo": {
		binPaths: []string{"echo"},
		args:     []string{"hello"},
	},
	"cb_repcli": {
		binPaths: []string{"/Applications/VMware Carbon Black Cloud/repcli.bundle/Contents/MacOS/repcli"},
		args:     []string{"status"},
	},
}
