import React, {
  useContext,
  useEffect,
  useState,
  useCallback,
  useRef,
} from "react";
import { Command } from "cmdk";
import { browserHistory } from "react-router";

import { AppContext } from "context/app";
import { APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";
import Icon from "components/Icon";
import Button from "components/buttons/Button";
import { toggleDarkMode } from "utilities/theme";
import paths from "router/paths";

import {
  ICommandItem,
  ICommandSubItem,
  GROUPS,
  buildCommandItems,
} from "./helpers";

const baseClass = "command-palette";

type Page = "root" | "switch-fleet";

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

  // Toggle open on Cmd+K / Ctrl+K
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((prev) => !prev);
      }
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, []);

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

  // Backspace on empty input returns to root page
  const onKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (page !== "root" && e.key === "Backspace" && !search) {
        e.preventDefault();
        goBack();
      }
    },
    [page, search, goBack]
  );

  const handleSwitchFleet = useCallback(
    (fleetId: number) => {
      const selected = availableTeams?.find((t) => t.id === fleetId);
      if (selected) {
        setCurrentTeam(selected);
      }
      setOpen(false);

      // Update the current URL's fleet_id param to reflect the switch
      const { pathname, search: currentSearch } = window.location;
      const params = new URLSearchParams(currentSearch);
      if (fleetId === APP_CONTEXT_ALL_TEAMS_ID) {
        params.delete("fleet_id");
      } else {
        params.set("fleet_id", String(fleetId));
      }
      const qs = params.toString();
      browserHistory.push(qs ? `${pathname}?${qs}` : pathname);
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

  if (isNoAccess) {
    return null;
  }

  const items = buildCommandItems({
    search,
    currentTeam,
    config,
    canAccessControls,
    canWrite,
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
      toggleDarkMode();
      setOpen(false);
    },
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

  const getItemValue = (item: ICommandItem) => {
    const parts = [item.label, ...(item.keywords ?? [])];
    item.subItems?.forEach((sub) => {
      parts.push(sub.label, ...(sub.keywords ?? []));
    });
    return parts.join(" ");
  };

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
          </div>
          {item.teamName && (
            <span className={`${baseClass}__item-team`}>{item.teamName}</span>
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
              {item.teamName && (
                <span className={`${baseClass}__item-team`}>
                  {item.teamName}
                </span>
              )}
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
                  {"teamName" in item && item.teamName && (
                    <span className={`${baseClass}__item-team`}>
                      {item.teamName}
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
      {/* Navigate group — switch fleet + sign out */}
      <Command.Group heading="Navigate" className={`${baseClass}__group`}>
        {isPremiumTier && availableTeams && availableTeams.length > 1 && (
          <Command.Item
            value="Switch fleet team change"
            onSelect={() => goToPage("switch-fleet")}
            className={`${baseClass}__item`}
          >
            <span className={`${baseClass}__item-label`}>Switch fleet...</span>
            {currentTeam?.name && (
              <span className={`${baseClass}__item-team`}>
                {currentTeam.name}
              </span>
            )}
          </Command.Item>
        )}
        <Command.Item
          value="Sign out logout log out"
          onSelect={() => navigate(paths.LOGOUT)}
          className={`${baseClass}__item`}
        >
          <span className={`${baseClass}__item-label`}>Sign out</span>
        </Command.Item>
      </Command.Group>
    </>
  );

  const renderSwitchFleetPage = () => (
    <Command.Group heading="Switch fleet" className={`${baseClass}__group`}>
      {availableTeams?.map((fleet) => {
        const isActive = currentTeam?.id === fleet.id;
        return (
          <Command.Item
            key={`fleet-${fleet.id}`}
            value={fleet.name}
            onSelect={() => handleSwitchFleet(fleet.id)}
            className={`${baseClass}__item`}
          >
            <span className={`${baseClass}__item-label`}>{fleet.name}</span>
            {isActive && (
              <span className={`${baseClass}__item-team`}>Current</span>
            )}
          </Command.Item>
        );
      })}
    </Command.Group>
  );

  return (
    <Command.Dialog
      open={open}
      onOpenChange={setOpen}
      label="Command palette"
      className={baseClass}
      overlayClassName={`${baseClass}__overlay`}
      contentClassName={`${baseClass}__content`}
      filter={(value, searchTerm) => {
        // Always show exact match items at the top
        if (value.startsWith("EXACT_MATCH ")) {
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
        <Icon
          name="search"
          color="ui-fleet-black-75"
          className={`${baseClass}__input-icon`}
        />
        <Command.Input
          ref={inputRef}
          className={`${baseClass}__input`}
          placeholder={
            page === "switch-fleet"
              ? "Search fleets"
              : "Search pages or actions"
          }
          value={search}
          onValueChange={setSearch}
          onKeyDown={onKeyDown}
        />
        {page !== "root" && (
          <Button
            variant="inverse"
            size="small"
            className={`${baseClass}__back-button`}
            onClick={goBack}
          >
            <Icon name="chevron-left" color="ui-fleet-black-50" />
            <span>Back</span>
          </Button>
        )}
      </div>
      <Command.List className={`${baseClass}__list`}>
        <Command.Empty className={`${baseClass}__empty`}>
          No results found.
        </Command.Empty>
        {page === "root" && renderRootPage()}
        {page === "switch-fleet" && renderSwitchFleetPage()}
      </Command.List>
    </Command.Dialog>
  );
};

export default CommandPalette;
