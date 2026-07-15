import { APP_CONTEXT_NO_TEAM_ID, ITeamSummary } from "interfaces/team";

/**
 * Returns the trailing fleet-context phrase used in picker empty states.
 * Mirrors the convention from EmptyVulnerabilitiesTable elsewhere in the
 * codebase:
 *   - Real team (id > 0)        → " in Engineering"
 *   - Unassigned (id === 0)     → " in this fleet"
 *   - All fleets (id === -1)    → "" (no suffix; context is global)
 *   - Undefined currentTeam     → "" (defensive)
 *
 * Used with copy like `\`No reports found${getFleetSuffix(currentTeam)}.\``.
 */
const getFleetSuffix = (currentTeam?: ITeamSummary): string => {
  if (!currentTeam) return "";
  if (currentTeam.id > 0) return ` in ${currentTeam.name}`;
  if (currentTeam.id === APP_CONTEXT_NO_TEAM_ID) return " in this fleet";
  return "";
};

export default getFleetSuffix;
