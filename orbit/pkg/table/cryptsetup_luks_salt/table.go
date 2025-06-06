package cryptsetup_luks_salt

import (
	"context"
	"errors"
	"fmt"
	"github.com/fleetdm/fleet/v4/orbit/pkg/luks"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
	"strings"
)

const TblName = "cryptsetup_luks_salt"
const requiredCriteria = "device"

type criteria struct {
	device string
}

func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("device"), // required
		table.TextColumn("key_slot"),
		table.TextColumn("salt"),
	}
}

func getCriteria(qContext table.QueryContext) (*criteria, error) {
	missingPropErr := fmt.Errorf(
		"the %s table requires the following columns in the where clause: %s",
		TblName,
		requiredCriteria,
	)
	if len(qContext.Constraints) == 0 {
		return nil, missingPropErr
	}
	for _, c := range strings.Split(requiredCriteria, ", ") {
		constraint, ok := qContext.Constraints[c]
		if !ok || len(constraint.Constraints) == 0 || len(constraint.Constraints[0].Expression) == 0 {
			return nil, missingPropErr
		}
		if constraint.Constraints[0].Operator != table.OperatorEquals {
			return nil, errors.New("only the = operator is supported on the where clause")
		}
	}
	return &criteria{
		device: qContext.Constraints["device"].Constraints[0].Expression,
	}, nil
}

func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	criteria, err := getCriteria(queryContext)
	if err != nil {
		log.Debug().Err(err).Msg("error parsing query criteria")
		return nil, err
	}

	result, err := luks.GetLuksDump(ctx, criteria.device)
	if err != nil {
		return nil, fmt.Errorf("failed to run luksDump: %w", err)
	}

	if result != nil {
		rows := make([]map[string]string, 0, len(result.Keyslots))
		for keySlot, entries := range result.Keyslots {
			rows = append(rows, map[string]string{
				"device":   criteria.device,
				"key_slot": keySlot,
				"salt":     entries.KDF.Salt,
			})
		}
		return rows, nil
	}
	return nil, nil
}
