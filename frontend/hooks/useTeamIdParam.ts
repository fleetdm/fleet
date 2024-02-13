import { useCallback, useContext, useEffect, useMemo } from "react";
import { InjectedRouter } from "react-router";
import { findLastIndex, trimStart } from "lodash";

import { AppContext } from "context/app";
import { TableContext } from "context/table";
import {
  API_NO_TEAM_ID,
  API_ALL_TEAMS_ID,
  APP_CONTEXT_ALL_TEAMS_ID,
  APP_CONTEXT_NO_TEAM_ID,
  isAnyTeamSelected,
  ITeamSummary,
  ITeam,
} from "interfaces/team";
import { IUser, IUserRole } from "interfaces/user";
import permissions from "utilities/permissions";
import sort from "utilities/sort";

const splitQueryStringParts = (queryString: string) =>
  trimStart(queryString, "?")
    .split("&")
    .filter((p) => p.includes("="));

const joinQueryStringParts = (parts: string[]) =>
  parts.length ? `?${parts.join("&")}` : "";

const rebuildQueryStringWithTeamId = (
  queryString: string,
  newTeamId: number
) => {
  const parts = splitQueryStringParts(queryString);

  // Reset page to 0
  const pageIndex = parts.findIndex((p) => p.startsWith("page="));
  const inheritedPageIndex = parts.findIndex((p) =>
    p.startsWith("inherited_page=")
  );
  if (pageIndex !== -1) {
    parts.splice(pageIndex, 1, "page=0");
  }
  if (inheritedPageIndex !== -1) {
    parts.splice(inheritedPageIndex, 1, "inherited_page=0");
  }

  const teamIndex = parts.findIndex((p) => p.startsWith("team_id="));
  // URLs for the app represent "All teams" by the absence of the team id param
  const newTeamPart =
    newTeamId > APP_CONTEXT_ALL_TEAMS_ID ? `team_id=${newTeamId}` : "";

  if (teamIndex === -1) {
    // nothing to remove/replace so add the new part (if any) and rejoin
    return joinQueryStringParts(
      newTeamPart ? parts.concat(newTeamPart) : parts
    );
  }

  if (teamIndex !== findLastIndex(parts, (p) => p.startsWith("team_id="))) {
    console.warn(
      `URL contains more than one team_id parameter: ${queryString}`
    );
  }

  if (newTeamPart) {
    parts.splice(teamIndex, 1, newTeamPart); // remove the old part and replace with the new
  } else {
    parts.splice(teamIndex, 1); // just remove the old team part
  }

  return joinQueryStringParts(parts);
};

const filterUserTeamsByRole = (
  userTeams: ITeam[],
  permittedAccessByUserRole?: Record<IUserRole, boolean>
) => {
  if (!permittedAccessByUserRole) {
    return userTeams;
  }

  return userTeams
    .filter(
      ({ role }) => role && !!permittedAccessByUserRole[role as IUserRole]
    )
    .sort((a, b) => sort.caseInsensitiveAsc(a.name, b.name));
};

const getUserTeams = ({
  availableTeams,
  currentUser,
  permittedAccessByTeamRole,
}: {
  availableTeams?: ITeamSummary[];
  currentUser: IUser | null;
  permittedAccessByTeamRole?: Record<IUserRole, boolean>;
}) => {
  if (!currentUser || !availableTeams?.length) {
    return undefined;
  }

  return permissions.isOnGlobalTeam(currentUser)
    ? availableTeams
    : filterUserTeamsByRole(currentUser.teams, permittedAccessByTeamRole);
};

