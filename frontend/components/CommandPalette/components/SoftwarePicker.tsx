import React, { useEffect, useState } from "react";
import { Command } from "cmdk";
import { useQuery } from "react-query";

import { APP_CONTEXT_ALL_TEAMS_ID, ITeamSummary } from "interfaces/team";
import {
  formatSoftwareType,
  isIpadOrIphoneSoftwareSource,
  ISoftwareTitle,
} from "interfaces/software";
import softwareAPI, {
  ISoftwareTitlesResponse,
} from "services/entities/software";
import { getAutomaticInstallPoliciesCount } from "pages/SoftwarePage/helpers";
import { InstallIconWithTooltip } from "components/TableContainer/DataTable/SoftwareNameCell/SoftwareNameCell";

const baseClass = "command-palette";

const SOFTWARE_SEARCH_LIMIT = 50;
const SOFTWARE_SEARCH_DEBOUNCE_MS = 200;

type SoftwareScope = "inventory" | "library";

// Derives the install-icon tooltip props from a software title, mirroring
// getSoftwareNameCellData in SoftwareLibraryTableConfig. Returns null when
// the title has no installer attached (not in any library).
const getInstallerProps = (title: ISoftwareTitle) => {
  const installer = title.software_package || title.app_store_app;
  if (!installer) {
    return null;
  }
  return {
    isSelfService: installer.self_service,
    automaticInstallPoliciesCount: getAutomaticInstallPoliciesCount(title),
    isIosOrIpadosApp: isIpadOrIphoneSoftwareSource(title.source),
    isAndroidPlayStoreApp:
      !!title.app_store_app && title.source === "android_apps",
  };
};

interface ISoftwarePickerProps {
  search: string;
  currentTeam?: ITeamSummary;
  scope?: SoftwareScope;
  onSelect: (softwareId: number) => void;
}

const SoftwarePicker = ({
  search,
  currentTeam,
  scope = "inventory",
  onSelect,
}: ISoftwarePickerProps): JSX.Element => {
  // Debounce the raw search input so we don't fire a request per keystroke.
  const [debouncedQuery, setDebouncedQuery] = useState(search.trim());
  useEffect(() => {
    const id = window.setTimeout(() => {
      setDebouncedQuery(search.trim());
    }, SOFTWARE_SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(id);
  }, [search]);

  // Software inventory is team-scoped — different fleets have different
  // installed apps. The picker shows what's relevant to the user's current
  // fleet context.
  const teamId =
    currentTeam && currentTeam.id !== APP_CONTEXT_ALL_TEAMS_ID
      ? currentTeam.id
      : undefined;
  const fleetLabel =
    currentTeam && currentTeam.id !== APP_CONTEXT_ALL_TEAMS_ID
      ? currentTeam.name
      : "All fleets";

  const libraryOnly = scope === "library";

  const { data, isLoading } = useQuery<ISoftwareTitlesResponse, Error>(
    ["commandPaletteSoftware", scope, teamId, debouncedQuery],
    () =>
      softwareAPI.getSoftwareTitles({
        page: 0,
        perPage: SOFTWARE_SEARCH_LIMIT,
        teamId,
        availableForInstall: libraryOnly || undefined,
        query: debouncedQuery || undefined,
        orderKey: "name",
        orderDirection: "asc",
      }),
    {
      keepPreviousData: true,
      staleTime: 30000,
    }
  );

  const titles = data?.software_titles ?? [];

  if (isLoading && titles.length === 0) {
    return <div className={`${baseClass}__empty`}>Looking for software...</div>;
  }

  if (titles.length === 0) {
    let emptyMessage: string;
    if (libraryOnly) {
      emptyMessage = debouncedQuery
        ? `No library software matches "${debouncedQuery}" in ${fleetLabel}.`
        : `No software in ${fleetLabel}'s library.`;
    } else {
      emptyMessage = debouncedQuery
        ? `No software matches "${debouncedQuery}" in ${fleetLabel}.`
        : `No software found in ${fleetLabel}.`;
    }
    return <div className={`${baseClass}__empty`}>{emptyMessage}</div>;
  }

  return (
    <Command.Group className={`${baseClass}__group`}>
      {titles.map((title) => {
        const label = title.display_name || title.name;
        const typeLabel = formatSoftwareType(title);
        const installerProps = getInstallerProps(title);
        // value prefixed with SOFTWARE_RESULT so cmdk's local filter passes
        // it through — the server already filtered by debouncedQuery.
        return (
          <Command.Item
            key={`software-${title.id}`}
            value={`SOFTWARE_RESULT ${title.id}`}
            onSelect={() => onSelect(title.id)}
            className={`${baseClass}__item`}
          >
            <div className={`${baseClass}__item-left`}>
              <span className={`${baseClass}__item-label`}>{label}</span>
              {installerProps && <InstallIconWithTooltip {...installerProps} />}
            </div>
            {typeLabel && (
              <span className={`${baseClass}__item-meta`}>{typeLabel}</span>
            )}
          </Command.Item>
        );
      })}
    </Command.Group>
  );
};

export default SoftwarePicker;
