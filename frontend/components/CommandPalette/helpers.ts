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

// Best match needs at least 2 characters total. At 2 chars we restrict
// to the strongest label tiers (exact + prefix) so the noise floor stays
// reasonable — typing "os" should promote "OS settings"/"OS updates"
// without pulling in every keyword that contains "os". 3+ chars unlocks
// the full ladder including word-prefix, substring, and keyword tiers.
export const BEST_MATCH_MIN_QUERY = 2;
export const BEST_MATCH_FULL_LADDER_MIN = 3;

// Tier scores for Best match. Labels are the primary name of an action,
// so even a weak label hit (substring, 70) outranks the strongest
// keyword hit (exact, 50). The gap is intentional — keywords are
// synonyms/aliases, not the canonical surface.
//
// Substring tier applies to labels only — keywords are short and noisy
// as substrings ("u" inside "user" would promote too much), so they
// stop at word-prefix.
export const SCORE_LABEL_EXACT = 100;
export const SCORE_LABEL_PREFIX = 90;
export const SCORE_LABEL_WORD_PREFIX = 80;
export const SCORE_LABEL_SUBSTRING = 70;
export const SCORE_KEYWORD_EXACT = 50;
export const SCORE_KEYWORD_PREFIX = 40;
export const SCORE_KEYWORD_WORD_PREFIX = 30;

// Word boundaries split on whitespace OR hyphens. "API-only user" yields
// ["api", "only", "user"] so a query for "only" word-prefix-matches.
const WORD_SPLIT = /[\s-]+/;

/**
 * Score a single text (label or keyword) against the search query.
 * Returns 0 when no rule matches, or one of the SCORE_* tier values.
 *
 * `isLabel` controls both which tier ladder applies (labels score
 * higher across the board) and whether the substring tier is allowed
 * (labels yes, keywords no — see header comment).
 *
 * Inputs are expected pre-lowercased by the caller; avoids
 * re-lowercasing per-token in hot loops.
 */
export const scoreMatch = (
  textLower: string,
  queryLower: string,
  isLabel: boolean
): number => {
  if (!textLower || !queryLower) return 0;
  if (textLower === queryLower) {
    return isLabel ? SCORE_LABEL_EXACT : SCORE_KEYWORD_EXACT;
  }
  if (textLower.startsWith(queryLower)) {
    return isLabel ? SCORE_LABEL_PREFIX : SCORE_KEYWORD_PREFIX;
  }
  // Word-prefix: any word starts with the query. Skip index 0 — that
  // case is already covered by startsWith above.
  const words = textLower.split(WORD_SPLIT);
  for (let i = 1; i < words.length; i += 1) {
    if (words[i].startsWith(queryLower)) {
      return isLabel ? SCORE_LABEL_WORD_PREFIX : SCORE_KEYWORD_WORD_PREFIX;
    }
  }
  // Substring is a label-only tier.
  if (isLabel && textLower.includes(queryLower)) return SCORE_LABEL_SUBSTRING;
  return 0;
};

/**
 * Score one token across an item — max over (label, ...keywords). Used
 * inside the multi-token pass.
 */
const scoreTokenAgainstItem = (
  labelLower: string,
  keywordsLower: string[],
  tokenLower: string
): number => {
  let best = scoreMatch(labelLower, tokenLower, true);
  if (best === SCORE_LABEL_EXACT) return best;
  for (let i = 0; i < keywordsLower.length; i += 1) {
    const s = scoreMatch(keywordsLower[i], tokenLower, false);
    if (s > best) best = s;
  }
  return best;
};

/**
 * Score an item against the search query.
 *
 * Two passes:
 *  1. Single-pass: score the full query as one string against label and
 *     each keyword (this is what catches phrase-level matches like
 *     "create user" exact-matching a keyword "create user").
 *  2. Multi-token: split the query on whitespace, score each token
 *     individually against the item's texts, take the min across tokens.
 *     This handles order-independent searches like "settings org" →
 *     "Organization settings", where neither token-as-phrase matches but
 *     each token finds a home in the label.
 *
 * Returns the max of the two passes. Multi-token only contributes when
 * every token finds a positive match — partial coverage is rejected so
 * "xyz settings" doesn't promote "Settings."
 */
export const scoreItemForBestMatch = (
  label: string,
  keywords: string[] | undefined,
  queryLower: string,
  tokens: string[]
): number => {
  const labelLower = label.toLowerCase();
  const keywordsLower = keywords ? keywords.map((k) => k.toLowerCase()) : [];

  // Pass 1: full-query single-text scoring.
  const fullQueryScore = scoreTokenAgainstItem(
    labelLower,
    keywordsLower,
    queryLower
  );
  if (fullQueryScore === SCORE_LABEL_EXACT) return fullQueryScore;

  // Pass 2: multi-token. Only meaningful when there's more than one
  // token — otherwise it produces the same answer as pass 1.
  let multiTokenScore = 0;
  if (tokens.length > 1) {
    let minPerToken = Infinity;
    for (let i = 0; i < tokens.length; i += 1) {
      const tokenScore = scoreTokenAgainstItem(
        labelLower,
        keywordsLower,
        tokens[i]
      );
      if (tokenScore === 0) {
        minPerToken = 0;
        break;
      }
      if (tokenScore < minPerToken) minPerToken = tokenScore;
    }
    multiTokenScore = minPerToken === Infinity ? 0 : minPerToken;
  }

  return Math.max(fullQueryScore, multiTokenScore);
};

