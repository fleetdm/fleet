package cryptsetup_luks_salt

import (
	"context"
	"errors"
	"fmt"
	"github.com/fleetdm/fleet/v4/orbit/pkg/luks"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
	"strconv"
	"strings"
)

const TblName = "cryptsetup_luks_salt"
const requiredCriteria = "key_slot, device"

type criteria struct {
	keySlot uint
	device  string
}

func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("key_slot"), // required
		table.TextColumn("device"),   // required
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

	result := criteria{}
	kS := qContext.Constraints["key_slot"]
	kSVal, err := strconv.ParseUint(kS.Constraints[0].Expression, 10, 64)
	if err != nil || kSVal < 0 {
		return nil, errors.New("key_slot must be an integer greater than zero")
	}

	result.keySlot = uint(kSVal)
	result.device = qContext.Constraints["device"].Constraints[0].Expression
	return &result, nil
}

func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	criteria, err := getCriteria(queryContext)
	if err != nil {
		log.Debug().Err(err).Msg("error parsing query criteria")
		return nil, err
	}

	storedSalt, err := luks.GetSaltForKeySlot(ctx, criteria.device, criteria.keySlot)
	if err != nil {
		if errors.Is(err, luks.ErrKeySlotNotFound) {
			log.Debug().Msgf("key slot %d not found", criteria.keySlot)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get salt for key slot: %w", err)
	}

	return []map[string]string{{
		"device":   criteria.device,
		"key_slot": strconv.Itoa(int(criteria.keySlot)),
		"salt":     storedSalt,
	}}, nil
}
