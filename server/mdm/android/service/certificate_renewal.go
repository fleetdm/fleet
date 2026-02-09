package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// RenewCertificateTemplates identifies Android certificate templates that are approaching
// expiration and marks them for renewal by updating their status.
func RenewCertificateTemplates(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger) error {
	const batchSize = 1000
	templates, err := ds.GetAndroidCertificateTemplatesForRenewal(ctx, batchSize)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get android certificate templates for renewal")
	}
	if len(templates) == 0 {
		return nil
	}
	if err := ds.SetAndroidCertificateTemplatesForRenewal(ctx, templates); err != nil {
		return ctxerr.Wrap(ctx, err, "set android certificate templates for renewal")
	}
	level.Info(logger).Log("msg", "marked android certificate templates for renewal", "count", len(templates))
	return nil
}