/**
 * Compute the Best match entries for an items list and a search query.
 * Returns an empty array when the query is shorter than
 * BEST_MATCH_MIN_QUERY or nothing scores above the noise floor.
 *
 * Entries are sorted by score desc, then alphabetical by display label.
 * Sub-items rank on their own score independent of the parent — a strong
 * sub-item hit can promote even when the parent label/keywords don't
 * match.
 */
export interface IBestMatchEntry {
  item: ICommandItem;
  sub?: ICommandSubItem;
  score: number;
}

export const computeBestMatch = (
  items: ICommandItem[],
  query: string
): IBestMatchEntry[] => {
  const queryLower = query.toLowerCase().trim();
  // Gate on typed characters, not raw length — interior whitespace
  // shouldn't count toward the floor (otherwise "o s" reads as 3 chars
  // and skips the stricter 2-char ladder).
  const charCount = queryLower.replace(/\s+/g, "").length;
  if (charCount < BEST_MATCH_MIN_QUERY) return [];

  // At 2 chars, raise the noise floor to label-prefix (exact + prefix
  // only). Below that floor the result is too noisy to surface as Best
  // match. 3+ chars uses the full tier ladder.
  const minScore =
    charCount < BEST_MATCH_FULL_LADDER_MIN ? SCORE_LABEL_PREFIX : 1;

  // Split on whitespace only for tokenization — hyphens stay attached
  // because hyphenated phrases like "fleet-maintained" are intentional
  // single tokens in user input. (Hyphen-splitting happens inside the
  // text being matched against, via scoreMatch's WORD_SPLIT.)
  const tokens = queryLower.split(/\s+/).filter(Boolean);

  const entries: IBestMatchEntry[] = [];
  items.forEach((item) => {
    const itemScore = scoreItemForBestMatch(
      item.label,
      item.keywords,
      queryLower,
      tokens
    );
    if (itemScore >= minScore) entries.push({ item, score: itemScore });
    item.subItems?.forEach((sub) => {
      const subScore = scoreItemForBestMatch(
        sub.label,
        sub.keywords,
        queryLower,
        tokens
      );
      if (subScore >= minScore) entries.push({ item, sub, score: subScore });
    });
  });
  entries.sort((a, b) => {
    if (b.score !== a.score) return b.score - a.score;
    const aLabel = (a.sub ?? a.item).label;
    const bLabel = (b.sub ?? b.item).label;
    return aLabel.localeCompare(bLabel);
  });
  return entries;
};

/**
 * Highlight matched ranges of the query (or tokens) in a display text.
 * Returns a list of segments — `matched: true` segments are the parts
 * the user typed (or close enough) and should render with emphasis.
 *
 * The matching here is intentionally simpler than scoring: it finds raw
 * substring occurrences for the full query plus each token, merges
 * overlapping ranges, and walks the source text. We don't replay the
 * scoring tiers — the goal is "tell the user where you matched," not
 * "explain the tier."
 */
export interface IHighlightSegment {
  text: string;
  matched: boolean;
}

export const highlightMatches = (
  text: string,
  query: string
): IHighlightSegment[] => {
  if (!text) return [{ text: "", matched: false }];
  const queryLower = query.toLowerCase().trim();
  if (!queryLower) return [{ text, matched: false }];

  const textLower = text.toLowerCase();
  // Always include the full query as one needle so phrase matches get
  // a contiguous highlight, plus each individual token for multi-token
  // queries.
  const needles = new Set<string>([queryLower]);
  queryLower.split(/\s+/).forEach((tok) => {
    if (tok) needles.add(tok);
  });

  const ranges: Array<[number, number]> = [];
  needles.forEach((needle) => {
    let idx = textLower.indexOf(needle);
    while (idx !== -1) {
      ranges.push([idx, idx + needle.length]);
      idx = textLower.indexOf(needle, idx + needle.length);
    }
  });

  if (ranges.length === 0) return [{ text, matched: false }];

  ranges.sort((a, b) => a[0] - b[0]);
  const merged: Array<[number, number]> = [ranges[0]];
  for (let i = 1; i < ranges.length; i += 1) {
    const last = merged[merged.length - 1];
    if (ranges[i][0] <= last[1]) {
      last[1] = Math.max(last[1], ranges[i][1]);
    } else {
      merged.push(ranges[i]);
    }
  }

  const segments: IHighlightSegment[] = [];
  let cursor = 0;
  merged.forEach(([start, end]) => {
    if (start > cursor) {
      segments.push({ text: text.slice(cursor, start), matched: false });
    }
    segments.push({ text: text.slice(start, end), matched: true });
    cursor = end;
  });
  if (cursor < text.length) {
    segments.push({ text: text.slice(cursor), matched: false });
  }
  return segments;
};

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
