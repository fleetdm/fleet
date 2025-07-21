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

type KeyProtector struct {
	KeyProtector int `json:"KeyProtectorType"`
}

type BitLockerVolume struct {
	MountPoint    string         `json:"MountPoint"`
	KeyProtectors []KeyProtector `json:"KeyProtector"`
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

	output, err := tablehelpers.Exec(
		ctx,
		t.logger,
		30,
		[]string{"powershell.exe"},
		[]string{
			"-NoProfile",
			"-Command",
			"Get-BitLockerVolume | ConvertTo-Json -Depth 5",
		}, true)

	if err != nil {
		t.logger.Info().Err(err).Msg("failed to get BitLocker volume data")
		return nil, fmt.Errorf("failed to get BitLocker volume data: %w", err)
	}

	return t.parseOutput(output)

}

func (t *Table) parseOutput(output []byte) ([]map[string]string, error) {
	var results []map[string]string

	var volumes []BitLockerVolume
	if err := json.Unmarshal(output, &volumes); err != nil {
		t.logger.Info().Err(err).Msg("failed to parse BitLocker volume data")
		return nil, fmt.Errorf("failed to parse BitLocker volume data: %w", err)
	}

	for _, volume := range volumes {
		for _, protector := range volume.KeyProtectors {
			results = append(results, map[string]string{
				"drive_letter":       volume.MountPoint,
				"key_protector_type": fmt.Sprintf("%d", protector.KeyProtector),
			})
		}
	}

	return results, nil
}
