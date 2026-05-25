package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

var policyStatusAllowedOrderKeys = common_mysql.OrderKeyAllowlist{
	"consecutive_failures": "consecutive_failures",
	"created_at":           "created_at",
}

func (ds *Datastore) GetPolicyStatus(
	ctx context.Context,
	policyID uint,
	filter fleet.TeamFilter,
	req fleet.GetPolicyStatusRequest,
) ([]fleet.GetPolicyStatusPolicyRun, int, *fleet.PaginationMetadata, error) {
	teamFilter := ds.whereFilterHostsByTeams(filter, "h")

	// Part A: hosts that have a host_policy_runs row. Driving from host_policy_runs (INNER
	// JOIN) lets MySQL use the composite indexes on (policy_id, consecutive_failures)
	// and (policy_id, created_at) when Part B is excluded (automation filters).
	// When Part B is included the UNION still requires a merge sort; the indexes
	// speed up Part A's contribution to it.
	const partASelect = `
		SELECT
			h.id        AS host_id,
			h.hostname  AS host_name,
			h.platform  AS host_platform,
			COALESCE(pm.passes, 1) AS new_status,
			pr.consecutive_failures,
			pr.created_at,
			pr.id       AS policy_run_id
	`
	partAFrom := `
		FROM host_policy_runs pr
		JOIN policy_membership pm ON pm.policy_id = pr.policy_id AND pm.host_id = pr.host_id
		JOIN hosts h ON h.id = pr.host_id
	`
	// Team-scoping: a global policy's membership rows can span many teams; without
	// this filter a team observer querying a global policy could enumerate
	// hostnames from teams they have no access to.
	partAWhere := " WHERE pr.policy_id = ? AND " + teamFilter
	partAArgs := []any{policyID}

	// Part B: hosts in policy_membership with no host_policy_runs row. This covers
	// always-passing hosts but also failing hosts whose host_policy_runs row was not yet
	// written (async policy collection mode, transient write errors). pm.passes is
	// read from the table — not hardcoded — so those hosts are reported correctly.
	const partBSelect = `
		SELECT
			h.id          AS host_id,
			h.hostname    AS host_name,
			h.platform    AS host_platform,
			COALESCE(pm.passes, 1) AS new_status,
			0             AS consecutive_failures,
			pm.updated_at AS created_at,
			NULL          AS policy_run_id
	`
	partBFrom := `
		FROM policy_membership pm
		JOIN hosts h ON pm.host_id = h.id
	`
	partBWhere := " WHERE pm.policy_id = ? AND " + teamFilter + `
		AND NOT EXISTS (
			SELECT 1 FROM host_policy_runs pr2
			WHERE pr2.policy_id = pm.policy_id AND pr2.host_id = pm.host_id
		)`
	partBArgs := []any{policyID}

	// Automation filters require a host_policy_runs row — Part B can never match them.
	// The policy_failed filter applies to both parts via pm.passes.
	includePartB := req.RunStatus != "automation_failed"

	if req.HostNameQuery != "" {
		partAWhere += " AND h.hostname LIKE ?"
		partAArgs = append(partAArgs, "%"+req.HostNameQuery+"%")
		if includePartB {
			partBWhere += " AND h.hostname LIKE ?"
			partBArgs = append(partBArgs, "%"+req.HostNameQuery+"%")
		}
	}

	switch req.RunStatus {
	case "policy_failed":
		partAWhere += " AND pm.passes = 0"
		partBWhere += " AND pm.passes = 0"
	case "automation_failed":
		// Correlated EXISTS over pr.id is cheaper than IN(SELECT UNION SELECT ...):
		// the planner reuses the outer pr row and short-circuits on the first match.
		// VPP install failures live across host_vpp_software_installs and the
		// associated nano_command_results row (see fetchAutomationsForPolicyRuns
		// for the full status mapping); we mirror the "verification_failed_at
		// set OR MDM error response" failure shape here.
		partAWhere += ` AND (
				EXISTS (
					SELECT 1
					FROM host_policy_runs_to_policy_automation_executions prpa
					JOIN policy_automation_executions pa ON prpa.batch_id = pa.batch_id
					WHERE prpa.policy_run_id = pr.id AND pa.status = 'failure'
				)
				OR EXISTS (
					SELECT 1 FROM host_script_results hsr
					WHERE hsr.policy_run_id = pr.id AND hsr.exit_code IS NOT NULL AND hsr.exit_code != 0
				)
				OR EXISTS (
					SELECT 1 FROM host_software_installs hsi
					WHERE hsi.policy_run_id = pr.id AND hsi.execution_status IN ('failed_install', 'failed_uninstall')
				)
				OR EXISTS (
					SELECT 1 FROM host_vpp_software_installs hvsi
					LEFT JOIN nano_command_results ncr ON ncr.command_uuid = hvsi.command_uuid
					WHERE hvsi.policy_run_id = pr.id
					  AND (
						hvsi.verification_failed_at IS NOT NULL
						OR ncr.status IN ('Error', 'CommandFormatError')
					  )
				)
			)`
	}

	// COUNT queries run separately for each part to avoid a subquery wrapper.
	var totalCount int
	countAQuery := "SELECT COUNT(*) " + partAFrom + partAWhere
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &totalCount, countAQuery, partAArgs...); err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "count policy status runs")
	}
	if includePartB {
		var countB int
		countBQuery := "SELECT COUNT(*) " + partBFrom + partBWhere
		if err := sqlx.GetContext(ctx, ds.reader(ctx), &countB, countBQuery, partBArgs...); err != nil {
			return nil, 0, nil, ctxerr.Wrap(ctx, err, "count policy status passing hosts")
		}
		totalCount += countB
	}

	// IncludeMetadata=true makes appendListOptionsToSQLSecure emit LIMIT n+1
	// so the trim logic below can populate HasNextResults.
	req.ListOptions.IncludeMetadata = true

	// Pre-allocate to avoid aliasing partAArgs' backing array on append.
	dataArgs := make([]any, 0, len(partAArgs)+len(partBArgs))
	var dataQuery string
	if includePartB {
		// Wrap the UNION ALL in a derived table so that appendListOptionsToSQLSecure
		// appends ORDER BY and cursor conditions (AND col op ?) at the outer level.
		// Without the wrapper, AND-clauses would attach to Part B's WHERE only.
		// WHERE 1=1 gives the outer query a WHERE clause for the cursor AND to join.
		dataQuery = "SELECT * FROM (\n" +
			partASelect + partAFrom + partAWhere +
			"\nUNION ALL\n" +
			partBSelect + partBFrom + partBWhere +
			"\n) _union WHERE 1=1"
		dataArgs = append(dataArgs, partAArgs...)
		dataArgs = append(dataArgs, partBArgs...)
	} else {
		dataQuery = partASelect + partAFrom + partAWhere
		dataArgs = append(dataArgs, partAArgs...)
	}

	var err error
	var listOptArgs []any
	dataQuery, listOptArgs, err = appendListOptionsToSQLSecure(dataQuery, &req.ListOptions, policyStatusAllowedOrderKeys)
	if err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "append list options to sql")
	}
	// listOptArgs carries the cursor value (if After was set); append after the
	// query-body args so placeholder positions match.
	dataArgs = append(dataArgs, listOptArgs...)

	var runs []struct {
		fleet.GetPolicyStatusPolicyRun
		PolicyRunID  *uint  `db:"policy_run_id"`
		HostPlatform string `db:"host_platform"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &runs, dataQuery, dataArgs...); err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "select policy status runs")
	}

	hasNext := req.ListOptions.PerPage > 0 && uint(len(runs)) > req.ListOptions.PerPage //nolint:gosec // len() result fits in uint here
	if hasNext {
		runs = runs[:len(runs)-1]
	}

	meta := &fleet.PaginationMetadata{
		HasNextResults:     hasNext,
		HasPreviousResults: req.ListOptions.Page > 0,
		TotalResults:       uint(totalCount), //nolint:gosec // dismiss G115
	}

	if len(runs) == 0 {
		return nil, totalCount, meta, nil
	}

	var runIDs []uint
	for _, r := range runs {
		if r.PolicyRunID != nil {
			runIDs = append(runIDs, *r.PolicyRunID)
		}
	}

	automationsByRun := map[uint][]fleet.GetPolicyStatusAutomationExecution{}
	if len(runIDs) > 0 {
		automationsByRun, err = ds.fetchAutomationsForPolicyRuns(ctx, runIDs)
		if err != nil {
			return nil, 0, nil, err
		}
	}

	// Synthesize entries for automations that were configured on the policy but
	// could not run on this host (platform mismatch → not_compatible, software
	// installer label scope mismatch → not_in_target). The osquery hot path
	// skips these silently — without this augmentation the UI would show no
	// automation row at all, leaving the operator unable to distinguish "never
	// configured" from "couldn't run."
	failingRunHosts := make(map[uint]uint, len(runs)) // runID → hostID
	failingRunPlatforms := make(map[uint]string, len(runs))
	for _, r := range runs {
		if r.PolicyRunID == nil || r.NewStatus {
			continue
		}
		failingRunHosts[*r.PolicyRunID] = r.HostID
		failingRunPlatforms[*r.PolicyRunID] = r.HostPlatform
	}
	if len(failingRunHosts) > 0 {
		if err := ds.augmentSkippedAutomations(ctx, policyID, failingRunHosts, failingRunPlatforms, automationsByRun); err != nil {
			return nil, 0, nil, err
		}
	}

	out := make([]fleet.GetPolicyStatusPolicyRun, 0, len(runs))
	for _, r := range runs {
		row := r.GetPolicyStatusPolicyRun
		if r.PolicyRunID != nil {
			if autos, ok := automationsByRun[*r.PolicyRunID]; ok {
				row.AutomationExecutions = autos
			}
		}
		out = append(out, row)
	}

	return out, totalCount, meta, nil
}

// augmentSkippedAutomations adds synthetic rows to automationsByRun for any
// policy_run that is currently failing AND has a policy-configured automation
// (script_id or software_installer_id) AND has no actual execution row for that
// automation type. The osquery hot path skips scheduling on:
//   - script extension vs host platform mismatch → not_compatible
//   - installer platform vs host platform mismatch → not_compatible
//   - installer label-scope mismatch → not_in_target
//
// When a real row exists (even queued), the synthetic row is suppressed: real
// state is always more informative than the synthesized "would-have-skipped"
// reason.
func (ds *Datastore) augmentSkippedAutomations(
	ctx context.Context,
	policyID uint,
	failingRunHosts map[uint]uint,
	failingRunPlatforms map[uint]string,
	automationsByRun map[uint][]fleet.GetPolicyStatusAutomationExecution,
) error {
	var policyCfg struct {
		ScriptID            *uint `db:"script_id"`
		SoftwareInstallerID *uint `db:"software_installer_id"`
	}
	err := sqlx.GetContext(ctx, ds.reader(ctx), &policyCfg,
		`SELECT script_id, software_installer_id FROM policies WHERE id = ?`, policyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return ctxerr.Wrap(ctx, err, "fetch policy automation config")
	}
	if policyCfg.ScriptID == nil && policyCfg.SoftwareInstallerID == nil {
		return nil
	}

	var scriptName string
	if policyCfg.ScriptID != nil {
		if err := sqlx.GetContext(ctx, ds.reader(ctx), &scriptName,
			`SELECT name FROM scripts WHERE id = ?`, *policyCfg.ScriptID); err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return ctxerr.Wrap(ctx, err, "fetch policy script name")
			}
			// Script row disappeared (e.g., deleted concurrently); skip the
			// script branch — no row means we can't say "not_compatible" with
			// confidence.
			policyCfg.ScriptID = nil
		}
	}

	var installerPlatform string
	var softwareTitleName string
	if policyCfg.SoftwareInstallerID != nil {
		var installerInfo struct {
			Platform  string `db:"platform"`
			TitleName string `db:"title_name"`
		}
		if err := sqlx.GetContext(ctx, ds.reader(ctx), &installerInfo,
			`SELECT si.platform, COALESCE(st.name, '') AS title_name
			 FROM software_installers si
			 LEFT JOIN software_titles st ON st.id = si.title_id
			 WHERE si.id = ?`, *policyCfg.SoftwareInstallerID); err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return ctxerr.Wrap(ctx, err, "fetch policy installer platform")
			}
			policyCfg.SoftwareInstallerID = nil
		} else {
			installerPlatform = installerInfo.Platform
			softwareTitleName = installerInfo.TitleName
		}
	}

	// Pre-compute label scope for all candidate hosts in one batch query instead
	// of one DB call per host. A host needs the check only when its platform
	// matches the installer and it has no existing software_installation row.
	// nil map is fine: reading a nil map[uint]struct{} returns the zero value.
	var installerInScope map[uint]struct{}
	if policyCfg.SoftwareInstallerID != nil {
		var candidates []uint
		for runID, hostID := range failingRunHosts {
			if fleet.PlatformFromHost(failingRunPlatforms[runID]) == installerPlatform &&
				!hasAutomationOfType(automationsByRun[runID], "software_installation") {
				candidates = append(candidates, hostID)
			}
		}
		if len(candidates) > 0 {
			var scopeErr error
			installerInScope, scopeErr = ds.batchInstallerLabelScope(ctx, *policyCfg.SoftwareInstallerID, candidates)
			if scopeErr != nil {
				return ctxerr.Wrap(ctx, scopeErr, "batch check installer label scope")
			}
		}
	}

	for runID, hostID := range failingRunHosts {
		hostPlatform := fleet.PlatformFromHost(failingRunPlatforms[runID])
		existing := automationsByRun[runID]

		if policyCfg.ScriptID != nil && !hasAutomationOfType(existing, "script_run") {
			if scriptNotCompatible(hostPlatform, scriptName) {
				automationsByRun[runID] = append(existing, fleet.GetPolicyStatusAutomationExecution{
					Type:   "script_run",
					Status: "not_compatible",
					Name:   scriptName,
				})
				existing = automationsByRun[runID]
			}
		}

		if policyCfg.SoftwareInstallerID != nil && !hasAutomationOfType(existing, "software_installation") {
			if hostPlatform != installerPlatform {
				automationsByRun[runID] = append(existing, fleet.GetPolicyStatusAutomationExecution{
					Type:   "software_installation",
					Status: "not_compatible",
					Name:   softwareTitleName,
				})
				continue
			}
			if _, ok := installerInScope[hostID]; !ok {
				automationsByRun[runID] = append(existing, fleet.GetPolicyStatusAutomationExecution{
					Type:   "software_installation",
					Status: "not_in_target",
					Name:   softwareTitleName,
				})
			}
		}
	}
	return nil
}

// batchInstallerLabelScope returns a set of host IDs that are in scope for
// the given software installer. It mirrors the four-branch
// logic of isSoftwareLabelScoped but evaluates all supplied hosts in a single
// round-trip instead of one query per host.
//
// Branches (any match → in scope):
//  1. No labels on the installer → all hosts in scope (Go-level fast path).
//  2. Include-any: host is a member of at least one include-any label.
//  3. Exclude-any: installer has exclude-any labels, all are resolved for the
//     host, and the host is not a member of any of them.
//  4. Include-all: host is a member of every include-all label.
func (ds *Datastore) batchInstallerLabelScope(ctx context.Context, installerID uint, hostIDs []uint) (map[uint]struct{}, error) {
	if len(hostIDs) == 0 {
		return nil, nil
	}

	// Branch 1: no labels → all hosts are in scope.
	var labelCount int
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &labelCount,
		`SELECT COUNT(*) FROM software_installer_labels WHERE software_installer_id = ?`,
		installerID,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "count installer labels for scope check")
	}
	if labelCount == 0 {
		result := make(map[uint]struct{}, len(hostIDs))
		for _, hid := range hostIDs {
			result[hid] = struct{}{}
		}
		return result, nil
	}

	// Branches 2–4 in a single UNION query that returns the in-scope host IDs.
	const q = `
		SELECT DISTINCT host_id FROM (

			/* include any: host is a member of at least one include-any label */
			SELECT lm.host_id
			FROM software_installer_labels sil
			JOIN label_membership lm ON lm.label_id = sil.label_id
			WHERE sil.software_installer_id = ?
			  AND sil.exclude = 0 AND sil.require_all = 0
			  AND lm.host_id IN (?)
			GROUP BY lm.host_id
			HAVING COUNT(lm.label_id) > 0

			UNION

			/* exclude any: host is not a member of any evaluated exclude-any label */
			SELECT h.id AS host_id
			FROM hosts h
			WHERE h.id IN (?)
			  AND EXISTS (
			      SELECT 1 FROM software_installer_labels
			      WHERE software_installer_id = ? AND exclude = 1 AND require_all = 0
			  )
			  AND NOT EXISTS (
			      SELECT 1 FROM software_installer_labels sil2
			      JOIN label_membership lm2 ON lm2.label_id = sil2.label_id
			      WHERE sil2.software_installer_id = ?
			        AND sil2.exclude = 1 AND sil2.require_all = 0
			        AND lm2.host_id = h.id
			  )
			  AND (
			      SELECT COUNT(*)
			      FROM software_installer_labels sil3
			      JOIN labels lbl ON lbl.id = sil3.label_id
			      WHERE sil3.software_installer_id = ?
			        AND sil3.exclude = 1 AND sil3.require_all = 0
			        AND (lbl.label_membership_type = 1 OR h.label_updated_at >= lbl.created_at)
			  ) = (
			      SELECT COUNT(*) FROM software_installer_labels
			      WHERE software_installer_id = ? AND exclude = 1 AND require_all = 0
			  )

			UNION

			/* include all: host is a member of every include-all label */
			SELECT lm.host_id
			FROM label_membership lm
			JOIN software_installer_labels sil ON sil.label_id = lm.label_id
			WHERE sil.software_installer_id = ?
			  AND sil.exclude = 0 AND sil.require_all = 1
			  AND lm.host_id IN (?)
			GROUP BY lm.host_id
			HAVING COUNT(lm.label_id) = (
			    SELECT COUNT(*) FROM software_installer_labels
			    WHERE software_installer_id = ? AND exclude = 0 AND require_all = 1
			)

		) in_scope
	`
	// Argument positions (10 total):
	//   branch 2: installerID, hostIDs
	//   branch 3: hostIDs, installerID (EXISTS), installerID (NOT EXISTS),
	//             installerID (first COUNT), installerID (second COUNT)
	//   branch 4: installerID, hostIDs, installerID (HAVING subquery)
	query, args, err := sqlx.In(q,
		installerID, hostIDs,
		hostIDs, installerID, installerID, installerID, installerID,
		installerID, hostIDs, installerID,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build batch installer label scope query")
	}

	var inScopeIDs []uint
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &inScopeIDs, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "query batch installer label scope")
	}

	result := make(map[uint]struct{}, len(inScopeIDs))
	for _, hid := range inScopeIDs {
		result[hid] = struct{}{}
	}
	return result, nil
}

func hasAutomationOfType(autos []fleet.GetPolicyStatusAutomationExecution, t string) bool {
	for _, a := range autos {
		if a.Type == t {
			return true
		}
	}
	return false
}

// scriptNotCompatible mirrors the platform check in
// server/service/osquery.go's failing-policy script trigger: `.sh` on Windows
// or `.ps1` on non-Windows is skipped. Any other extension is considered
// compatible (e.g. `.py`).
func scriptNotCompatible(hostPlatform, scriptName string) bool {
	switch {
	case hostPlatform == "windows" && strings.HasSuffix(scriptName, ".sh"):
		return true
	case hostPlatform != "windows" && strings.HasSuffix(scriptName, ".ps1"):
		return true
	}
	return false
}

func (ds *Datastore) fetchAutomationsForPolicyRuns(ctx context.Context, runIDs []uint) (map[uint][]fleet.GetPolicyStatusAutomationExecution, error) {
	// Each branch projects (policy_run_id, type, status, error_message). Status
	// is mapped to the API contract: success | failed | queued.
	//
	// Script + software branches gate the error_message projection on a
	// failure status, so the field is never populated with stdout from a
	// successful script.
	//
	// VPP installs surface as 'software_installation' (same type as
	// software-installer installs) so the UI doesn't have to distinguish the
	// two install sources. Status is derived from verification timestamps and
	// the MDM command result row, mirroring GetSummaryHostVPPAppInstalls.
	//
	// Upcoming-activity rows are excluded when a result row already exists
	// for the same policy_run_id — without this dedupe the page would show
	// the same automation twice during the brief window between insert and
	// promotion-time delete (and permanently if the upcoming row is never
	// cleared).
	autoQuery := `
		SELECT
			prpa.policy_run_id,
			prpa.automation_type as type,
			CASE pa.status WHEN 'failure' THEN 'failed' WHEN 'pending' THEN 'queued' ELSE pa.status END as status,
			COALESCE(pa.error_message, '') as error_message,
			'' as name
		FROM host_policy_runs_to_policy_automation_executions prpa
		JOIN policy_automation_executions pa ON prpa.batch_id = pa.batch_id
		WHERE prpa.policy_run_id IN (?)

		UNION ALL

		SELECT
			hsr.policy_run_id,
			'script_run' as type,
			CASE WHEN hsr.exit_code = 0 THEN 'success'
			     WHEN hsr.exit_code IS NOT NULL THEN 'failed'
			     ELSE 'queued' END as status,
			CASE WHEN hsr.exit_code IS NOT NULL AND hsr.exit_code != 0
			     THEN COALESCE(hsr.output, '') ELSE '' END as error_message,
			COALESCE(s.name, '') as name
		FROM host_script_results hsr
		LEFT JOIN scripts s ON s.id = hsr.script_id
		WHERE hsr.policy_run_id IN (?)

		UNION ALL

		SELECT
			sua.policy_run_id,
			'script_run' as type,
			'queued' as status,
			'' as error_message,
			COALESCE(s.name, '') as name
		FROM script_upcoming_activities sua
		LEFT JOIN scripts s ON s.id = sua.script_id
		WHERE sua.policy_run_id IN (?)
		  AND NOT EXISTS (
			SELECT 1 FROM host_script_results hsr2
			WHERE hsr2.policy_run_id = sua.policy_run_id
		  )

		UNION ALL

		SELECT
			hsi.policy_run_id,
			'software_installation' as type,
			CASE
				WHEN hsi.execution_status = 'installed' THEN 'success'
				WHEN hsi.execution_status IN ('failed_install', 'failed_uninstall') THEN 'failed'
				ELSE 'queued'
			END as status,
			CASE
				WHEN hsi.execution_status = 'failed_install' THEN
					COALESCE(
						NULLIF(hsi.post_install_script_output, ''),
						NULLIF(hsi.install_script_output, ''),
						NULLIF(hsi.pre_install_query_output, ''),
						''
					)
				WHEN hsi.execution_status = 'failed_uninstall' THEN
					COALESCE(NULLIF(hsi.uninstall_script_output, ''), '')
				ELSE ''
			END as error_message,
			COALESCE(st.name, hsi.software_title_name) as name
		FROM host_software_installs hsi
		LEFT JOIN software_titles st ON st.id = hsi.software_title_id
		WHERE hsi.policy_run_id IN (?)

		UNION ALL

		SELECT
			siua.policy_run_id,
			'software_installation' as type,
			'queued' as status,
			'' as error_message,
			COALESCE(st.name, '') as name
		FROM software_install_upcoming_activities siua
		LEFT JOIN software_titles st ON st.id = siua.software_title_id
		WHERE siua.policy_run_id IN (?)
		  AND NOT EXISTS (
			SELECT 1 FROM host_software_installs hsi2
			WHERE hsi2.policy_run_id = siua.policy_run_id
		  )

		UNION ALL

		SELECT
			hvsi.policy_run_id,
			'software_installation' as type,
			CASE
				WHEN hvsi.verification_at IS NOT NULL THEN 'success'
				WHEN hvsi.verification_failed_at IS NOT NULL THEN 'failed'
				WHEN ncr.status IN ('Error', 'CommandFormatError') THEN 'failed'
				ELSE 'queued'
			END as status,
			'' as error_message,
			COALESCE(va.name, '') as name
		FROM host_vpp_software_installs hvsi
		LEFT JOIN nano_command_results ncr ON ncr.command_uuid = hvsi.command_uuid
		LEFT JOIN vpp_apps va ON va.adam_id = hvsi.adam_id AND va.platform = hvsi.platform
		WHERE hvsi.policy_run_id IN (?)

		UNION ALL

		SELECT
			vaua.policy_run_id,
			'software_installation' as type,
			'queued' as status,
			'' as error_message,
			COALESCE(va.name, '') as name
		FROM vpp_app_upcoming_activities vaua
		LEFT JOIN vpp_apps va ON va.adam_id = vaua.adam_id AND va.platform = vaua.platform
		WHERE vaua.policy_run_id IN (?)
		  AND NOT EXISTS (
			SELECT 1 FROM host_vpp_software_installs hvsi2
			WHERE hvsi2.policy_run_id = vaua.policy_run_id
		  )
	`
	autoQuery, autoArgs, err := sqlx.In(autoQuery, runIDs, runIDs, runIDs, runIDs, runIDs, runIDs, runIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build automations query")
	}

	var automations []struct {
		PolicyRunID  uint   `db:"policy_run_id"`
		Type         string `db:"type"`
		Status       string `db:"status"`
		ErrorMessage string `db:"error_message"`
		Name         string `db:"name"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &automations, autoQuery, autoArgs...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select policy status automations")
	}

	out := make(map[uint][]fleet.GetPolicyStatusAutomationExecution, len(runIDs))
	for _, a := range automations {
		out[a.PolicyRunID] = append(out[a.PolicyRunID], fleet.GetPolicyStatusAutomationExecution{
			Type:         a.Type,
			Status:       a.Status,
			ErrorMessage: a.ErrorMessage,
			Name:         a.Name,
		})
	}
	return out, nil
}
