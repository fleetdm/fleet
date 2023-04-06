//go:build darwin
// +build darwin

package nvram_info

import (
	"context"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
	"os/exec"
	"strings"
	"time"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.IntegerColumn("amfi_enabled"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	amfiEnabled, err := getAMFIEnabled(ctx)

	return []map[string]string{
		{"amfi_enabled": amfiEnabled},
	}, err
}

func getAMFIEnabled(ctx context.Context) (amfiEnabled string, err error) {
	res, err := runCommand(ctx, "/usr/sbin/nvram", "-p")
	amfiEnabled = ""
	if err == nil {
		amfiEnabled = "0"
		if !strings.Contains(res, "amfi_get_out_of_my_way=1") {
			amfiEnabled = "1"
		}
	}
	return amfiEnabled, err
}

func runCommand(ctx context.Context, name string, arg ...string) (res string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, arg...)

	out, err := cmd.Output()
	if err != nil {
		log.Debug().Err(err).Msg("failed while generating nvram table")
		return "", err
	}
	return string(out), nil
}
