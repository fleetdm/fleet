package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ListPendingSoftwareInstallDetails(ctx context.Context, hostID uint) ([]*fleet.SoftwareInstallDetails, error) {
	const stmt = `
  SELECT
    hsi.host_id AS host_id
    hsi.execution_id AS execution_id,
    hsi.software_installer_id AS installer_id,
    si.pre_install_query AS pre_install_condition,
    is.contents AS install_script,
    pis.contents AS post_install_script
  FROM
    host_software_installs hsi
  JOIN
    software_installers si
    ON hsi.software_installer_id = si.id
  JOIN
    script_contents is
    ON is.id = si.install_script_content_id
  JOIN
    script_contents pis
    ON pis.id = si.post_install_script_content_id
  WHERE
    hsi.host_id = ?
  AND
    hsi.install_script_exit_code IS NOT NULL
`

	var results []*fleet.SoftwareInstallDetails
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, stmt, hostID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list pending software installs")
	}
	return results, nil
}

func (ds *Datastore) GetSoftwareInstallDetails(ctx context.Context, executionId string) (*fleet.SoftwareInstallDetails, error) {
	const stmt = `
  SELECT
    hsi.host_id AS host_id
    hsi.execution_id AS execution_id,
    hsi.software_installer_id AS installer_id,
    si.pre_install_query AS pre_install_condition,
    is.contents AS install_script,
    pis.contents AS post_install_script
  FROM
    host_software_installs hsi
  JOIN
    software_installers si
    ON hsi.software_installer_id = si.id
  JOIN
    script_contents is
    ON is.id = si.install_script_content_id
  JOIN
    script_contents pis
    ON pis.id = si.post_install_script_content_id
  WHERE
    hsi.execution_id = ?
`

	var result *fleet.SoftwareInstallDetails
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &result, stmt, executionId); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list pending software installs")
	}
	return result, nil

}
