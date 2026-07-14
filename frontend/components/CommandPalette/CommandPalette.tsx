import React, {
  useContext,
  useEffect,
  useLayoutEffect,
  useMemo,
  useState,
  useCallback,
  useRef,
} from "react";
import { Command, useCommandState } from "cmdk";
import {
  Title as DialogTitle,
  Description as DialogDescription,
} from "@radix-ui/react-dialog";
import { browserHistory } from "react-router";

import { AppContext } from "context/app";
import {
  APP_CONTEXT_ALL_TEAMS_ID,
  APP_CONTEXT_NO_TEAM_ID,
} from "interfaces/team";
import Icon from "components/Icon";
import { isDarkMode, setThemeMode } from "utilities/theme";
import paths from "router/paths";

import {
  ICommandItem,
  ICommandSubItem,
  GROUPS,
  buildPaletteItems,
  buildFleetSwitchUrl,
  computeBestMatch,
  pathSupportsAllFleets,
  pathSupportsUnassigned,
} from "./helpers";
import FleetPicker from "./components/FleetPicker";
import HostPicker from "./components/HostPicker";
import SoftwarePicker from "./components/SoftwarePicker";
import ReportPicker from "./components/ReportPicker";
import PolicyPicker from "./components/PolicyPicker";
import HighlightedLabel from "./components/HighlightedLabel";
import UprightEmoji from "./components/UprightEmoji";
import { isPreFilteredResult } from "./components/constants";

const baseClass = "command-palette";

// Subscribes to cmdk's selected value via its internal store. We can't
// rely on `Command.Dialog`'s `onValueChange` prop for this: in cmdk@1.1.1
// that callback only fires when `value` is also controlled, and the
// palette runs cmdk uncontrolled so arrow-key highlight changes never
// reach React without this bridge. Rendered inside `Command.Dialog` so
// the `useCommandState` context is available.
const HighlightSubscriber = ({
  onChange,
}: {
  onChange: (value: string) => void;
}) => {
  const value = useCommandState((state) => state.value);
  useEffect(() => {
    onChange(value ?? "");
  }, [value, onChange]);
  return null;
};

// cmdk treats `value` as the row's identity *and* as the substring it
// filters against. Two rows with the same value collide. Including the
// item's id guarantees uniqueness without losing the user-typeable
// label/keyword content cmdk needs for its filter pass.
//
// Hoisted to module scope so callers can use these from memoization
// safely (stable identity across renders).
const getUniqueItemValue = (item: ICommandItem) => {
  const parts = [item.id, item.label, ...(item.keywords ?? [])];
  item.subItems?.forEach((sub) => {
    parts.push(sub.label, ...(sub.keywords ?? []));
  });
  return parts.join(" ");
};

const getUniqueSubItemValue = (sub: ICommandSubItem) => {
  return [sub.id, sub.label, ...(sub.keywords ?? [])].join(" ");
};

type Page =
  | "root"
  | "switch-fleet"
  | "view-host"
  | "view-software"
  | "view-software-library"
  | "view-report"
  | "view-policy";

