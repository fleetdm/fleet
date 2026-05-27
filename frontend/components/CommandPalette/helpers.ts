import { ITeamSummary } from "interfaces/team";
import { IConfig } from "interfaces/config";

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
    ...buildSettingsItems(ctx, derived),
    ...buildCommandsItems(ctx, derived),
    ...buildMdmItems(ctx, derived),
    ...buildAutomationsItems(ctx, derived),
  ];
};
