package fleet

import (
	"context"
)

// HistoricalDataActivityEmitter is the narrow interface needed by
// OnHistoricalDataChanged. Both the free service and the EE service
// satisfy it via their NewActivity method.
type HistoricalDataActivityEmitter interface {
	NewActivity(ctx context.Context, user *User, activity ActivityDetails) error
}

// OnHistoricalDataChanged is the hook called when historical_data config changes.
// It emits one activity per historical_data sub-key whose value differs between
// oldHD and newHD. fleetID and fleetName are nil for global toggles and populated
// for per-fleet toggles. Dataset names in the activity payload are the public
// config sub-keys ("uptime", "vulnerabilities"), not internal dataset names.
func OnHistoricalDataChanged(
	ctx context.Context,
	emitter HistoricalDataActivityEmitter,
	user *User,
	oldHD, newHD HistoricalDataSettings,
	fleetID *uint, fleetName *string,
) error {
	changes := []struct {
		dataset string
		oldVal  bool
		newVal  bool
	}{
		{"uptime", oldHD.Uptime, newHD.Uptime},
		{"vulnerabilities", oldHD.Vulnerabilities, newHD.Vulnerabilities},
	}
	for _, c := range changes {
		if c.oldVal == c.newVal {
			continue
		}
		var act ActivityDetails
		if c.newVal {
			act = ActivityTypeEnabledHistoricalDataset{
				Dataset:   c.dataset,
				FleetID:   fleetID,
				FleetName: fleetName,
			}
		} else {
			act = ActivityTypeDisabledHistoricalDataset{
				Dataset:   c.dataset,
				FleetID:   fleetID,
				FleetName: fleetName,
			}
		}
		if err := emitter.NewActivity(ctx, user, act); err != nil {
			return err
		}
	}
	return nil
}
