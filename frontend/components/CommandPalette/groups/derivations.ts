import { ICommandPaletteContext } from "../helpers";

/**
 * Values derived once from the raw context and shared across all group
 * builders. Computed in `deriveContext` and passed as the second argument
 * to each builder so the derivation logic lives in exactly one place.
 */
export interface IDerivedContext {
  /** Apple Business Manager configured at the org level. */
  isAbmConfigured: boolean;
  /** GitOps mode active — disables Create-fleet etc. */
  isGitOpsMode: boolean;
  /** currentTeam is the "No team" / Unassigned sentinel (id 0). */
  isUnassigned: boolean;
  /** A specific team OR unassigned is selected (not "All fleets"). */
  hasTeamOrUnassigned: boolean;
  /** Chip label when navigating from Unassigned to an "All fleets" page. */
  switchesFromUnassigned: string | undefined;
  /**
   * Chip label when navigating from "All fleets" to a page that requires a
   * specific team (Controls etc.). Returns the default fleet name.
   */
  switchesFromAllFleets: string | undefined;
  /**
   * Chip label for team-required commands (add-hosts, manage-enroll-secrets)
   * — set to "Unassigned" only when on "All fleets" and would switch context.
   */
  teamRequiredDestination: string | undefined;
  /**
   * Chip label for default-context commands (add-report, software automations)
   * — set to "All fleets" only when on Unassigned and would switch context.
   */
  defaultDestination: string | undefined;
}

/** Run once per buildPaletteItems call; passed to every group builder. */
export const deriveContext = (ctx: ICommandPaletteContext): IDerivedContext => {
  const {
    config,
    currentTeam,
    availableTeams,
    hasTeamSelected,
    isPrimoMode,
  } = ctx;

  const isAbmConfigured = config?.mdm?.apple_bm_enabled_and_configured ?? false;

  // GitOps mode disables write actions in the UI; mirrors the predicate
  // ManageFleetsPage uses to disable its Create fleet button.
  const isGitOpsMode = !!(
    config?.gitops?.gitops_mode_enabled && config?.gitops?.repository_url
  );

  const isUnassigned = currentTeam?.id === 0;
  const hasTeamOrUnassigned = !!hasTeamSelected || isUnassigned;

  // In Primo Mode the user perceives a single-fleet install, so the
  // concept of "switching fleet context" doesn't apply. All destination
  // chips collapse to undefined.

  const switchesFromUnassigned =
    !isPrimoMode && isUnassigned ? "All fleets" : undefined;

  const getDefaultTeamName = (): string | undefined => {
    if (isPrimoMode) return undefined;
    if (hasTeamOrUnassigned) return undefined;
    const realFleets = availableTeams?.filter((t) => t.id > 0) ?? [];
    if (!realFleets.length) return undefined;
    const workstations = realFleets.find((t) => {
      const lower = t.name.toLowerCase();
      return lower === "workstations" || lower === "\u{1F4BB} workstations";
    });
    return (workstations ?? realFleets.sort((a, b) => a.id - b.id)[0])?.name;
  };
  const switchesFromAllFleets = getDefaultTeamName();

  const teamRequiredDestination =
    !isPrimoMode && !hasTeamSelected && !isUnassigned
      ? "Unassigned"
      : undefined;

  const defaultDestination =
    !isPrimoMode && isUnassigned ? "All fleets" : undefined;

  return {
    isAbmConfigured,
    isGitOpsMode,
    isUnassigned,
    hasTeamOrUnassigned,
    switchesFromUnassigned,
    switchesFromAllFleets,
    teamRequiredDestination,
    defaultDestination,
  };
};
