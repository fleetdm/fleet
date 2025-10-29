//go:build windows
// +build windows

package bitlocker_key_protectors

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog"
)

const name = "bitlocker_key_protectors"

type BitLockerVolume struct {
	MountPoint       string `json:"MountPoint"`
	KeyProtectorType int    `json:"KeyProtectorType"`
}

type Table struct {
	logger zerolog.Logger
}

func TablePlugin(logger zerolog.Logger) *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("drive_letter"),
		table.IntegerColumn("key_protector_type"),
	}

	t := &Table{
		logger: logger.With().Str("table", name).Logger(),
	}

	return table.NewPlugin(name, columns, t.generate)
}

func (t *Table) generate(
	ctx context.Context,
	queryContext table.QueryContext,
) ([]map[string]string, error) {

	cmd := `Get-BitlockerVolume | ForEach-Object {
		$vol = $_
		$vol.KeyProtector | ForEach-Object {
			[PSCustomObject]@{
				MountPoint = $vol.MountPoint
				KeyProtectorType = $_.KeyProtectorType
			}
		}
	} | ConvertTo-Json`

	output, err := tablehelpers.Exec(
		ctx,
		t.logger,
		30,
		[]string{"powershell.exe"},
		[]string{"-NoProfile", "-Command", cmd},
		true,
	)

	if err != nil {
		t.logger.Info().Err(err).Msg("failed to get BitLocker volume data")
		return nil, fmt.Errorf("failed to get BitLocker volume data: %w", err)
	}

	return t.parseOutput(output)

}

func (t *Table) parseOutput(output []byte) ([]map[string]string, error) {
	if len(output) == 0 {
		return nil, nil
	}

	var results []map[string]string

	// The PS cmdlet might return a list if the system has more than one volume,
	// so first we try to parse it as an array ...
	var volumes []BitLockerVolume
	if err := json.Unmarshal(output, &volumes); err != nil {
		// If array parsing fails, try parsing as single object ...
		var volume BitLockerVolume
		if err := json.Unmarshal(output, &volume); err != nil {
			t.logger.Info().Err(err).Msg("failed to parse BitLocker volume data")
			return nil, fmt.Errorf("failed to parse BitLocker volume data: %w", err)
		}
		volumes = []BitLockerVolume{volume}
	}

	for _, volume := range volumes {
		results = append(results, map[string]string{
			"drive_letter":       volume.MountPoint,
			"key_protector_type": fmt.Sprintf("%d", volume.KeyProtectorType),
		})
	}

	return results, nil
}
