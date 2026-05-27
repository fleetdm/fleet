import React from "react";
import { Command } from "cmdk";

import {
  APP_CONTEXT_ALL_TEAMS_ID,
  APP_CONTEXT_NO_TEAM_ID,
  ITeamSummary,
} from "interfaces/team";
import { getFleetSuffix } from "./pickerCopy";
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

import usePickerSearch from "./usePickerSearch";
import { RESULT_PREFIXES } from "./constants";

const baseClass = "command-palette";

const SOFTWARE_SEARCH_LIMIT = 50;

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
  const teamId =
    currentTeam && currentTeam.id !== APP_CONTEXT_ALL_TEAMS_ID
      ? currentTeam.id
      : undefined;

  const fleetSuffix = getFleetSuffix(currentTeam);

  // For library copy ("No software in X's library.") we need a possessive
  // form. Library is hidden on All fleets so that branch shouldn't
  // render here, but we still default defensively.
  const libraryOwner = (() => {
    if (currentTeam && currentTeam.id > 0) return `${currentTeam.name}'s library`;
    if (currentTeam?.id === APP_CONTEXT_NO_TEAM_ID) return "this fleet's library";
    return "the library";
  })();

  const libraryOnly = scope === "library";

  const { items: titles, isLoading, debouncedQuery } = usePickerSearch<
    ISoftwareTitlesResponse,
    ISoftwareTitle
  >({
    search,
    queryKeyPrefix: ["commandPaletteSoftware", scope, teamId ?? "global"],
    queryFn: (q) =>
      softwareAPI.getSoftwareTitles({
        page: 0,
        perPage: SOFTWARE_SEARCH_LIMIT,
        teamId,
        availableForInstall: libraryOnly || undefined,
        query: q || undefined,
        orderKey: "name",
        orderDirection: "asc",
      }),
    selectItems: (data) => data?.software_titles ?? [],
  });

  if (isLoading && titles.length === 0) {
    return <div className={`${baseClass}__empty`}>Looking for software...</div>;
  }

  if (titles.length === 0) {
    let emptyMessage: string;
    if (libraryOnly) {
      emptyMessage = debouncedQuery
        ? `No library software matches "${debouncedQuery}" in ${libraryOwner}.`
        : `No software in ${libraryOwner}.`;
    } else {
      emptyMessage = debouncedQuery
        ? `No software matches "${debouncedQuery}"${fleetSuffix}.`
        : `No software found${fleetSuffix}.`;
    }
    return <div className={`${baseClass}__empty`}>{emptyMessage}</div>;
  }

  return (
    <Command.Group className={`${baseClass}__group`}>
      {titles.map((title) => {
        const label = title.display_name || title.name;
        const typeLabel = formatSoftwareType(title);
        const installerProps = getInstallerProps(title);
        return (
          <Command.Item
            key={`software-${title.id}`}
            value={`${RESULT_PREFIXES.software}${title.id}`}
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
