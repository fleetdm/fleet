import React, { useCallback, useContext } from "react";
import { InjectedRouter } from "react-router/lib/Router";

import { AppContext, IAppContext } from "context/app";
import { ALL_TEAMS_ID, NO_TEAM_ID } from "interfaces/team";

import TeamsDropdown from "../TeamsDropdown/TeamsDropdown";

export interface ITeamsDropdownState extends Partial<IAppContext> {
  teamId?: number;
}

interface ITeamsDropdownHeaderProps {
  router: InjectedRouter;
  location: {
    pathname: string;
    query: { team_id?: string; vulnerable?: string };
    search: string;
  };
  baseClass: string;
  defaultTitle: string;
  buttons?: (ctx: ITeamsDropdownState) => JSX.Element | null;
  onChange?: (ctx: ITeamsDropdownState) => void;
  description: (ctx: ITeamsDropdownState) => JSX.Element | string | null;
  includeNoTeam?: boolean;
  includeAll?: boolean;
}

const TeamsDropdownHeader = ({
  router,
  location,
  baseClass,
  defaultTitle,
  buttons,
  description,
  onChange,
  includeNoTeam = false,
  includeAll = true,
}: ITeamsDropdownHeaderProps): JSX.Element | null => {
  const {
    availableTeams,
    currentUser,
    currentTeam,
    isPreviewMode,
    isFreeTier,
    isPremiumTier,
    isGlobalAdmin,
    isGlobalMaintainer,
    isGlobalObserver,
    isOnGlobalTeam,
    isAnyTeamMaintainer,
    isAnyTeamMaintainerOrTeamAdmin,
    isAnyTeamAdmin,
    isTeamAdmin,
    isOnlyObserver,
  } = useContext(AppContext);

  // The dropdownState is the context and local state made available to callback functions.
  // Additional state/context can be made available here if needed for new uses cases.
  const dropdownState = {
    // NOTE: teamId is the value independently determined by this component
    // and may briefly be a step ahead of the AppContext for currentTeam
    // depending on the cycle of state updating and rendering
    teamId: currentTeam?.id,
    availableTeams,
    currentUser,
    isPreviewMode,
    isFreeTier,
    isPremiumTier,
    isGlobalAdmin,
    isGlobalMaintainer,
    isGlobalObserver,
    isOnGlobalTeam,
    isAnyTeamMaintainer,
    isAnyTeamMaintainerOrTeamAdmin,
    isAnyTeamAdmin,
    isTeamAdmin,
    isOnlyObserver,
  };

  const findAvailableTeam = (id: number) => {
    return availableTeams?.find((t) => t.id === id);
  };

  const buildQueryString = (queryString: string, newTeamId: number) => {
    queryString = queryString.startsWith("?")
      ? queryString.slice(1)
      : queryString;
    const queryParams = queryString.split("&").filter((el) => el.includes("="));
    const teamIndex = queryParams.findIndex((el) => el.includes("team_id"));

    if (newTeamId === ALL_TEAMS_ID) {
      // remove old team param if any
      teamIndex >= 0 && queryParams.splice(teamIndex, 1);
    } else {
      const teamParam = `team_id=${newTeamId}`;
      teamIndex >= 0
        ? queryParams.splice(teamIndex, 1, teamParam) // replace old param
        : queryParams.push(teamParam); // add new param
    }

    queryString = queryParams.length ? "?".concat(queryParams.join("&")) : "";

    return queryString;
  };

  // TODO: Add support for pages that use teamId in route params as alternative to query string
  const handleTeamSelect = useCallback(
    (id: number) => {
      const queryString = buildQueryString(location?.search, id);
      if (location?.search !== queryString) {
        const path = location?.pathname?.concat(queryString) || "";
        !!path && router.replace(path);
      }
      if (onChange) {
        onChange({ ...dropdownState, teamId: id });
      }
    },
    // TODO: add missing deps to this array if doens't cause bugs
    [location, router]
  );

  if (!availableTeams?.length) {
    return null;
  }

  let defaultId = availableTeams[0].id;
  if (includeAll) {
    defaultId = findAvailableTeam(ALL_TEAMS_ID) ? ALL_TEAMS_ID : defaultId;
  } else if (includeNoTeam) {
    defaultId = findAvailableTeam(NO_TEAM_ID) ? NO_TEAM_ID : defaultId;
  } else {
    const defaultTeam = availableTeams.find((t) => t.id > NO_TEAM_ID);
    defaultId = defaultTeam ? defaultTeam.id : defaultId;
  }

  if (currentTeam?.id === ALL_TEAMS_ID && !includeAll) {
    handleTeamSelect(defaultId);
  }
  if (currentTeam?.id === NO_TEAM_ID && !includeNoTeam) {
    handleTeamSelect(defaultId);
  }

  const renderButtons = () => {
    return buttons ? (
      <div className={`${baseClass} button-wrap`}>{buttons(dropdownState)}</div>
    ) : null;
  };

  const renderDescription = () => {
    const contents =
      typeof description === "function"
        ? description(dropdownState)
        : description;

    return contents ? (
      <div className={`${baseClass}__description`}>{contents}</div>
    ) : null;
  };

  const renderTeamsDropdown = () => {
    if (!availableTeams) {
      return null;
    }

    return (
      <>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <div className={`${baseClass}__title`}>
                {isFreeTier && <h1>{defaultTitle}</h1>}
                {isPremiumTier &&
                  (availableTeams.length > 1 || isOnGlobalTeam) && (
                    <TeamsDropdown
                      currentUserTeams={availableTeams || []}
                      selectedTeamId={currentTeam?.id || 0}
                      onChange={(newSelectedValue: number) =>
                        handleTeamSelect(newSelectedValue)
                      }
                      includeNoTeams={includeNoTeam}
                      includeAll={includeAll}
                    />
                  )}
                {isPremiumTier &&
                  !isOnGlobalTeam &&
                  availableTeams.length === 1 && (
                    <h1>{availableTeams[0].name}</h1>
                  )}
              </div>
            </div>
          </div>
          {renderButtons()}
        </div>
        {renderDescription()}
      </>
    );
  };

  return renderTeamsDropdown();
};

export default TeamsDropdownHeader;
