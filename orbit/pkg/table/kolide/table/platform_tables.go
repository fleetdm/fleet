//go:build !darwin && !windows && !linux
// +build !darwin,!windows,!linux

package table

import (
	"github.com/go-kit/kit/log"
	osquery "github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

// platformTables returns an empty set. It's here as a catchall for
// unimplemented platforms.
func platformTables(client *osquery.ExtensionManagerClient, logger log.Logger, currentOsquerydBinaryPath string) []*table.Plugin {
	return []*table.Plugin{}
}