const getDefaultTeam = ({
  currentUser,
  includeAllTeams,
  includeNoTeam,
  userTeams,
}: {
  currentUser: IUser | null;
  includeAllTeams: boolean;
  includeNoTeam: boolean;
  userTeams?: ITeamSummary[];
}) => {
  if (!currentUser || !userTeams?.length) {
    return undefined;
  }
  if (permissions.isOnGlobalTeam(currentUser)) {
    let defaultTeam: ITeamSummary | undefined;
    if (includeAllTeams) {
      defaultTeam = userTeams.find((t) => t.id === APP_CONTEXT_ALL_TEAMS_ID);
    }
    if (!defaultTeam && includeNoTeam) {
      defaultTeam = userTeams.find((t) => t.id === APP_CONTEXT_NO_TEAM_ID);
    }

    return defaultTeam || userTeams.find((t) => t.id > APP_CONTEXT_NO_TEAM_ID);
  }

  return (
    userTeams.find((t) => permissions.isTeamAdmin(currentUser, t.id)) ||
    userTeams.find((t) => permissions.isTeamMaintainer(currentUser, t.id)) ||
    userTeams.find((t) => t.id > APP_CONTEXT_NO_TEAM_ID)
  );
};

const getTeamIdForApi = ({
  currentTeam,
  includeAllTeams = true,
  includeNoTeam = false,
}: {
  currentTeam?: ITeamSummary;
  includeAllTeams?: boolean;
  includeNoTeam?: boolean;
}) => {
  if (includeNoTeam && currentTeam?.id === APP_CONTEXT_NO_TEAM_ID) {
    return API_NO_TEAM_ID;
  }
  if (includeAllTeams && currentTeam?.id === APP_CONTEXT_ALL_TEAMS_ID) {
    return API_ALL_TEAMS_ID;
  }
  if (currentTeam && currentTeam.id > APP_CONTEXT_NO_TEAM_ID) {
    return currentTeam.id;
  }
  return undefined;
};

const isValidTeamId = ({
  userTeams,
  includeAllTeams,
  includeNoTeam,
  teamId,
}: {
  userTeams: ITeamSummary[];
  includeAllTeams: boolean;
  includeNoTeam: boolean;
  teamId: number;
}) => {
  if (
    (teamId === APP_CONTEXT_ALL_TEAMS_ID && !includeAllTeams) ||
    (teamId === APP_CONTEXT_NO_TEAM_ID && !includeNoTeam) ||
    !userTeams?.find((t) => t.id === teamId)
  ) {
    return false;
  }
  return true;
};

const coerceAllTeamsId = (s?: string) => {
  // URLs for the app represent "All teams" by the absence of the team id param
  // "All teams" is represented in AppContext with -1 as the team id so empty
  // strings are coerced to -1 by this function
  return s?.length ? parseInt(s, 10) : APP_CONTEXT_ALL_TEAMS_ID;
};

const shouldRedirectToDefaultTeam = ({
  userTeams,
  includeAllTeams,
  includeNoTeam,
  query,
}: {
  userTeams: ITeamSummary[];
  includeAllTeams: boolean;
  includeNoTeam: boolean;
  query: { team_id?: string };
}) => {
  const teamIdString = query?.team_id || "";
  const parsedTeamId = parseInt(teamIdString, 10);

  // redirect non-numeric strings and negative numbers to default (e.g., `/hosts?team_id=-1` should
  // be redirected to `/hosts`)
  if (teamIdString.length && (isNaN(parsedTeamId) || parsedTeamId < 0)) {
    return true;
  }

  // coerce empty string to -1 (i.e. `ALL_TEAMS_ID`) and test again (this ensures that non-global users will be
  // redirected to their default team when they attempt to access the `/hosts` page and also ensures
  // all users are redirected to their default when they attempt to acess non-existent team ids).
  return !isValidTeamId({
    userTeams,
    includeAllTeams,
    includeNoTeam,
    teamId: coerceAllTeamsId(teamIdString),
  });
};

