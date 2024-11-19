package mysql

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetLinuxDiskEncryptionSummary(ctx context.Context, teamID *uint) (fleet.MDMLinuxDiskEncryptionSummary, error) {
	var args []interface{}
	var teamFilter string
	if teamID != nil {
		teamFilter = "AND h.team_id = ?"
		args = append(args, *teamID)
	} else {
		teamFilter = "AND h.team_id IS NULL"
	}

	stmt := fmt.Sprintf(`SELECT
			CASE WHEN hdek.base64_encrypted IS NOT NULL
					AND hdek.base64_encrypted != ''
					AND hdek.client_error = '' THEN
					'verified'
				WHEN hdek.client_error IS NOT NULL
					AND hdek.client_error != '' THEN
					'failed'
				WHEN hdek.base64_encrypted IS NULL
					OR (hdek.base64_encrypted = ''
					AND hdek.client_error = '') THEN
					'action_required'
				END AS status,
				COUNT(h.id) AS host_count
			FROM
				hosts h
				LEFT JOIN host_disk_encryption_keys hdek ON h.id = hdek.host_id
			WHERE
				(h.os_version LIKE '%%fedora%%'
				OR h.platform LIKE 'ubuntu')
				%s
			GROUP BY
				status`, teamFilter)

	type countRow struct {
		Status    string `db:"status"`
		HostCount uint   `db:"host_count"`
	}

	var counts []countRow
	summary := fleet.MDMLinuxDiskEncryptionSummary{}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &counts, stmt, args...); err != nil {
		return summary, err
	}

	for _, count := range counts {
		switch count.Status {
		case "verified":
			summary.Verified = count.HostCount
		case "action_required":
			summary.ActionRequired = count.HostCount
		case "failed":
			summary.Failed = count.HostCount
		}
	}

	return summary, nil
}
