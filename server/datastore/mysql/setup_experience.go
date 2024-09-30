package mysql

import "context"

func (ds *Datastore) SetSetupExperienceSoftwareTitles(ctx context.Context, teamID uint, softwareTitleIDs []uint) error {
	stmtUnselectInstallers := `
UPDATE software_installers
SET install_during_setup = false
WHERE team_or_global_id = ?`

	stmtUnselectVPPAppsTeams := `
UPDATE vpp_apps_teams vat
SET install_during_setup = false
WHERE team_or_global_id = ?`

	stmtSelectInstallersIDs := `
SELECT
	si.id
FROM
	software_titles st
LEFT JOIN
	software_installers si
	ON st.id = si.title_id
WHERE
	global_or_team_id = ?
AND
	st.id IN (%s)
`

	stmtSelectVPPAppsTeamsID := `
SELECT
	vat.id
FROM
	software_titles st
LEFT JOIN
	vpp_apps va
	ON st.id = va.title_id
LEFT JOIN
	vpp_apps_teams vat
	ON va.adam_id = vat.adam_id
WHERE
	global_or_team_id = ?
AND
	st.id IN (%s)
`

	stmtUpdateInstallers := `
UPDATE software_installers
SET install_during_setup = true
WHERE id IN `

	stmtUpdateVPPApps := `
UPDATE vpp_apps_teams
SET install_during_setup = true
WHERE id IN`

	return nil
}

// func (ds *Datastore) ListSetupExperienceSoftwareTitles(ctx context.Context, teamID uint) ([]string, error) {
// 	return nil, nil
// }
