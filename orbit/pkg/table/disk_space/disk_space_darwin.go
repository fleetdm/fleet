//go:build darwin

// Package disk_space provides a fleetd table that reports available and
// total disk capacity on macOS using NSURLVolumeAvailableCapacityForImportantUsageKey,
// which matches the "Available" space shown in macOS Finder's "Get Info" dialog.
package disk_space

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

func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.BigIntColumn("bytes_available"),
		table.BigIntColumn("bytes_total"),
	}
}

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

	script := `
ObjC.import('Foundation');
var url = $.NSURL.fileURLWithPath('/');
var err = Ref();
var availRef = Ref();
if (!url.getResourceValueForKeyError(availRef, $.NSURLVolumeAvailableCapacityForImportantUsageKey, err)) {
    throw new Error('failed to get available capacity: ' + ObjC.unwrap(err[0].localizedDescription));
}
var totalRef = Ref();
if (!url.getResourceValueForKeyError(totalRef, $.NSURLVolumeTotalCapacityKey, err)) {
    throw new Error('failed to get total capacity: ' + ObjC.unwrap(err[0].localizedDescription));
}
JSON.stringify({available: availRef[0].js, total: totalRef[0].js})
`
	cmd := exec.CommandContext(ctx, "osascript", "-l", "JavaScript", "-e", script)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		log.Debug().Err(err).Str("stderr", stderr.String()).Msg("failed to get disk space via osascript")
		return 0, 0, fmt.Errorf("failed to run osascript: %w (stderr: %s)", err, stderr.String())
	}

	var result struct {
		Available int64 `json:"available"`
		Total     int64 `json:"total"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &result); err != nil {
		return 0, 0, fmt.Errorf("failed to parse disk space result (stdout: %q, stderr: %q): %w", out, stderr.String(), err)
	}

	return result.Available, result.Total, nil
}
