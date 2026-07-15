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
  group: typeof GROUPS[number];
  path?: string;
  keywords?: string[];
  /** Displayed on the right when navigating would switch your team context */
  teamName?: string;
  /** Nested items shown when the parent is expanded via chevron */
  subItems?: ICommandSubItem[];
  /** Custom action instead of navigation */
  onAction?: () => void;
  /** True when selecting this item opens a picker page (not a navigation). */
  opensPickerPage?: boolean;
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
  /** Global or any-team admin/maintainer. Gates the admin/maintainer-only
   *  Controls > OS settings sub-items (Certificates, Passwords, Host names),
   *  which technicians can't manage despite passing `canAccessControls`. */
  isAdminOrMaintainer?: boolean;
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
  // NFD-decompose, drop combining marks, lowercase. Mirrors
  // utf8mb4_unicode_ci's accent-insensitive folding so the highlighter
  // never undershoots a row the backend surfaced.
  const foldChar = (cp: string): string =>
    cp.normalize("NFD").replace(/\p{M}/gu, "").toLowerCase();
  const fold = (s: string): string => Array.from(s, foldChar).join("");
  const queryLower = fold(query).trim();
  if (!queryLower) return [{ text, matched: false }];

  // Always include the full query as one needle so phrase matches get
  // a contiguous highlight, plus each individual token for multi-token
  // queries.
  const needles = new Set<string>([queryLower]);
  queryLower.split(/\s+/).forEach((tok) => {
    if (tok) needles.add(tok);
  });

  // Build textLower in lockstep with offset maps back to the original.
  // Iterate by codepoint (not code unit) so supplementary-plane chars
  // fold correctly — text[i].toLowerCase() on a lone surrogate is a
  // no-op. Some folds change length (Turkish "İ" → "i", combining marks
  // stripped from "Café" → "Cafe"); lowerToOrigStart/End translate
  // matches back to original-text ranges.
  let textLower = "";
  const lowerToOrigStart: number[] = [];
  const lowerToOrigEnd: number[] = [];
  let origIdx = 0;
  Array.from(text).forEach((cp) => {
    const folded = foldChar(cp);
    for (let j = 0; j < folded.length; j += 1) {
      lowerToOrigStart.push(origIdx);
      lowerToOrigEnd.push(origIdx + cp.length);
    }
    textLower += folded;
    origIdx += cp.length;
  });
  // Sentinels for end-of-text alignment.
  lowerToOrigStart.push(text.length);
  lowerToOrigEnd.push(text.length);

  const ranges: Array<[number, number]> = [];
  needles.forEach((needle) => {
    let idx = textLower.indexOf(needle);
    while (idx !== -1) {
      const lowerEnd = idx + needle.length;
      // Translate to original-text coords. lowerToOrigEnd already
      // accounts for surrogate-pair widths and length-changing folds.
      const origStart = lowerToOrigStart[idx];
      const origEnd = lowerToOrigEnd[lowerEnd - 1];
      ranges.push([origStart, origEnd]);
      idx = textLower.indexOf(needle, lowerEnd);
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

// Per-page fleet picker behavior. One row per path prefix; each row
// declares overrides off the common defaults:
//
//   default all:        "native"  — page renders All fleets inline
//   default unassigned: "hidden"  — picker omits Unassigned
//
// Overrides:
//   all = "redirect" — keep in the picker as a shortcut, but selecting it
//                      sends the user to /hosts/manage (which supports All).
//                      Mirrors `useTeamIdParam({ includeAllTeams: false })`
//                      or, for SOFTWARE_LIBRARY, the explicit redirect in
//                      SoftwarePage when All is selected on the Library tab.
//   all = "hidden"   — omit from picker. Reserved for admin views where
//                      "all fleets" is conceptually meaningless (no global
//                      view of fleet-level user/option/settings config).
//   unassigned = "native" — page renders Unassigned (id 0) inline. Mirrors
//                      `useTeamIdParam({ includeNoTeam: true })`.
//
// Multiple rows may match a path; the longest-matching prefix wins per
// dimension (e.g. /software/library inherits unassigned:"native" from
// /software while overriding all to "redirect" via its own row).
//
// Scope: Premium, non-Primo only. The fleet picker doesn't exist on Fleet
// Free or in Primo mode — both entry points (Cmd+Shift+F and the
// fleet-switcher button) are gated behind `canSwitchFleet`, which requires
// `isPremiumTier && !isPrimoMode`. This table and its resolver describe
// the picker's behavior within that gated surface; do not consult them
// from Free or Primo code paths.

type AllOverride = "redirect" | "hidden";
type UnassignedOverride = "native";

interface PageFleetRule {
  prefix: string;
  all?: AllOverride;
  unassigned?: UnassignedOverride;
}

const PAGE_FLEET_RULES: ReadonlyArray<PageFleetRule> = [
  { prefix: paths.CONTROLS, all: "redirect", unassigned: "native" },
  { prefix: paths.SOFTWARE_LIBRARY, all: "redirect" },
  { prefix: paths.SOFTWARE, unassigned: "native" },
  { prefix: paths.NEW_REPORT, all: "redirect" },
  { prefix: `${paths.ROOT}hosts`, unassigned: "native" },
  { prefix: `${paths.ROOT}policies`, unassigned: "native" },
  { prefix: `${paths.ROOT}settings/fleets/users`, all: "hidden" },
  { prefix: `${paths.ROOT}settings/fleets/options`, all: "hidden" },
  { prefix: `${paths.ROOT}settings/fleets/settings`, all: "hidden" },
];

interface ResolvedFleetSupport {
  all: "native" | AllOverride;
  unassigned: "native" | "hidden";
}

const resolvePageFleetSupport = (pathname: string): ResolvedFleetSupport => {
  let all: ResolvedFleetSupport["all"] = "native";
  let unassigned: ResolvedFleetSupport["unassigned"] = "hidden";
  let allPrefixLen = -1;
  let unassignedPrefixLen = -1;
  PAGE_FLEET_RULES.forEach((rule) => {
    if (!pathname.startsWith(rule.prefix)) return;
    if (rule.all && rule.prefix.length > allPrefixLen) {
      all = rule.all;
      allPrefixLen = rule.prefix.length;
    }
    if (rule.unassigned && rule.prefix.length > unassignedPrefixLen) {
      unassigned = rule.unassigned;
      unassignedPrefixLen = rule.prefix.length;
    }
  });
  return { all, unassigned };
};

export const pathSupportsUnassigned = (pathname: string): boolean =>
  resolvePageFleetSupport(pathname).unassigned === "native";

export const pathSupportsAllFleets = (pathname: string): boolean =>
  resolvePageFleetSupport(pathname).all !== "hidden";

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
  const support = resolvePageFleetSupport(pathname);

  // All fleets: page can't render it but the picker keeps the option as a
  // shortcut. Redirect to Hosts (which supports All).
  if (isAll && support.all === "redirect") {
    return paths.MANAGE_HOSTS;
  }
  // Unassigned: page can't render it. Keep fleet_id=0 on the fallback URL
  // — useTeamIdParam coerces a missing param back to All fleets (-1),
  // which would silently undo the switch. This branch is defensive; the
  // picker already filters unsupported Unassigned via pathSupportsUnassigned.
  if (isUnassignedTarget && support.unassigned !== "native") {
    return `${paths.MANAGE_HOSTS}?fleet_id=0`;
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
