package mysql

import (
	"context"
	"maps"
	"slices"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ConfigListsByTeamIDs(ctx context.Context, teamIDs []uint) (map[uint]*fleet.ConfigLists, error) {
	lists := make(map[uint]*fleet.ConfigLists, len(teamIDs))
	for _, id := range teamIDs {
		lists[id] = fleet.NewConfigLists()
	}
	if len(teamIDs) == 0 {
		return lists, nil
	}

	if err := ds.loadConfigListsProfiles(ctx, teamIDs, lists); err != nil {
		return nil, err
	}
	if err := ds.loadConfigListsScriptsAndSetup(ctx, teamIDs, lists); err != nil {
		return nil, err
	}
	if noTeam, ok := lists[0]; ok {
		vpp, err := ds.listVPPTokenAssignments(ctx)
		if err != nil {
			return nil, err
		}
		if vpp != nil {
			noTeam.VolumePurchasingProgram = vpp
		}
	}
	return lists, nil
}

// Unlike ListVPPTokens, this doesn't decrypt the tokens, so a token that can't
// be decrypted doesn't break the config response.
func (ds *Datastore) listVPPTokenAssignments(ctx context.Context) ([]fleet.MDMAppleVolumePurchasingProgramInfo, error) {
	const stmt = `
SELECT v.id, v.location, vt.team_id, vt.null_team_type, COALESCE(t.name, '') AS team_name
FROM vpp_tokens v
LEFT JOIN vpp_token_teams vt ON vt.vpp_token_id = v.id
LEFT JOIN teams t ON t.id = vt.team_id
ORDER BY v.id, team_name
`
	var rows []struct {
		ID       uint                `db:"id"`
		Location string              `db:"location"`
		TeamID   *uint               `db:"team_id"`
		NullTeam *fleet.NullTeamType `db:"null_team_type"`
		TeamName string              `db:"team_name"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select VPP token assignments")
	}

	var tokens []*fleet.VPPTokenDB
	byID := make(map[uint]*fleet.VPPTokenDB)
	for _, r := range rows {
		tok, ok := byID[r.ID]
		if !ok {
			tok = &fleet.VPPTokenDB{ID: r.ID, Location: r.Location}
			byID[r.ID] = tok
			tokens = append(tokens, tok)
		}
		if r.NullTeam == nil {
			continue // no vpp_token_teams row: not assigned to any fleet
		}
		switch *r.NullTeam {
		case fleet.NullTeamAllTeams:
			tok.Teams = []fleet.TeamTuple{}
		case fleet.NullTeamNoTeam:
			tok.Teams = append(tok.Teams, fleet.TeamTuple{ID: 0, Name: fleet.TeamNameNoTeam})
		case fleet.NullTeamNone:
			if r.TeamID != nil {
				tok.Teams = append(tok.Teams, fleet.TeamTuple{ID: *r.TeamID, Name: r.TeamName})
			}
		}
	}
	return fleet.VolumePurchasingProgramFromTokens(tokens), nil
}

func (ds *Datastore) loadConfigListsProfiles(ctx context.Context, teamIDs []uint, lists map[uint]*fleet.ConfigLists) error {
	// Same filters as ListMDMConfigProfiles, so the lists match the List
	// configuration profiles endpoint.
	const profilesStmt = `
SELECT 'darwin' AS platform, profile_uuid, team_id, name, description, self_service, hidden
FROM mdm_apple_configuration_profiles
WHERE team_id IN (?) AND identifier NOT IN (?)

UNION ALL

SELECT 'darwin' AS platform, declaration_uuid AS profile_uuid, team_id, name, description, 0 AS self_service, hidden
FROM mdm_apple_declarations
WHERE team_id IN (?) AND name NOT IN (?)

UNION ALL

SELECT 'windows' AS platform, profile_uuid, team_id, name, description, 0 AS self_service, hidden
FROM mdm_windows_configuration_profiles
WHERE team_id IN (?) AND name NOT IN (?)

UNION ALL

SELECT 'android' AS platform, profile_uuid, team_id, name, description, 0 AS self_service, hidden
FROM mdm_android_configuration_profiles
WHERE team_id IN (?) AND name NOT IN (?)

ORDER BY name, profile_uuid
`
	// Joined on team_id rather than filtered by profile UUID so the number of
	// placeholders doesn't grow with the number of profiles across fleets.
	const labelsStmt = `
SELECT mcpl.apple_profile_uuid AS profile_uuid, mcpl.label_name, mcpl.exclude, mcpl.require_all
FROM mdm_configuration_profile_labels mcpl
JOIN mdm_apple_configuration_profiles p ON p.profile_uuid = mcpl.apple_profile_uuid
WHERE p.team_id IN (?)

UNION ALL

SELECT mdl.apple_declaration_uuid AS profile_uuid, mdl.label_name, mdl.exclude, mdl.require_all
FROM mdm_declaration_labels mdl
JOIN mdm_apple_declarations d ON d.declaration_uuid = mdl.apple_declaration_uuid
WHERE d.team_id IN (?)

UNION ALL

SELECT mcpl.windows_profile_uuid AS profile_uuid, mcpl.label_name, mcpl.exclude, mcpl.require_all
FROM mdm_configuration_profile_labels mcpl
JOIN mdm_windows_configuration_profiles p ON p.profile_uuid = mcpl.windows_profile_uuid
WHERE p.team_id IN (?)

UNION ALL

SELECT mcpl.android_profile_uuid AS profile_uuid, mcpl.label_name, mcpl.exclude, mcpl.require_all
FROM mdm_configuration_profile_labels mcpl
JOIN mdm_android_configuration_profiles p ON p.profile_uuid = mcpl.android_profile_uuid
WHERE p.team_id IN (?)

ORDER BY label_name
`

	fleetIdentifiers := slices.Collect(maps.Keys(mobileconfig.FleetPayloadIdentifiers()))
	fleetNames := slices.Collect(maps.Keys(mdm.FleetReservedProfileNames()))

	stmt, args, err := sqlx.In(profilesStmt,
		teamIDs, fleetIdentifiers, teamIDs, fleetNames, teamIDs, fleetNames, teamIDs, fleetNames)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "sqlx.In config lists profiles")
	}
	var profiles []struct {
		Platform    string `db:"platform"`
		ProfileUUID string `db:"profile_uuid"`
		TeamID      uint   `db:"team_id"`
		Name        string `db:"name"`
		Description string `db:"description"`
		SelfService bool   `db:"self_service"`
		Hidden      bool   `db:"hidden"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &profiles, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "select config lists profiles")
	}
	if len(profiles) == 0 {
		return nil
	}

	stmt, args, err = sqlx.In(labelsStmt, teamIDs, teamIDs, teamIDs, teamIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "sqlx.In config lists profile labels")
	}
	var labels []struct {
		ProfileUUID string `db:"profile_uuid"`
		LabelName   string `db:"label_name"`
		Exclude     bool   `db:"exclude"`
		RequireAll  bool   `db:"require_all"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labels, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "select config lists profile labels")
	}

	specs := make(map[string]*fleet.MDMProfileSpec, len(profiles))
	for _, p := range profiles {
		specs[p.ProfileUUID] = &fleet.MDMProfileSpec{
			Name:        p.Name,
			Description: p.Description,
			SelfService: p.SelfService,
			Hidden:      p.Hidden,
		}
	}
	for _, l := range labels {
		spec, ok := specs[l.ProfileUUID]
		if !ok {
			continue
		}
		switch {
		case !l.Exclude && l.RequireAll:
			spec.LabelsIncludeAll = append(spec.LabelsIncludeAll, l.LabelName)
		case !l.Exclude && !l.RequireAll:
			spec.LabelsIncludeAny = append(spec.LabelsIncludeAny, l.LabelName)
		case l.Exclude && !l.RequireAll:
			spec.LabelsExcludeAny = append(spec.LabelsExcludeAny, l.LabelName)
		}
	}

	for _, p := range profiles {
		l, ok := lists[p.TeamID]
		spec, specOK := specs[p.ProfileUUID]
		if !ok || !specOK {
			continue
		}
		switch p.Platform {
		case "darwin":
			l.AppleProfiles = append(l.AppleProfiles, *spec)
		case "windows":
			l.WindowsProfiles = append(l.WindowsProfiles, *spec)
		case "android":
			l.AndroidProfiles = append(l.AndroidProfiles, *spec)
		}
	}
	return nil
}

func (ds *Datastore) loadConfigListsScriptsAndSetup(ctx context.Context, teamIDs []uint, lists map[uint]*fleet.ConfigLists) error {
	const stmt = `
SELECT 'script' AS kind, global_or_team_id AS team_id, name
FROM scripts
WHERE global_or_team_id IN (?)

UNION ALL

SELECT 'setup_script' AS kind, global_or_team_id AS team_id, name
FROM setup_experience_scripts
WHERE global_or_team_id IN (?)

UNION ALL

SELECT 'setup_assistant' AS kind, global_or_team_id AS team_id, name
FROM mdm_apple_setup_assistants
WHERE global_or_team_id IN (?)

UNION ALL

SELECT 'bootstrap_package' AS kind, team_id, COALESCE(name, '') AS name
FROM mdm_apple_bootstrap_packages
WHERE team_id IN (?)

ORDER BY name
`
	query, args, err := sqlx.In(stmt, teamIDs, teamIDs, teamIDs, teamIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "sqlx.In config lists scripts and setup experience")
	}
	var rows []struct {
		Kind   string `db:"kind"`
		TeamID uint   `db:"team_id"`
		Name   string `db:"name"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, query, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "select config lists scripts and setup experience")
	}

	for _, r := range rows {
		l, ok := lists[r.TeamID]
		if !ok {
			continue
		}
		switch r.Kind {
		case "script":
			l.Scripts = append(l.Scripts, r.Name)
		case "setup_script":
			l.SetupScript = r.Name
		case "setup_assistant":
			l.SetupAssistant = r.Name
		case "bootstrap_package":
			l.BootstrapPackage = r.Name
		}
	}
	return nil
}
