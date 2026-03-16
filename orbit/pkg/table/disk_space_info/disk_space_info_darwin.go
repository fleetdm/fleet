//go:build darwin
// +build darwin

// Package disk_space_info provides a fleetd table that reports available and
// total disk capacity on macOS using NSURLVolumeAvailableCapacityForImportantUsageKey,
// which matches the "Available" space shown in macOS Finder's "Get Info" dialog.
package disk_space_info

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.BigIntColumn("bytes_available"),
		table.BigIntColumn("bytes_total"),
	}
}

// Generate is called to return the results for the table at query time.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	bytesAvailable, bytesTotal, err := getDiskSpace(ctx)
	if err != nil {
		return nil, err
	}
	return []map[string]string{
		{
			"bytes_available": fmt.Sprintf("%d", bytesAvailable),
			"bytes_total":     fmt.Sprintf("%d", bytesTotal),
		},
	}, nil
}

func getDiskSpace(ctx context.Context) (bytesAvailable, bytesTotal int64, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Use JXA (JavaScript for Automation) to call macOS Foundation APIs.
	// NSURLVolumeAvailableCapacityForImportantUsageKey returns available
	// capacity including purgeable space, which matches what macOS reports in
	// Finder's "Get Info" dialog and System Settings → General → Storage.
	script := `
ObjC.import('Foundation');
var url = $.NSURL.fileURLWithPath('/');
var availRef = Ref();
url.getResourceValueForKeyError(availRef, $.NSURLVolumeAvailableCapacityForImportantUsageKey, null);
var totalRef = Ref();
url.getResourceValueForKeyError(totalRef, $.NSURLVolumeTotalCapacityKey, null);
JSON.stringify({available: availRef[0].js, total: totalRef[0].js})
`
	cmd := exec.CommandContext(ctx, "osascript", "-l", "JavaScript", "-e", script)
	out, err := cmd.Output()
	if err != nil {
		log.Debug().Err(err).Msg("failed to get disk space via osascript")
		return 0, 0, fmt.Errorf("failed to run osascript: %w", err)
	}

	var result struct {
		Available int64 `json:"available"`
		Total     int64 `json:"total"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &result); err != nil {
		return 0, 0, fmt.Errorf("failed to parse disk space result %q: %w", out, err)
	}

	return result.Available, result.Total, nil
}
