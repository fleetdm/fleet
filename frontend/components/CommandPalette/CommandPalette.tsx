import React, {
  useContext,
  useEffect,
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
import TooltipWrapper from "components/TooltipWrapper";
import { isDarkMode, setThemeMode } from "utilities/theme";
import paths from "router/paths";

import {
  ICommandItem,
  ICommandSubItem,
  GROUPS,
  buildCommandItems,
} from "./helpers";
import FleetPicker from "./components/FleetPicker";
import HostPicker from "./components/HostPicker";
import SoftwarePicker from "./components/SoftwarePicker";
import ReportPicker from "./components/ReportPicker";
import PolicyPicker from "./components/PolicyPicker";

const baseClass = "command-palette";

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
    isGlobalTechnician,
    isAnyTeamTechnician,
    isObserverPlus,
    isAnyTeamObserverPlus,
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

  // Observer+ users can run live queries even though they can't write.
  const canRunLiveReport =
    canWrite || !!isObserverPlus || !!isAnyTeamObserverPlus;

  // Policy automations: same as canAddOrDeletePolicies in ManagePoliciesPage
  const canManagePolicyAutomations =
    isGlobalAdmin ||
    isGlobalMaintainer ||
    isAnyTeamAdmin ||
    isAnyTeamMaintainer;

  // Software automations require global admin (all fleets view)
  const canManageSoftwareAutomations = isGlobalAdmin;

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
    isPremiumTier && !!availableTeams && availableTeams.length > 1;

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
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey)) return;
      if (e.key === "k") {
        e.preventDefault();
        setOpen((prev) => !prev);
      } else if (
        e.shiftKey &&
        (e.key === "f" || e.key === "F") &&
        canSwitchFleet
      ) {
        e.preventDefault();
        setOpen(true);
        setSearch("");
        setPage("switch-fleet");
      }
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [canSwitchFleet]);

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

  // Intercept dialog close: Escape on a sub-page goes back to root, but
  // click-outside still closes the palette outright. cmdk 1.1.1's
  // Command.Dialog does not forward props (e.g., onEscapeKeyDown) to the
  // underlying Radix Dialog.Content, and onOpenChange alone can't tell us
  // why the dialog is closing. So we flag Escape via a capture-phase
  // document listener and consume the flag in handleOpenChange.
  const pageRef = useRef(page);
  pageRef.current = page;
  const closeViaEscapeRef = useRef(false);

  useEffect(() => {
    if (!open) {
      return undefined;
    }
    const onDocKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        closeViaEscapeRef.current = true;
      }
    };
    document.addEventListener("keydown", onDocKey, true);
    return () => document.removeEventListener("keydown", onDocKey, true);
  }, [open]);

  const handleOpenChange = useCallback(
    (nextOpen: boolean) => {
      const viaEscape = closeViaEscapeRef.current;
      closeViaEscapeRef.current = false;
      if (!nextOpen && viaEscape && pageRef.current !== "root") {
        goBack();
        return;
      }
      setOpen(nextOpen);
    },
    [goBack]
  );

  const handleSwitchFleet = useCallback(
    (fleetId: number) => {
      const selected = availableTeams?.find((t) => t.id === fleetId);
      if (selected) {
        setCurrentTeam(selected);
      }

      const { pathname, search: currentSearch } = window.location;
      const isAll = fleetId === APP_CONTEXT_ALL_TEAMS_ID;
      const isUnassignedTarget = fleetId === 0;

      // Pages that require a specific fleet — can't render "All fleets" or
      // (with some overlap) "Unassigned". When switching to those contexts
      // from one of these pages, fall back to Hosts which supports both.
      const teamRequiredPrefixes = [
        paths.CONTROLS,
        paths.SOFTWARE_LIBRARY,
        paths.NEW_REPORT,
      ];
      const isOnTeamRequiredPage = teamRequiredPrefixes.some((p) =>
        pathname.startsWith(p)
      );

      if ((isAll || isUnassignedTarget) && isOnTeamRequiredPage) {
        browserHistory.push(paths.MANAGE_HOSTS);
      } else {
        const params = new URLSearchParams(currentSearch);
        if (isAll) {
          params.delete("fleet_id");
        } else {
          params.set("fleet_id", String(fleetId));
        }
        const qs = params.toString();
        browserHistory.push(qs ? `${pathname}?${qs}` : pathname);
      }

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

  const items = buildCommandItems({
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
    isTechnician,
    isPremiumTier,
    isMacMdmEnabledAndConfigured,
    isWindowsMdmEnabledAndConfigured,
    isAndroidMdmEnabledAndConfigured,
    isVppEnabled,
    hasTeamSelected,
    withTeamId,
    onToggleDarkMode: () => {
      setThemeMode(isDarkMode() ? "light" : "dark");
      setOpen(false);
    },
    onViewHost: () => goToPage("view-host"),
    onViewSoftware: () => goToPage("view-software"),
    onViewSoftwareLibrary: () => goToPage("view-software-library"),
    onViewReport: () => goToPage("view-report"),
    onViewPolicy: () => goToPage("view-policy"),
  });

  const groupedItems = items.reduce<Record<string, ICommandItem[]>>(
    (acc, item) => {
      if (!acc[item.group]) {
        acc[item.group] = [];
      }
      acc[item.group].push(item);
      return acc;
    },
    {}
  );

  const isSearching = search.length > 0;
  const searchLower = search.toLowerCase().trim();

  const getItemValue = (item: ICommandItem) => {
    const parts = [item.label, ...(item.keywords ?? [])];
    item.subItems?.forEach((sub) => {
      parts.push(sub.label, ...(sub.keywords ?? []));
    });
    return parts.join(" ");
  };

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

  // Auto expand/collapse sub-items as the user arrows through items
  const handleHighlightChange = useCallback(
    (value: string) => {
      if (isSearching) return;
      const parentId = valueToParentId.get(value);
      setExpandedItems(parentId ? new Set([parentId]) : new Set());
    },
    [valueToParentId, isSearching]
  );

  // Find exact match — an item or sub-item whose label exactly matches the search
  const exactMatchIds = isSearching
    ? new Set(
        items.reduce<string[]>((acc, item) => {
          if (item.label.toLowerCase() === searchLower) {
            acc.push(item.id);
          }
          item.subItems
            ?.filter((sub) => sub.label.toLowerCase() === searchLower)
            .forEach((sub) => acc.push(sub.id));
          return acc;
        }, [])
      )
    : new Set<string>();

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
              <span
                aria-hidden
                className={`${baseClass}__item-more`}
              >
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
  const exactMatchItems =
    exactMatchIds.size > 0
      ? items.reduce<Array<{ item: ICommandItem; sub?: ICommandSubItem }>>(
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
        )
      : [];

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
        // Host & software results are pre-filtered by the server; show
        // everything we got.
        if (
          value.startsWith("HOST_RESULT ") ||
          value.startsWith("SOFTWARE_RESULT ") ||
          value.startsWith("REPORT_RESULT ") ||
          value.startsWith("POLICY_RESULT ")
        ) {
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
          <button
            type="button"
            aria-label="Back"
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
          <TooltipWrapper
            tipContent="Use ⌘ + Shift + F to select a fleet."
            position="bottom-end"
            tipOffset={2}
            underline={false}
            className={`${baseClass}__fleet-switcher-tooltip`}
          >
            <button
              type="button"
              className={`${baseClass}__fleet-switcher`}
              onClick={() => goToPage("switch-fleet")}
            >
              <span className={`${baseClass}__fleet-switcher-label`}>
                {currentTeam?.name || "All fleets"}
              </span>
              <Icon
                name="chevron-down"
                color="ui-fleet-black-75"
                className={`${baseClass}__fleet-switcher-caret`}
              />
            </button>
          </TooltipWrapper>
        )}
        {page !== "root" && <kbd className={`${baseClass}__esc-hint`}>ESC</kbd>}
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
          <HostPicker search={search} onSelect={handleSelectHost} />
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
            onSelect={handleSelectReport}
          />
        )}
        {page === "view-policy" && (
          <PolicyPicker
            search={search}
            currentTeam={currentTeam}
            onSelect={handleSelectPolicy}
          />
        )}
      </Command.List>
    </Command.Dialog>
  );
};

export default CommandPalette;
