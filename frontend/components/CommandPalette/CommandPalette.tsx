import React, {
  useContext,
  useEffect,
  useLayoutEffect,
  useMemo,
  useState,
  useCallback,
  useRef,
} from "react";
import { Command } from "cmdk";
import { browserHistory } from "react-router";

import { AppContext } from "context/app";
import { APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";
import Icon from "components/Icon";
import { isDarkMode, setThemeMode } from "utilities/theme";
import paths from "router/paths";

import {
  ICommandItem,
  ICommandSubItem,
  GROUPS,
  buildPaletteItems,
  buildFleetSwitchUrl,
} from "./helpers";
import FleetPicker from "./components/FleetPicker";
import HostPicker from "./components/HostPicker";
import SoftwarePicker from "./components/SoftwarePicker";
import ReportPicker from "./components/ReportPicker";
import PolicyPicker from "./components/PolicyPicker";
import { isPreFilteredResult } from "./components/constants";

const baseClass = "command-palette";

// Pure helper hoisted to module scope so it's stable across renders and
// can be safely called from inside memoization.
const getItemValue = (item: ICommandItem) => {
  const parts = [item.label, ...(item.keywords ?? [])];
  item.subItems?.forEach((sub) => {
    parts.push(sub.label, ...(sub.keywords ?? []));
  });
  return parts.join(" ");
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

  // Policy automations: same as canAddOrDeletePolicies in ManagePoliciesPage
  const canManagePolicyAutomations =
    isGlobalAdmin ||
    isGlobalMaintainer ||
    isAnyTeamAdmin ||
    isAnyTeamMaintainer;

  // Software automations require global admin (all fleets view)
  const canManageSoftwareAutomations = isGlobalAdmin;

  // Report automations: mirrors ManageQueriesPage's `canManageAutomations`
  // (isGlobalAdmin || isTeamAdmin). At the palette level we don't know
  // which team the user will land on, so admit anyone admin somewhere.
  const canManageReportAutomations = isGlobalAdmin || isAnyTeamAdmin;

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

  // For team-required commands (add hosts, manage enroll secrets): on
  // "All fleets" the destination chip says "Unassigned", so the URL
  // must actually route there with fleet_id=0 — otherwise the modal
  // would use global enroll secrets instead of the Unassigned team's.
  // Free has no team concept, so emit the bare path; useTeamIdParam
  // would otherwise redirect-strip fleet_id on Free anyway.
  const withTeamRequired = useCallback(
    (path: string) => {
      if (!isPremiumTier) return path;
      const targetId = hasTeamSelected || isUnassigned ? currentTeam?.id : 0;
      const separator = path.includes("?") ? "&" : "?";
      return `${path}${separator}fleet_id=${targetId}`;
    },
    [isPremiumTier, hasTeamSelected, isUnassigned, currentTeam?.id]
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

  const subPagePlaceholders: Partial<Record<Page, string>> = {
    "switch-fleet": "Search a fleet...",
    "view-host": "Search hosts...",
    "view-software": "Search software inventory...",
    "view-software-library": "Search software library...",
    "view-report": "Search reports...",
    "view-policy": "Search policies...",
  };
  const subPagePlaceholder = subPagePlaceholders[page];

  // Toggle open on Cmd+K / Ctrl+K; jump to switch-fleet on Cmd+Shift+F.
  // Focus is handled by the [open, page] effect below — don't rAF here, the
  // input ref isn't set until Radix's portal mounts.
  // Skip registration entirely for no-access users so we don't intercept
  // (and preventDefault) keyboard shortcuts for a palette they can't open.
  useEffect(() => {
    if (isNoAccess) return undefined;
    const onKeyDown = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey)) return;
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
  }, [canSwitchFleet, isNoAccess]);

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

  // Backspace on empty input returns to root from a sub-page.
  const onKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (page !== "root" && e.key === "Backspace" && !search) {
        e.preventDefault();
        goBack();
      }
    },
    [page, search, goBack]
  );

  // Intercept Escape on a sub-page so it returns to root instead of
  // closing the dialog. cmdk 1.1.1's Command.Dialog doesn't forward
  // `onEscapeKeyDown` to Radix's Dialog.Content, so we can't override
  // the close intent via props.
  //
  // Approach: a capture-phase document listener that calls
  // `stopImmediatePropagation` on Escape from a sub-page. This prevents
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
        withTeamRequired,
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
      withTeamRequired,
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
        map.set(getItemValue(item).toLowerCase().trim(), item.id);
        item.subItems.forEach((sub) => {
          const subValue = `${sub.label} ${sub.keywords?.join(" ") ?? ""}`
            .toLowerCase()
            .trim();
          map.set(subValue, item.id);
        });
      }
    });
    return map;
  }, [items]);

  // Auto expand/collapse sub-items as the user arrows through items.
  // Normalize to match how valueToParentId stores its keys — cmdk
  // hands back the raw `value` prop (preserves casing/whitespace).
  const handleHighlightChange = useCallback(
    (value: string) => {
      if (isSearching) return;
      const parentId = valueToParentId.get(value.toLowerCase().trim());
      setExpandedItems(parentId ? new Set([parentId]) : new Set());
    },
    [valueToParentId, isSearching]
  );

  // Find exact match — an item or sub-item whose label exactly matches the
  // search. Memoized so we don't rebuild the Set on every keystroke once
  // items is stable.
  const exactMatchIds = useMemo(() => {
    if (!isSearching) return new Set<string>();
    return new Set(
      items.reduce<string[]>((acc, item) => {
        if (item.label.toLowerCase() === searchLower) {
          acc.push(item.id);
        }
        item.subItems
          ?.filter((sub) => sub.label.toLowerCase() === searchLower)
          .forEach((sub) => acc.push(sub.id));
        return acc;
      }, [])
    );
  }, [items, isSearching, searchLower]);

  const renderItem = (item: ICommandItem) => {
    const isExpanded = expandedItems.has(item.id);
    const hasSubItems = item.subItems && item.subItems.length > 0;

    return (
      <React.Fragment key={item.id}>
        <Command.Item
          value={getItemValue(item)}
          onSelect={() =>
            item.onAction ? item.onAction() : navigate(item.path!)
          }
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
            {item.opensSubPage && (
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
            <span className={`${baseClass}__item-fleet`}>{item.teamName}</span>
          )}
        </Command.Item>
        {/* Render sub-items when expanded (browsing) or always when searching */}
        {hasSubItems &&
          (isExpanded || isSearching) &&
          item.subItems &&
          item.subItems.map((sub) => (
            <Command.Item
              key={sub.id}
              value={`${sub.label} ${sub.keywords?.join(" ") ?? ""}`}
              onSelect={() => navigate(sub.path)}
              className={`${baseClass}__item ${baseClass}__item--sub`}
            >
              <span className={`${baseClass}__item-label`}>{sub.label}</span>
            </Command.Item>
          ))}
      </React.Fragment>
    );
  };

  // Collect exact match items for the "Best match" section
  const exactMatchItems = useMemo(() => {
    if (exactMatchIds.size === 0) return [];
    return items.reduce<Array<{ item: ICommandItem; sub?: ICommandSubItem }>>(
      (acc, item) => {
        if (exactMatchIds.has(item.id)) {
          acc.push({ item });
        }
        item.subItems
          ?.filter((sub) => exactMatchIds.has(sub.id))
          .forEach((sub) => acc.push({ item, sub }));
        return acc;
      },
      []
    );
  }, [items, exactMatchIds]);

  const renderRootPage = () => (
    <>
      {/* Exact match at the top with a separator */}
      {exactMatchItems.length > 0 && (
        <>
          <Command.Group heading="Best match" className={`${baseClass}__group`}>
            {exactMatchItems.map(({ item, sub }) => {
              const target = sub || item;
              return (
                <Command.Item
                  key={`exact-${target.id}`}
                  value={`EXACT_MATCH ${target.label}`}
                  onSelect={() =>
                    item.onAction ? item.onAction() : navigate(target.path!)
                  }
                  className={`${baseClass}__item`}
                >
                  <span className={`${baseClass}__item-label`}>
                    {target.label}
                  </span>
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
        return (
          <Command.Group
            key={group}
            heading={group}
            className={`${baseClass}__group`}
          >
            {groupItems.map(renderItem)}
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
      onValueChange={handleHighlightChange}
      label="Command palette"
      className={baseClass}
      overlayClassName={`${baseClass}__overlay`}
      contentClassName={`${baseClass}__content`}
      filter={(value, searchTerm) => {
        // Always show exact match items at the top
        if (value.startsWith("EXACT_MATCH ")) {
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
          placeholder={subPagePlaceholder ?? "Search for a page or command..."}
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
              <span className={`${baseClass}__shortcut-sep`}>+</span>
              <kbd className={`${baseClass}__shortcut-key`}>⇧</kbd>
              <span className={`${baseClass}__shortcut-sep`}>+</span>
              <kbd className={`${baseClass}__shortcut-key`}>F</kbd>
            </span>
          </button>
        )}
        {page !== "root" && <kbd className={`${baseClass}__esc-hint`}>ESC</kbd>}
      </div>
      {/* Announce sub-page transitions to screen readers — the placeholder
          text changes but isn't reliably announced on its own. Strip the
          trailing ellipsis so the announcement isn't verbalized as
          "dot dot dot" by some screen readers. */}
      <div role="status" aria-live="polite" className="sr-only">
        {page === "root" ? "" : subPagePlaceholder?.replace(/\.{3}$/, "") ?? ""}
      </div>
      <Command.List className={`${baseClass}__list`}>
        {/* Sub-pages render their own contextual empty state, so only show
            cmdk's generic Empty on the root page. */}
        {page === "root" && (
          <Command.Empty className={`${baseClass}__empty`}>
            No results found.
          </Command.Empty>
        )}
        {page === "root" && renderRootPage()}
        {page === "switch-fleet" && (
          <FleetPicker
            availableTeams={availableTeams}
            currentTeam={currentTeam}
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
