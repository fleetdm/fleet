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
		table.IntegerColumn("new_software_available"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	newSoftwareAvailable, err := isNewSoftwareAvailable(ctx)

	return []map[string]string{
		{"new_software_available": newSoftwareAvailable},
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
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, arg...)

	out, err := cmd.Output()
	if err != nil {
		log.Debug().Err(err).Msg("failed while generating nvram table")
		return "", err
	}
	return string(out), nil
}
