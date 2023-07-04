//go:build darwin
// +build darwin

package software_update

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.IntegerColumn("software_update_required"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	newSoftwareAvailable, err := isNewSoftwareAvailable(ctx)

	return []map[string]string{
		{"software_update_required": newSoftwareAvailable},
	}, err
}

func isNewSoftwareAvailable(ctx context.Context) (newSoftwareAvailable string, err error) {
	res, err := runCommand(ctx, "/usr/sbin/softwareupdate", "-l")
	newSoftwareAvailable = ""
	if err == nil {
		newSoftwareAvailable = "1"
		if strings.Contains(res, "No new software available") {
			newSoftwareAvailable = "0"
		}
	}
	return newSoftwareAvailable, err
}

func runCommand(ctx context.Context, name string, arg ...string) (res string, err error) {
	// This query may take more than the avg query
	// I doubled the typical time from my tests and ended up with 30 seconds timeout.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, arg...)

	// Must use CombinedOutput and not Output since on some Intel machines we got the err result separately.
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Debug().Err(err).Msg("failed while generating software_update table")
		return "", err
	}
	return string(out), nil
}
