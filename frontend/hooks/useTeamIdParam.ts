import { useCallback, useContext, useEffect, useMemo } from "react";
import { InjectedRouter } from "react-router";
import { findLastIndex, trimStart } from "lodash";

import { AppContext } from "context/app";
import {
  API_NO_TEAM_ID,
  API_ALL_TEAMS_ID,
  APP_CONTEXT_ALL_TEAMS_ID,
  APP_CONTEXT_NO_TEAM_ID,
  isAnyTeamSelected,
  ITeamSummary,
  ITeam,
} from "interfaces/team";
import { IUserRole } from "interfaces/user";
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
  const teamIndex = parts.findIndex((p) => p.startsWith("team_id="));

  // URLs for the app represent "All teams" by the absence of the team id param
  const newTeamPart =
    newTeamId > APP_CONTEXT_ALL_TEAMS_ID ? `team_id=${newTeamId}` : "";

  if (teamIndex < 0) {
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

const getDefaultTeam = ({
  userTeams,
  includeAllTeams,
  includeNoTeam,
}: {
  userTeams: ITeamSummary[];
  includeAllTeams: boolean;
  includeNoTeam: boolean;
}) => {
  let defaultTeam: ITeamSummary | undefined;
  if (includeAllTeams) {
    defaultTeam =
      userTeams.find((t) => t.id === APP_CONTEXT_ALL_TEAMS_ID) || defaultTeam;
  } else if (includeNoTeam) {
    defaultTeam =
      userTeams.find((t) => t.id === APP_CONTEXT_NO_TEAM_ID) || defaultTeam;
  } else {
    defaultTeam =
      userTeams.find((t) => t.id > APP_CONTEXT_NO_TEAM_ID) || defaultTeam;
  }
  return defaultTeam;
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
  defaultTeamId: number;
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
}) => {
  const { hash, pathname, query, search } = location;
  const {
    availableTeams,
    currentTeam: contextTeam,
    currentUser,
    isOnGlobalTeam,
    setCurrentTeam: setContextTeam,
  } = useContext(AppContext);

  const memoizedUserTeams = useMemo(() => {
    if (!currentUser || !availableTeams?.length) {
      return undefined;
    }
    return isOnGlobalTeam
      ? availableTeams
      : filterUserTeamsByRole(currentUser.teams, permittedAccessByTeamRole);
  }, [availableTeams, currentUser, isOnGlobalTeam, permittedAccessByTeamRole]);

  const memoizedDefaultTeam = useMemo(() => {
    if (!memoizedUserTeams?.length) {
      return undefined;
    }
    return getDefaultTeam({
      userTeams: memoizedUserTeams,
      includeAllTeams,
      includeNoTeam,
    });
  }, [includeAllTeams, includeNoTeam, memoizedUserTeams]);

  const handleTeamChange = useCallback(
    (teamId: number) => {
      router.replace(
        pathname
          .concat(rebuildQueryStringWithTeamId(search, teamId))
          .concat(hash || "")
      );
    },
    [pathname, search, hash, router]
  );

  let isRouteOk = false;
  if (memoizedUserTeams?.length && memoizedDefaultTeam) {
    // first reconcile router location and redirect to default team as applicable
    if (
      shouldRedirectToDefaultTeam({
        userTeams: memoizedUserTeams,
        defaultTeamId: memoizedDefaultTeam.id,
        includeAllTeams,
        includeNoTeam,
        query,
      })
    ) {
      handleTeamChange(memoizedDefaultTeam.id);
    } else {
      isRouteOk = true;
    }
  }

  const foundTeam = memoizedUserTeams?.find(
    (t) => t.id === coerceAllTeamsId(query?.team_id || "")
  );

  useEffect(() => {
    if (isRouteOk && foundTeam?.id !== contextTeam?.id) {
      setContextTeam(foundTeam);
    }
  }, [contextTeam?.id, foundTeam, isRouteOk, setContextTeam]);

  return {
    currentTeamId: foundTeam?.id,
    currentTeamName: foundTeam?.name,
    currentTeamSummary: foundTeam
      ? { id: foundTeam.id, name: foundTeam.name }
      : undefined,
    isAnyTeamSelected: isAnyTeamSelected(foundTeam?.id),
    isRouteOk,
    isTeamAdmin:
      !!foundTeam?.id && permissions.isTeamAdmin(currentUser, foundTeam.id),
    isTeamMaintainer:
      !!foundTeam?.id &&
      permissions.isTeamMaintainer(currentUser, foundTeam.id),
    isTeamMaintainerOrTeamAdmin:
      !!foundTeam?.id &&
      permissions.isTeamMaintainerOrTeamAdmin(currentUser, foundTeam.id),
    isTeamObserver:
      !!foundTeam?.id && permissions.isTeamObserver(currentUser, foundTeam.id),
    teamIdForApi: getTeamIdForApi({ currentTeam: foundTeam, includeNoTeam }),
    userTeams: memoizedUserTeams,
    handleTeamChange,
  };
};

export default useTeamIdParam;