export const useTeamIdParam = ({
  location = { pathname: "", search: "", hash: "", query: {} },
  router,
  includeAllTeams,
  includeNoTeam,
  permittedAccessByTeamRole,
  resetSelectedRowsOnTeamChange = true,
}: {
  location?: {
    pathname: string;
    search: string;
    query: { team_id?: string };
    hash?: string;
  };
  router: InjectedRouter;
  includeAllTeams: boolean;
  includeNoTeam: boolean;
  permittedAccessByTeamRole?: Record<IUserRole, boolean>;
  resetSelectedRowsOnTeamChange?: boolean;
}) => {
  const { hash, pathname, query, search } = location;
  const {
    availableTeams,
    currentTeam: contextTeam,
    currentUser,
    isFreeTier,
    isPremiumTier,
    setCurrentTeam: setContextTeam,
  } = useContext(AppContext);

  const { setResetSelectedRows } = useContext(TableContext);

  const userTeams = useMemo(
    () =>
      getUserTeams({ currentUser, availableTeams, permittedAccessByTeamRole }),
    [availableTeams, currentUser, permittedAccessByTeamRole]
  );

  const defaultTeam = useMemo(
    () =>
      getDefaultTeam({
        currentUser,
        includeAllTeams,
        includeNoTeam,
        userTeams,
      }),
    [currentUser, includeAllTeams, includeNoTeam, userTeams]
  );

  const currentTeam = useMemo(
    () =>
      userTeams?.find((t) => t.id === coerceAllTeamsId(query?.team_id || "")),
    [query?.team_id, userTeams]
  );

  const handleTeamChange = useCallback(
    (teamId: number) => {
      // TODO: This results in a warning that TableProvider is being updated while rendering while
      // rendering a different component (the component that invokes the useTeamIdParam hook).
      // This requires further investigation but is not currently causing any known issues.
      if (resetSelectedRowsOnTeamChange) {
        setResetSelectedRows(true);
      }

      router.replace(
        pathname
          .concat(rebuildQueryStringWithTeamId(search, teamId))
          .concat(hash || "")
      );
    },
    [
      resetSelectedRowsOnTeamChange,
      router,
      pathname,
      search,
      hash,
      setResetSelectedRows,
    ]
  );

  // reconcile router location and redirect to default team as applicable
  let isRouteOk = false;
  if (isFreeTier) {
    // free tier should never have team_id param
    if (query.team_id) {
      handleTeamChange(-1); // remove team_id param from url
    } else {
      isRouteOk = true;
    }
  } else if (isPremiumTier && userTeams?.length && defaultTeam) {
    if (
      shouldRedirectToDefaultTeam({
        includeAllTeams,
        includeNoTeam,
        query,
        userTeams,
      })
    ) {
      handleTeamChange(defaultTeam.id);
    } else {
      isRouteOk = true;
    }
  }

  useEffect(() => {
    if (isRouteOk && currentTeam?.id !== contextTeam?.id) {
      setContextTeam(currentTeam);
    }
  }, [contextTeam?.id, currentTeam, isRouteOk, setContextTeam]);

  return {
    // essentially `currentTeamIdForAppContext`, where -1: all teams, 0: no team, and positive integers: team ids
    currentTeamId: currentTeam?.id,
    currentTeamName: currentTeam?.name,
    currentTeamSummary: currentTeam,
    isAnyTeamSelected: isAnyTeamSelected(currentTeam?.id),
    isRouteOk,
    isTeamAdmin:
      !!currentTeam?.id && permissions.isTeamAdmin(currentUser, currentTeam.id),
    isTeamMaintainer:
      !!currentTeam?.id &&
      permissions.isTeamMaintainer(currentUser, currentTeam.id),
    isTeamMaintainerOrTeamAdmin:
      !!currentTeam?.id &&
      permissions.isTeamMaintainerOrTeamAdmin(currentUser, currentTeam.id),
    isTeamObserver:
      !!currentTeam?.id &&
      permissions.isTeamObserver(currentUser, currentTeam.id),
    isObserverPlus:
      !!currentTeam?.id &&
      !!currentUser &&
      permissions.isObserverPlus(currentUser, currentTeam.id),
    teamIdForApi: getTeamIdForApi({ currentTeam, includeNoTeam }), // for everywhere except AppContext
    userTeams,
    handleTeamChange,
  };
};

export default useTeamIdParam;
