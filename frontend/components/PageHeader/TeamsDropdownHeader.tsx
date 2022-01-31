import React, { useCallback, useContext, useEffect } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router/lib/Router";

import { AppContext, IAppContext } from "context/app";
import usersAPI, { IGetMeResponse } from "services/entities/users";

import TeamsDropdown from "../TeamsDropdown/TeamsDropdown";

export interface ITeamsDropdownState extends Partial<IAppContext> {
  teamId?: number;
}

interface ITeamsDropdownHeaderProps {
  router: InjectedRouter;
  location: {
    pathname: string;
    query: { team_id?: string; vulnerable?: boolean };
    search: string;
  };
  baseClass: string;
  defaultTitle: string;
  buttons?: (ctx: ITeamsDropdownState) => JSX.Element | null;
  onChange: (ctx: ITeamsDropdownState) => void;
  description: (ctx: ITeamsDropdownState) => JSX.Element | string | null;
}

const TeamsDropdownHeader = ({
  router,
  location,
  baseClass,
  defaultTitle,
  buttons,
  description,
  onChange,
}: ITeamsDropdownHeaderProps): JSX.Element | null => {
  const teamId = parseInt(location?.query?.team_id || "", 10) || 0;

  const {
    availableTeams,
    config,
    currentUser,
    currentTeam,
    enrollSecret,
    isPreviewMode,
    isFreeTier,
    isPremiumTier,
    isGlobalAdmin,
    isGlobalMaintainer,
    isGlobalObserver,
    isOnGlobalTeam,
    isAnyTeamMaintainer,
    isAnyTeamMaintainerOrTeamAdmin,
    isTeamObserver,
    isTeamMaintainer,
    isTeamMaintainerOrTeamAdmin,
    isAnyTeamAdmin,
    isTeamAdmin,
    isOnlyObserver,
    setAvailableTeams,
    setCurrentTeam,
    setCurrentUser,
  } = useContext(AppContext);

  // The dropdownState is the context and local state made available to callback functions.
  // Additional state/context can be made available here if needed for new uses cases.
  const dropdownState = {
    // NOTE: teamId is the value independently determined by this component
    // and may briefly be a step ahead of the AppContext for currentTeam
    // depending on the cycle of state updating and rendering
    teamId,
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

  useQuery(["me"], () => usersAPI.me(), {
    onSuccess: ({ user, available_teams }: IGetMeResponse) => {
      setCurrentUser(user);
      setAvailableTeams(available_teams);
    },
  });

  const findAvailableTeam = (id: number) => {
    return availableTeams?.find((t) => t.id === id);
  };

  const buildQueryString = (queryString: string, newTeamId: number) => {
    queryString = queryString.startsWith("?")
      ? queryString.slice(1)
      : queryString;
    const queryParams = queryString.split("&").filter((el) => el.includes("="));
    const teamIndex = queryParams.findIndex((el) => el.includes("team_id"));

    if (newTeamId) {
      const teamParam = `team_id=${newTeamId}`;
      if (teamIndex >= 0) {
        // replace old team param
        queryParams.splice(teamIndex, 1, teamParam);
      } else {
        // add new team param
        queryParams.push(teamParam);
      }
    } else {
      // remove old team param
      teamIndex >= 0 && queryParams.splice(teamIndex, 1);
    }
    queryString = queryParams.length ? "?".concat(queryParams.join("&")) : "";

    return queryString;
  };

  // TODO: Add support for pages that use teamId in route params as alternative to query string
  const handleTeamSelect = useCallback(
    (id: number) => {
      const availableTeam = findAvailableTeam(id);
      setCurrentTeam(availableTeam);
      const queryString = buildQueryString(location?.search, id);
      if (location?.search !== queryString) {
        const path = location?.pathname?.concat(queryString) || "";
        !!path && router.replace(path);
      }
      if (onChange) {
        onChange({ ...dropdownState, teamId: availableTeam?.id });
      }
    },
    [location, router]
  );

  // If team_id from URL query params is not valid, we instead use a default team
  // either the current team (if any) or all teams (for global users) or
  // the first available team (for non-global users)
  const getValidatedTeamId = () => {
    if (findAvailableTeam(teamId)) {
      return teamId;
    }
    if (!teamId && currentTeam) {
      return currentTeam.id;
    }
    if (!teamId && !currentTeam && !isOnGlobalTeam && availableTeams) {
      return availableTeams[0]?.id;
    }
    return 0;
  };

  // If team_id or currentTeam doesn't match validated id, switch to validated id
  useEffect(() => {
    if (availableTeams) {
      const validatedId = getValidatedTeamId();

      if (validatedId !== currentTeam?.id || validatedId !== teamId) {
        handleTeamSelect(validatedId);
      }
    }
  }, [availableTeams]);

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
                      selectedTeamId={teamId}
                      onChange={(newSelectedValue: number) =>
                        handleTeamSelect(newSelectedValue)
                      }
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
