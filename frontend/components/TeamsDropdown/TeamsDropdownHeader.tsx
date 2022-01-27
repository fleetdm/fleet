import React, { useCallback, useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";

import { AppContext, IAppContext } from "context/app";
import usersAPI, { IGetMeResponse } from "services/entities/users";

import TeamsDropdown from "./TeamsDropdown";

export interface ITeamsDropdownContext extends Partial<IAppContext> {
  teamId?: number;
}

interface ITeamsDropdownHeaderProps {
  router: any;
  location: any;
  params?: any;
  routeParams?: any;
  route?: any;
  baseClass: string;
  defaultTitle: string;
  buttons?: (ctx: ITeamsDropdownContext) => JSX.Element | null;
  onChange: (ctx: ITeamsDropdownContext) => void;
  description: (ctx: ITeamsDropdownContext) => JSX.Element | string | null;
}

const TeamsDropdownHeader = ({
  router,
  location,
  params,
  routeParams,
  route,
  baseClass,
  defaultTitle,
  buttons,
  description,
  onChange,
}: ITeamsDropdownHeaderProps): JSX.Element | null => {
  const teamId = parseInt(location?.query?.team_id, 10) || 0;

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

  // The dropdownContext is made available to callback functions.
  // Additional context can be made available here if needed for new uses cases.
  const dropdownContext = {
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

  // TODO confirm approach to path and location
  const handleTeamSelect = (id: number) => {
    const availableTeam = findAvailableTeam(id);
    const path = availableTeam?.id
      ? `${location?.pathname}?team_id=${availableTeam.id}`
      : location?.pathname;

    router.replace(path);
    setCurrentTeam(availableTeam);
    if (typeof onChange === "function") {
      onChange({ ...dropdownContext, teamId: availableTeam?.id });
    }
  };

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
      <div className={`${baseClass} button-wrap`}>
        {buttons(dropdownContext)}
      </div>
    ) : null;
  };

  const renderDescription = () => {
    const contents =
      typeof description === "function"
        ? description(dropdownContext)
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
