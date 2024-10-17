//go:build darwin
// +build darwin

package csrutil_info

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
		table.IntegerColumn("ssv_enabled"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	SSVEnabled, err := getSSVEnabled(ctx)
	return []map[string]string{
		{"ssv_enabled": SSVEnabled},
	}, err
}

func getSSVEnabled(ctx context.Context) (ssvEnabled string, err error) {
	res, err := runCommand(ctx, "/usr/bin/csrutil", "authenticated-root", "status")
	ssvEnabled = ""
	if err == nil {
		ssvEnabled = "0"
		if strings.Contains(res, "Authenticated Root status: enabled") {
			ssvEnabled = "1"
		}
	}
	return ssvEnabled, err
}

func runCommand(ctx context.Context, name string, arg ...string) (res string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, arg...)

	out, err := cmd.Output()
	if err != nil {
		log.Debug().Err(err).Msg("failed while generating csrutil_info table")
		return "", err
	}
	return string(out), nil
}
