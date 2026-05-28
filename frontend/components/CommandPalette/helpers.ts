import { APP_CONTEXT_ALL_TEAMS_ID, ITeamSummary } from "interfaces/team";
import { IConfig } from "interfaces/config";
import paths from "router/paths";

import { deriveContext } from "./groups/derivations";
import buildPagesItems from "./groups/pages";
import buildControlsItems from "./groups/controls";
import buildSoftwareItems from "./groups/software";
import buildSettingsItems from "./groups/settings";
import buildCommandsItems from "./groups/commands";
import buildMdmItems from "./groups/mdm";
import buildAutomationsItems from "./groups/automations";

export interface ICommandSubItem {
  id: string;
  label: string;
  path: string;
  keywords?: string[];
}

export interface ICommandItem {
  id: string;
  label: string;
  group: string;
  path?: string;
  keywords?: string[];
  /** Displayed on the right when navigating would switch your team context */
  teamName?: string;
  /** Nested items shown when the parent is expanded via chevron */
  subItems?: ICommandSubItem[];
  /** Custom action instead of navigation */
  onAction?: () => void;
  /** True when selecting this item opens a sub-page (not a navigation). */
  opensSubPage?: boolean;
}

export interface ICommandPaletteContext {
  search: string;
  currentTeam?: ITeamSummary;
  availableTeams?: ITeamSummary[];
  config: IConfig | null;
  canAccessControls?: boolean;
  canWrite?: boolean;
  canRunLiveReport?: boolean;
  canAccessSettings?: boolean;
  canManagePolicyAutomations?: boolean;
  canManageSoftwareAutomations?: boolean;
  /** Mirrors ManageQueriesPage `canManageAutomations`:
   *  isGlobalAdmin || isTeamAdmin. canWrite includes maintainers and
   *  technicians, whom the destination page won't let manage report
   *  automations, so this narrower flag gates `manage-report-automations`. */
  canManageReportAutomations?: boolean;
  /** Mirrors Variables.tsx `canEdit` — only global admins and global
   *  maintainers can create custom variables. canWrite includes team
   *  roles and technicians, which the destination page rejects, so
   *  the palette uses this narrower flag for `add-custom-variable`. */
  canEditCustomVariable?: boolean;
  /** Mirrors SoftwarePage.tsx `canAddSoftware`:
   *  isGlobalAdmin || isGlobalMaintainer || isTeamAdmin (current team)
   *  || isTeamMaintainer (current team). Excludes technicians and
   *  cross-team admin/maintainers — neither can use the Add software
   *  page despite passing `canWrite`. Gates every software-add palette
   *  item (FMA, VPP, Android, custom package). */
  canAddSoftware?: boolean;
  isTechnician?: boolean;
  isPremiumTier?: boolean;
  isPrimoMode?: boolean;
  /** Reactive theme state — true when dark mode is active. Passed in so
   *  the toggle-dark-mode label re-renders when the user (or system)
   *  changes the theme while the palette is open. */
  isDarkMode?: boolean;
  isMacMdmEnabledAndConfigured?: boolean;
  isWindowsMdmEnabledAndConfigured?: boolean;
  isAndroidMdmEnabledAndConfigured?: boolean;
  isVppEnabled?: boolean;
  hasTeamSelected?: boolean;
  withTeamId: (path: string) => string;
  onToggleDarkMode: () => void;
  onViewHost: () => void;
  onViewSoftware: () => void;
  onViewSoftwareLibrary: () => void;
  onViewReport: () => void;
  onViewPolicy: () => void;
}

export const GROUPS = [
  "Pages",
  "Controls",
  "Software",
  "Settings",
  "MDM",
  "Automations",
  "Commands",
] as const;

/**
 * Top-level orchestrator. Each group's items live in its own file under
 * ./groups/, and all share the values derived once via deriveContext.
 * Cross-group order in the returned array doesn't affect rendering —
 * CommandPalette.tsx groups by `group` field and renders in GROUPS order.
 */
export const buildPaletteItems = (
  ctx: ICommandPaletteContext
): ICommandItem[] => {
  const derived = deriveContext(ctx);
  return [
    ...buildPagesItems(ctx, derived),
    ...buildControlsItems(ctx, derived),
    ...buildSoftwareItems(ctx, derived),
    ...buildSettingsItems(ctx),
    ...buildCommandsItems(ctx, derived),
    ...buildMdmItems(ctx, derived),
    ...buildAutomationsItems(ctx, derived),
  ];
};

// Pages that require a specific fleet — they can't render "All fleets" or
// (with some overlap) "Unassigned". When switching to either of those from
// one of these pages, fall back to Hosts which supports both.
const TEAM_REQUIRED_PREFIXES = [
  paths.CONTROLS,
  paths.SOFTWARE_LIBRARY,
  paths.NEW_REPORT,
];

/**
 * Build the next URL for a fleet switch initiated from the command palette.
 *
 * This bypasses useTeamIdParam.handleTeamChange (which owns per-page
 * `overrideParamsOnTeamChange` config), so it reproduces handleTeamChange's
 * generic strip rules (page, legacy team_id) plus the fleet-scoped Hosts
 * filter keys (script_batch_execution_*, software_status on switch to All).
 */
export const buildFleetSwitchUrl = ({
  pathname,
  currentSearch,
  fleetId,
}: {
  pathname: string;
  currentSearch: string;
  fleetId: number;
}): string => {
  const isAll = fleetId === APP_CONTEXT_ALL_TEAMS_ID;
  const isUnassignedTarget = fleetId === 0;
  const isOnTeamRequiredPage = TEAM_REQUIRED_PREFIXES.some((p) =>
    pathname.startsWith(p)
  );

  if ((isAll || isUnassignedTarget) && isOnTeamRequiredPage) {
    // For Unassigned, keep fleet_id=0 on the fallback URL.
    // useTeamIdParam coerces a missing param back to All fleets (-1),
    // which would silently undo the setCurrentTeam({id:0}) caller.
    return isUnassignedTarget
      ? `${paths.MANAGE_HOSTS}?fleet_id=0`
      : paths.MANAGE_HOSTS;
  }

  const params = new URLSearchParams(currentSearch);
  if (isAll) {
    params.delete("fleet_id");
  } else {
    params.set("fleet_id", String(fleetId));
  }
  params.delete("page");
  params.delete("team_id");
  params.delete("script_batch_execution_id");
  params.delete("script_batch_execution_status");
  if (isAll) {
    params.delete("software_status");
  }
  const qs = params.toString();
  return qs ? `${pathname}?${qs}` : pathname;
};
