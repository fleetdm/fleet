package mysql

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (ds *Datastore) UpdateCertificateStatus(ctx context.Context, hostUUID string, certificateTemplateID uint, status fleet.OSSettingsStatus) error {
	// Attempt to update the certificate status for the given host and template.
	result, err := ds.writer(ctx).ExecContext(ctx, `
    UPDATE host_certificate_templates
    SET status = ?
    WHERE host_uuid = ? AND certificate_template_id = ?
`, status, hostUUID, certificateTemplateID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ctxerr.Wrap(ctx, notFound("Label").WithMessage(fmt.Sprintf("No certificate found for host UUID '%s' and template ID '%s'", hostUUID, certificateTemplateID)))
	}

	return nil
}
