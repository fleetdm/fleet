package service

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// GenerateActivity is the premium version of GenerateActivity that
// overrides the wrapped Service's GenerateActivity method.
func (svc *Service) GenerateActivity(
	ctx context.Context,
	user *fleet.User,
	activityType string,
	details *map[string]interface{},
) error {
	activity, err := fleet.CreateActivity(ctx, svc.ds, user, activityType, details)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "create activity")
	}

	if svc.config.Activity.EnableAuditLog {
		b, err := json.Marshal(activity)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "marshal activity")
		}
		if err := svc.auditLogger.Write(ctx, []json.RawMessage{json.RawMessage(b)}); err != nil {
			return ctxerr.Wrap(ctx, err, "log activity")
		}
	}

	return nil
}