const CommandPalette = (): JSX.Element | null => {
  const [open, setOpen] = useState(false);
  const [page, setPage] = useState<Page>("root");
  const [search, setSearch] = useState("");
  const [expandedItems, setExpandedItems] = useState<Set<string>>(new Set());
  const inputRef = useRef<HTMLInputElement>(null);

  const {
    availableTeams,
    currentTeam,
    setCurrentTeam,
    isGlobalAdmin,
    isGlobalMaintainer,
    isAnyTeamAdmin,
    isAnyTeamMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
    isGlobalTechnician,
    isAnyTeamTechnician,
    isObserverPlus,
    isAnyTeamObserverPlus,
    isGlobalObserver,
    isTeamObserver,
    isPremiumTier,
    isMacMdmEnabledAndConfigured,
    isWindowsMdmEnabledAndConfigured,
    isAndroidMdmEnabledAndConfigured,
    isVppEnabled,
    config,
    isNoAccess,
  } = useContext(AppContext);

  const isTechnician = isGlobalTechnician || isAnyTeamTechnician;

  const canAccessControls =
    isGlobalAdmin ||
    isGlobalMaintainer ||
    isAnyTeamAdmin ||
    isAnyTeamMaintainer ||
    isTechnician;

  const canWrite =
    isGlobalAdmin ||
    isGlobalMaintainer ||
    isAnyTeamAdmin ||
    isAnyTeamMaintainer ||
    isTechnician;

  // Custom variables are admin-tier global config (mirrors Variables.tsx
  // `canEdit`). Team admins/maintainers/technicians lack the role even
  // though they have `canWrite`, so the destination page would render
  // a read-only view — gate the palette entry accordingly.
  const canEditCustomVariable = !!isGlobalAdmin || !!isGlobalMaintainer;

  // Mirrors SoftwarePage.tsx canAddSoftware. Note isTeamAdmin /
  // isTeamMaintainer here are scoped to currentTeam by AppContext — a
  // user who is admin of Team A but observer of Team B (currently
  // selected) correctly evaluates to false. canWrite would have
  // accepted them via isAnyTeamAdmin.
  const canAddSoftware =
    !!isGlobalAdmin ||
    !!isGlobalMaintainer ||
    !!isTeamAdmin ||
    !!isTeamMaintainer;

  // Admin/maintainer-only Controls sub-items (Certificates, Passwords, Host
  // names). Technicians can reach Controls (canAccessControls) but not these,
  // so gate them on the positive admin/maintainer role rather than
  // `!isTechnician`.
  const isAdminOrMaintainer =
    isGlobalAdmin ||
    isGlobalMaintainer ||
    isAnyTeamAdmin ||
    isAnyTeamMaintainer;

  // Observer+ users can run live queries even though they can't write.
  const canRunLiveReport =
    canWrite || !!isObserverPlus || !!isAnyTeamObserverPlus;

  // Used by ReportPicker to decide whether to render the "Observers can
  // run" affordance on a report — that hint is meant for non-observers
  // (it advertises which reports they can hand off), so suppress for
  // observers viewing their own scope.
  const isViewerObserverInScope = !!isGlobalObserver || !!isTeamObserver;

  // Primo Mode is a single-fleet premium installation. The fleet switcher
  // should be hidden, fleet creation disabled, and All-fleets-only commands
  // need to surface for the user's single fleet.
  const isPrimoMode = !!config?.partnerships?.enable_primo;

  // Track theme as reactive state so the toggle-dark-mode item's label
  // updates if the theme flips externally (system theme media query,
  // another tab, sibling component). utilities/theme dispatches a
  // `fleet-theme-change` window event on every change.
  const [isDarkModeActive, setIsDarkModeActive] = useState(isDarkMode);
  useEffect(() => {
    const onThemeChange = (e: Event) => {
      const detail = (e as CustomEvent<{ dark: boolean }>).detail;
      setIsDarkModeActive(!!detail?.dark);
    };
    window.addEventListener("fleet-theme-change", onThemeChange);
    return () =>
      window.removeEventListener("fleet-theme-change", onThemeChange);
  }, []);

  // Policy automations: mirrors ManagePoliciesPage.canEditAutomationsSettings.
  // Maintainers can add/delete policies but the in-page Automations button
  // is hidden for them, and the deep-link useEffect re-checks the same
  // gate — so the palette item must match, otherwise it's a dead link for
  // maintainers. isTeamAdmin is scoped to currentTeam by AppContext, so a
  // user who's team admin of A but viewing B correctly won't see this.
  const canManagePolicyAutomations = isGlobalAdmin || isTeamAdmin;

  // Software automations require global admin (all fleets view)
  const canManageSoftwareAutomations = isGlobalAdmin;

  // Report automations: mirror ManageQueriesPage's `canManageAutomations`
  // exactly (isGlobalAdmin || isTeamAdmin) — the palette item points at
  // the current team via withTeamId, so the destination's gate evaluates
  // against the same team. Using isAnyTeamAdmin here would surface the
  // command for users who are admin of *some* team but observer of the
  // current one — they'd land on Reports and see no button.
  const canManageReportAutomations = isGlobalAdmin || isTeamAdmin;

  const canAccessSettings = isGlobalAdmin;

  // Whether a specific team is selected (not "All teams")
  const hasTeamSelected = currentTeam && currentTeam.id > 0;
  const isUnassigned = currentTeam?.id === 0;

  // Append fleet_id to a path so navigation preserves the current team context.
  // Includes Unassigned (id 0) so navigation doesn't drop the no-team context.
  const withTeamId = useCallback(
    (path: string) => {
      if (!hasTeamSelected && !isUnassigned) {
        return path;
      }
      const separator = path.includes("?") ? "&" : "?";
      return `${path}${separator}fleet_id=${currentTeam?.id}`;
    },
    [hasTeamSelected, isUnassigned, currentTeam?.id]
  );

  // Reset page and search when dialog opens/closes
  useEffect(() => {
    if (!open) {
      setPage("root");
      setSearch("");
      setExpandedItems(new Set());
    }
  }, [open]);

  const canSwitchFleet =
    isPremiumTier &&
    !isPrimoMode &&
    !!availableTeams &&
    availableTeams.length > 1;

  // Display label for the fleet switcher button — falls back to the
  // "All fleets" sentinel when no specific team is selected. Used in
  // both the visible button text and the aria-label.
  const fleetSwitcherLabel = currentTeam?.name || "All fleets";

  // Detect macOS so we can render the Cmd glyph (⌘) vs. "Ctrl" inline on
  // the fleet-switcher shortcut. navigator.platform is deprecated but
  // still the most reliable cross-browser signal for this binary check.
  const isMacPlatform =
    typeof navigator !== "undefined" &&
    /Mac|iPhone|iPad|iPod/i.test(navigator.platform);

  const pickerPagePlaceholders: Partial<Record<Page, string>> = {
    "switch-fleet": "Search a fleet...",
    "view-host": "Search hosts...",
    "view-software": "Search software inventory...",
    "view-software-library": "Search software library...",
    "view-report": "Search reports...",
    "view-policy": "Search policies...",
  };
  const pickerPagePlaceholder = pickerPagePlaceholders[page];

  // Toggle open on Cmd+K / Ctrl+K; jump to switch-fleet on Cmd+Shift+F.
  // Focus is handled by the [open, page] effect below — don't rAF here, the
  // input ref isn't set until Radix's portal mounts.
  // Skip registration entirely for no-access users so we don't intercept
  // (and preventDefault) keyboard shortcuts for a palette they can't open.
  useEffect(() => {
    if (isNoAccess) return undefined;
    const onKeyDown = (e: KeyboardEvent) => {
      // Require the platform-native modifier: Cmd on macOS, Ctrl elsewhere.
      // Accepting either on both platforms hijacks native shortcuts —
      // e.g. Ctrl+K readline kill-line in text fields on macOS, and
      // Ctrl+Shift+F (find in files / system shortcut) on macOS.
      const correctModifier = isMacPlatform ? e.metaKey : e.ctrlKey;
      if (!correctModifier) return;
      // Normalize once so Caps Lock (or shift layouts) don't miss.
      const key = e.key.toLowerCase();
      if (key === "k") {
        e.preventDefault();
        setOpen((prev) => !prev);
      } else if (e.shiftKey && key === "f" && canSwitchFleet) {
        e.preventDefault();
        setOpen(true);
        setSearch("");
        setPage("switch-fleet");
      }
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [canSwitchFleet, isNoAccess, isMacPlatform]);

  const navigate = useCallback((path: string) => {
    setOpen(false);
    browserHistory.push(path);
  }, []);

  const goToPage = useCallback((newPage: Page) => {
    setSearch("");
    setPage(newPage);
  }, []);

  const goBack = useCallback(() => {
    setSearch("");
    setPage("root");
  }, []);

  // Focus the input whenever the dialog is open or the page changes. Runs
  // after the portal has mounted the input, unlike rAF in event handlers
  // (which fires before Radix's first commit when opening from closed).
  useEffect(() => {
    if (open) {
      inputRef.current?.focus();
    }
  }, [open, page]);

  // Backspace on empty input returns to root from a picker page.
  const onKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (page !== "root" && e.key === "Backspace" && !search) {
        e.preventDefault();
        goBack();
      }
    },
    [page, search, goBack]
  );

  // Intercept Escape on a picker page so it returns to root instead of
  // closing the dialog. cmdk 1.1.1's Command.Dialog doesn't forward
  // `onEscapeKeyDown` to Radix's Dialog.Content, so we can't override
  // the close intent via props.
  //
  // Approach: a capture-phase document listener that calls
  // `stopImmediatePropagation` on Escape from a picker page. This prevents
  // both Radix's DismissableLayer ESC handler AND any sibling listeners
  // from firing on this event — the dialog never learns about the press,
  // so it doesn't close. `useLayoutEffect` is intentional: it attaches
  // before any `useEffect` in deeper Radix components, guaranteeing
  // priority in the capture phase regardless of mount order. Click-
  // outside still closes the palette outright via onOpenChange.
  const pageRef = useRef(page);
  pageRef.current = page;

  useLayoutEffect(() => {
    if (!open || isNoAccess) {
      return undefined;
    }
    const onDocKey = (e: KeyboardEvent) => {
      if (e.key === "Escape" && pageRef.current !== "root") {
        e.preventDefault();
        e.stopImmediatePropagation();
        goBack();
      }
    };
    document.addEventListener("keydown", onDocKey, true);
    return () => document.removeEventListener("keydown", onDocKey, true);
  }, [open, goBack, isNoAccess]);

  const handleOpenChange = useCallback((nextOpen: boolean) => {
    setOpen(nextOpen);
  }, []);

  const handleSwitchFleet = useCallback(
    (fleetId: number) => {
      const selected = availableTeams?.find((t) => t.id === fleetId);
      if (selected) {
        setCurrentTeam(selected);
      }

      const { pathname, search: currentSearch } = window.location;
      browserHistory.push(
        buildFleetSwitchUrl({ pathname, currentSearch, fleetId })
      );

      // Return to root so the palette stays open on the main view.
      setSearch("");
      setPage("root");
    },
    [availableTeams, setCurrentTeam]
  );

  const toggleExpanded = useCallback((id: string) => {
    setExpandedItems((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  // Memoize the item array on the values buildPaletteItems actually
  // consumes. Inline onToggle/onView callbacks are intentionally excluded
  // from deps — they only call already-stable setters/useCallback'd
  // goToPage, so they don't change semantically across renders.
  const items = useMemo(
    () =>
      buildPaletteItems({
        search,
        currentTeam,
        availableTeams,
        config,
        canAccessControls,
        canWrite,
        canRunLiveReport,
        canAccessSettings,
        canManagePolicyAutomations,
        canManageSoftwareAutomations,
        canManageReportAutomations,
        canEditCustomVariable,
        canAddSoftware,
        isAdminOrMaintainer,
        isTechnician,
        isPremiumTier,
        isPrimoMode,
        isDarkMode: isDarkModeActive,
        isMacMdmEnabledAndConfigured,
        isWindowsMdmEnabledAndConfigured,
        isAndroidMdmEnabledAndConfigured,
        isVppEnabled,
        hasTeamSelected,
        withTeamId,
        onToggleDarkMode: () => {
          setThemeMode(isDarkModeActive ? "light" : "dark");
          setOpen(false);
        },
        onViewHost: () => goToPage("view-host"),
        onViewSoftware: () => goToPage("view-software"),
        onViewSoftwareLibrary: () => goToPage("view-software-library"),
        onViewReport: () => goToPage("view-report"),
        onViewPolicy: () => goToPage("view-policy"),
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [
      search,
      currentTeam,
      availableTeams,
      config,
      canAccessControls,
      canWrite,
      canRunLiveReport,
      canAccessSettings,
      canManagePolicyAutomations,
      canManageSoftwareAutomations,
      canManageReportAutomations,
      canEditCustomVariable,
      canAddSoftware,
      isAdminOrMaintainer,
      isTechnician,
      isPremiumTier,
      isPrimoMode,
      isDarkModeActive,
      isMacMdmEnabledAndConfigured,
      isWindowsMdmEnabledAndConfigured,
      isAndroidMdmEnabledAndConfigured,
      isVppEnabled,
      hasTeamSelected,
      withTeamId,
      goToPage,
    ]
  );

  const groupedItems = useMemo(
    () =>
      items.reduce<Record<string, ICommandItem[]>>((acc, item) => {
        if (!acc[item.group]) {
          acc[item.group] = [];
        }
        acc[item.group].push(item);
        return acc;
      }, {}),
    [items]
  );

  const isSearching = search.length > 0;
  const searchLower = search.toLowerCase().trim();

  // Map cmdk values (normalized) to parent item IDs for auto-expand on keyboard nav
  const valueToParentId = useMemo(() => {
    const map = new Map<string, string>();
    items.forEach((item) => {
      if (item.subItems?.length) {
        map.set(getUniqueItemValue(item).toLowerCase().trim(), item.id);
        item.subItems.forEach((sub) => {
          map.set(getUniqueSubItemValue(sub).toLowerCase().trim(), item.id);
        });
      }
    });
    return map;
  }, [items]);

  // Track whether the most recent input was a keystroke or pointer move.
  // Hovering changes cmdk's selected value (selection-follows-pointer),
  // which would otherwise pop sub-items open every time the mouse passes
  // a parent row. We still want auto-expand on arrow-key navigation, so
  // gate it on this ref instead of turning off pointer selection
  // entirely (which would also kill the hover highlight).
  const lastInputSourceRef = useRef<"keyboard" | "pointer">("keyboard");
  useEffect(() => {
    if (!open) return undefined;
    const onKey = () => {
      lastInputSourceRef.current = "keyboard";
    };
    const onPointer = () => {
      lastInputSourceRef.current = "pointer";
    };
    document.addEventListener("keydown", onKey, true);
    document.addEventListener("pointermove", onPointer, true);
    return () => {
      document.removeEventListener("keydown", onKey, true);
      document.removeEventListener("pointermove", onPointer, true);
    };
  }, [open]);

  // Auto expand/collapse sub-items as the user arrows through items.
  // Normalize to match how valueToParentId stores its keys — cmdk
  // hands back the raw `value` prop (preserves casing/whitespace).
  const handleHighlightChange = useCallback(
    (value: string) => {
      if (isSearching) return;
      if (lastInputSourceRef.current !== "keyboard") return;
      const parentId = valueToParentId.get(value.toLowerCase().trim());
      // Bail out when the target set is equivalent to the current one
      // (arrowing within the same parent's sub-items, or moving between
      // two non-parent rows). Without this, every arrow press allocates
      // a new Set and forces a re-render even when no expansion state
      // actually changes.
      setExpandedItems((prev) => {
        if (parentId) {
          if (prev.size === 1 && prev.has(parentId)) return prev;
          return new Set([parentId]);
        }
        if (prev.size === 0) return prev;
        return new Set();
      });
    },
    [valueToParentId, isSearching]
  );

  // Find Best matches — items and sub-items scored by how strongly their
  // label or keywords match the query. Implementation is in helpers.ts
  // (so the scoring tiers are unit-testable independent of cmdk).
  const bestMatchItems = useMemo(() => computeBestMatch(items, searchLower), [
    items,
    searchLower,
  ]);

  // Set of IDs already shown in Best match — used to dedupe from regular
  // groups and inside expanded sub-item lists.
  const bestMatchIds = useMemo(() => {
    const ids = new Set<string>();
    bestMatchItems.forEach(({ item, sub }) => ids.add((sub ?? item).id));
    return ids;
  }, [bestMatchItems]);

  const renderItem = (item: ICommandItem) => {
    const isExpanded = expandedItems.has(item.id);
    // Sub-items promoted into Best match are hidden here so they don't
    // render twice. The parent stays visible because its own promotion
    // is handled one level up (in renderRootPage, via groupedItems
    // filtering).
    const visibleSubItems = item.subItems?.filter(
      (sub) => !bestMatchIds.has(sub.id)
    );
    const hasSubItems = !!visibleSubItems && visibleSubItems.length > 0;

    return (
      <React.Fragment key={item.id}>
        <Command.Item
          value={getUniqueItemValue(item)}
          onSelect={() => {
            if (item.onAction) {
              item.onAction();
              return;
            }
            if (item.path) navigate(item.path);
          }}
          className={`${baseClass}__item`}
        >
          <div className={`${baseClass}__item-left`}>
            <span className={`${baseClass}__item-label`}>{item.label}</span>
            {hasSubItems && !isSearching && (
              <button
                type="button"
                tabIndex={-1}
                className={`${baseClass}__item-more ${
                  isExpanded ? `${baseClass}__item-more--expanded` : ""
                }`}
                onClick={(e) => {
                  e.stopPropagation();
                  toggleExpanded(item.id);
                }}
                onPointerDown={(e) => e.preventDefault()}
              >
                <Icon
                  name="chevron-down"
                  size="small"
                  color="ui-fleet-black-50"
                />
              </button>
            )}
            {item.opensPickerPage && (
              <span aria-hidden className={`${baseClass}__item-more`}>
                <Icon
                  name="chevron-right"
                  size="small"
                  color="ui-fleet-black-50"
                />
              </span>
            )}
          </div>
          {item.teamName && (
            <span className={`${baseClass}__item-fleet`}>
              <UprightEmoji text={item.teamName} />
            </span>
          )}
        </Command.Item>
        {/* Render sub-items when expanded (browsing) or always when searching */}
        {hasSubItems &&
          (isExpanded || isSearching) &&
          visibleSubItems &&
          visibleSubItems.map((sub) => (
            <Command.Item
              key={sub.id}
              value={getUniqueSubItemValue(sub)}
              onSelect={() => navigate(sub.path)}
              className={`${baseClass}__item ${baseClass}__item--sub`}
            >
              <span className={`${baseClass}__item-label`}>{sub.label}</span>
            </Command.Item>
          ))}
      </React.Fragment>
    );
  };

  const renderRootPage = () => (
    <>
      {/* Top-ranked items render without a heading — visually they're
          just the most relevant matches, separated from the rest by a
          rule. The BEST_MATCH prefix on the value is a cmdk filter-bypass
          token (see filter prop) — it forces the item past substring
          filtering regardless of score. */}
      {bestMatchItems.length > 0 && (
        <>
          <Command.Group className={`${baseClass}__group`}>
            {bestMatchItems.map(({ item, sub }) => {
              const target = sub || item;
              // Sub-items get the same indented styling they have in
              // their regular group, so users can tell at a glance which
              // results are nested.
              const itemClass = sub
                ? `${baseClass}__item ${baseClass}__item--sub`
                : `${baseClass}__item`;
              return (
                <Command.Item
                  key={`best-${target.id}`}
                  // cmdk treats value as the row's identity. Include
                  // target.id so two promoted items with the same label
                  // (e.g., the "Users" Settings page and the "Users"
                  // setup-experience sub-item) don't collide.
                  value={`BEST_MATCH ${target.id} ${target.label}`}
                  onSelect={() => {
                    // Sub-items navigate to their own path — never route
                    // through the parent's onAction. Otherwise a future
                    // sub-item under an action-backed parent (e.g., a
                    // child of "View host") would open the parent flow
                    // instead of the sub-item destination.
                    if (sub) {
                      navigate(sub.path);
                      return;
                    }
                    if (item.onAction) {
                      item.onAction();
                      return;
                    }
                    if (item.path) navigate(item.path);
                  }}
                  className={itemClass}
                >
                  <div className={`${baseClass}__item-left`}>
                    <span className={`${baseClass}__item-label`}>
                      <HighlightedLabel text={target.label} query={search} />
                    </span>
                    {/* Render the picker-page chevron for items that open
                        a picker (View host, View software, etc.). The
                        chevron only belongs to parent items — sub-items
                        navigate directly. */}
                    {!sub && item.opensPickerPage && (
                      <span aria-hidden className={`${baseClass}__item-more`}>
                        <Icon
                          name="chevron-right"
                          size="small"
                          color="ui-fleet-black-50"
                        />
                      </span>
                    )}
                  </div>
                  {/* Team-context chip on items whose navigation switches
                      the user's current fleet (e.g., add-report on All
                      fleets shows "All fleets"). */}
                  {!sub && item.teamName && (
                    <span className={`${baseClass}__item-fleet`}>
                      <UprightEmoji text={item.teamName} />
                    </span>
                  )}
                  {/* Parent label as a context chip on promoted sub-items
                      — disambiguates rows that share a label (e.g., two
                      "Run script" sub-items, one under Setup experience
                      and one under Manage policy automations). */}
                  {sub && (
                    <span className={`${baseClass}__item-meta`}>
                      {item.label}
                    </span>
                  )}
                </Command.Item>
              );
            })}
          </Command.Group>
          <Command.Separator className={`${baseClass}__separator`} />
        </>
      )}
      {GROUPS.map((group) => {
        const groupItems = groupedItems[group];
        if (!groupItems?.length) {
          return null;
        }
        // Drop items already shown in Best match so the user doesn't see
        // the same row twice. Items with only a sub-item promoted stay —
        // renderItem hides the promoted sub-item inline.
        const visibleGroupItems = groupItems.filter(
          (item) => !bestMatchIds.has(item.id)
        );
        if (!visibleGroupItems.length) {
          return null;
        }
        return (
          <Command.Group
            key={group}
            heading={group}
            className={`${baseClass}__group`}
          >
            {visibleGroupItems.map(renderItem)}
          </Command.Group>
        );
      })}
    </>
  );

  const handleSelectHost = useCallback((hostId: number) => {
    // Match ManageHostsPage.handleRowSelect — navigate without fleet_id so
    // we don't switch the user's current team context. The host details
    // page reads the host's team from the host record itself.
    setOpen(false);
    browserHistory.push(paths.HOST_DETAILS(hostId));
  }, []);

  const handleSelectSoftware = useCallback(
    (softwareId: number) => {
      setOpen(false);
      const basePath = paths.SOFTWARE_TITLE_DETAILS(String(softwareId));
      // Software titles are team-scoped; the destination page reads fleet_id
      // from the URL. Pass the user's current team so we don't drop them on
      // "All fleets" view by accident. Unassigned (id 0) is preserved too.
      if (currentTeam && currentTeam.id !== APP_CONTEXT_ALL_TEAMS_ID) {
        browserHistory.push(`${basePath}?fleet_id=${currentTeam.id}`);
      } else {
        browserHistory.push(basePath);
      }
    },
    [currentTeam]
  );

  const handleSelectReport = useCallback(
    (reportId: number) => {
      setOpen(false);
      const basePath = paths.REPORT_DETAILS(reportId);
      if (currentTeam && currentTeam.id !== APP_CONTEXT_ALL_TEAMS_ID) {
        browserHistory.push(`${basePath}?fleet_id=${currentTeam.id}`);
      } else {
        browserHistory.push(basePath);
      }
    },
    [currentTeam]
  );

  const handleSelectPolicy = useCallback(
    (policyId: number) => {
      setOpen(false);
      const basePath = paths.POLICY_DETAILS(policyId);
      if (currentTeam && currentTeam.id !== APP_CONTEXT_ALL_TEAMS_ID) {
        browserHistory.push(`${basePath}?fleet_id=${currentTeam.id}`);
      } else {
        browserHistory.push(basePath);
      }
    },
    [currentTeam]
  );

  if (isNoAccess) {
    return null;
  }

  return (
    <Command.Dialog
      open={open}
      onOpenChange={handleOpenChange}
      label="Command palette"
      className={baseClass}
      overlayClassName={`${baseClass}__overlay`}
      contentClassName={`${baseClass}__content`}
      filter={(value, searchTerm) => {
        // Best match items bypass cmdk's substring filter — they're already
        // scored against the query, and we want them shown regardless.
        if (value.startsWith("BEST_MATCH ")) {
          return 1;
        }
        // Picker results are pre-filtered by the server; show everything.
        if (isPreFilteredResult(value)) {
          return 1;
        }
        // Default cmdk filtering
        if (value.toLowerCase().includes(searchTerm.toLowerCase())) {
          return 1;
        }
        return 0;
      }}
    >
      <HighlightSubscriber onChange={handleHighlightChange} />
      {/* cmdk's Dialog wraps Radix Dialog.Content, which requires a Title and
          a Description for screen reader accessibility — without these, Radix
          logs a console error/warning on every open. Both are rendered
          visually hidden so the palette UI stays unchanged. */}
      <DialogTitle className="sr-only">Command palette</DialogTitle>
      <DialogDescription className="sr-only">
        Search for a page, command, or resource across Fleet.
      </DialogDescription>
      <div className={`${baseClass}__input-wrapper`}>
        {page !== "root" && (
          // tabIndex=-1 so Radix's open-autofocus skips the back button
          // and lands on Command.Input — Backspace and Escape already
          // cover keyboard "go back" so we lose nothing.
          <button
            type="button"
            aria-label="Back"
            tabIndex={-1}
            className={`${baseClass}__back-button`}
            onClick={goBack}
          >
            <Icon name="arrow-left" color="ui-fleet-black-75" />
          </button>
        )}
        <Command.Input
          ref={inputRef}
          className={`${baseClass}__input`}
          placeholder={
            pickerPagePlaceholder ?? "Search for a page or command..."
          }
          value={search}
          onValueChange={setSearch}
          onKeyDown={onKeyDown}
        />
        {page === "root" && canSwitchFleet && (
          <button
            type="button"
            // Locks the accessible name to the team name so the kbd
            // shortcut pills (aria-hidden) can't pollute it later if
            // the markup changes.
            aria-label={`Switch fleet (currently ${fleetSwitcherLabel})`}
            className={`${baseClass}__fleet-switcher`}
            onClick={() => goToPage("switch-fleet")}
            onKeyDown={(e) => {
              // Stop Enter from bubbling to cmdk-root, which would
              // activate whichever list item is currently highlighted.
              // The button's native Enter still triggers the click above.
              if (e.key === "Enter") {
                e.stopPropagation();
              }
            }}
          >
            <span className={`${baseClass}__fleet-switcher-label`}>
              {fleetSwitcherLabel}
            </span>
            <span
              aria-hidden
              className={`${baseClass}__fleet-switcher-shortcut`}
            >
              <kbd className={`${baseClass}__shortcut-key`}>
                {isMacPlatform ? "⌘" : "Ctrl"}
              </kbd>
              <kbd className={`${baseClass}__shortcut-key`}>⇧</kbd>
              <kbd className={`${baseClass}__shortcut-key`}>F</kbd>
            </span>
          </button>
        )}
        {page !== "root" && <kbd className={`${baseClass}__esc-hint`}>ESC</kbd>}
      </div>
      {/* Announce picker-page transitions to screen readers — the placeholder
          text changes but isn't reliably announced on its own. Strip the
          trailing ellipsis so the announcement isn't verbalized as
          "dot dot dot" by some screen readers. */}
      <div role="status" aria-live="polite" className="sr-only">
        {page === "root"
          ? ""
          : pickerPagePlaceholder?.replace(/\.{3}$/, "") ?? ""}
      </div>
      <Command.List className={`${baseClass}__list`}>
        {/* Picker pages render their own contextual empty state, so only show
            cmdk's generic Empty on the root page. */}
        {page === "root" && (
          <Command.Empty className={`${baseClass}__empty`}>
            No results found.
          </Command.Empty>
        )}
        {page === "root" && renderRootPage()}
        {page === "switch-fleet" && (
          <FleetPicker
            // Drop "All fleets" and "Unassigned" on pages whose useTeamIdParam
            // config rejects them (e.g. Dashboard hides Unassigned; the Fleet
            // → Users/Options/Settings admin pages hide All). Otherwise the
            // option appears valid but selecting it triggers a redirect-to-
            // default and silently reverts. Read pathname at render — the
            // palette can't be navigated away from without closing, so the
            // value is stable per session.
            availableTeams={availableTeams?.filter((t) => {
              if (
                t.id === APP_CONTEXT_NO_TEAM_ID &&
                !pathSupportsUnassigned(window.location.pathname)
              ) {
                return false;
              }
              if (
                t.id === APP_CONTEXT_ALL_TEAMS_ID &&
                !pathSupportsAllFleets(window.location.pathname)
              ) {
                return false;
              }
              return true;
            })}
            currentTeam={currentTeam}
            search={search}
            onSelect={handleSwitchFleet}
          />
        )}
        {page === "view-host" && (
          <HostPicker
            search={search}
            showTeamColumn={!!isPremiumTier && !isPrimoMode}
            onSelect={handleSelectHost}
          />
        )}
        {page === "view-software" && (
          <SoftwarePicker
            search={search}
            currentTeam={currentTeam}
            onSelect={handleSelectSoftware}
          />
        )}
        {page === "view-software-library" && (
          <SoftwarePicker
            search={search}
            currentTeam={currentTeam}
            scope="library"
            onSelect={handleSelectSoftware}
          />
        )}
        {page === "view-report" && (
          <ReportPicker
            search={search}
            currentTeam={currentTeam}
            isViewerObserver={isViewerObserverInScope}
            onSelect={handleSelectReport}
          />
        )}
        {page === "view-policy" && (
          <PolicyPicker
            search={search}
            currentTeam={currentTeam}
            isPremiumTier={!!isPremiumTier}
            onSelect={handleSelectPolicy}
          />
        )}
      </Command.List>
    </Command.Dialog>
  );
};

export default CommandPalette;
