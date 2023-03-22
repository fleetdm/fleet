import { AppContext } from "context/app";
import {
  ALL_TEAMS_ID,
  isAnyTeamSelected,
  ITeamSummary,
  NO_TEAM_ID,
  parseTeamIdParam,
  teamIdParamFromUrlSearch,
} from "interfaces/team";
import { findLastIndex, trimStart } from "lodash";
import { useCallback, useContext, useEffect, useMemo } from "react";
import { InjectedRouter } from "react-router";
import { QueryParams } from "utilities/url";

const getDefaultTeam = (
  availableTeams: ITeamSummary[],
  includeAll: boolean,
  includeNoTeam: boolean
) => {
  let defaultTeam: ITeamSummary | undefined = availableTeams[0]; // TODO: fix this
  if (includeAll) {
    defaultTeam =
      availableTeams.find((t) => t.id === ALL_TEAMS_ID) || defaultTeam;
  } else if (includeNoTeam) {
    defaultTeam =
      availableTeams.find((t) => t.id === NO_TEAM_ID) || defaultTeam;
  } else {
    defaultTeam = availableTeams.find((t) => t.id > NO_TEAM_ID) || defaultTeam;
  }
  return defaultTeam;
};

const teamIdForApi = ({
  currentTeam,
  includeNoTeam = false,
}: {
  currentTeam?: ITeamSummary;
  includeNoTeam?: boolean;
}) => {
  if (includeNoTeam && currentTeam?.id === NO_TEAM_ID) {
    return NO_TEAM_ID;
  }

  if (currentTeam && currentTeam.id > NO_TEAM_ID) {
    return currentTeam.id;
  }

  return undefined;
};

const splitQueryStringParts = (queryString: string) =>
  trimStart(queryString, "?")
    .split("&")
    .filter((p) => p.includes("="));

const rebuildQueryStringWithTeamId = (
  queryString: string,
  newTeamId: number
) => {
  const parts = splitQueryStringParts(queryString);

  const teamIndex = parts.findIndex((p) => p.includes("team_id"));

  if (
    teamIndex >= 0 &&
    teamIndex !== findLastIndex(parts, (p) => p.includes("team_id"))
  ) {
    console.warn(
      `URL contains more than one team_id parameter: ${queryString}`
    );
  }

  if (newTeamId === ALL_TEAMS_ID) {
    // remove old team param if any
    teamIndex >= 0 && parts.splice(teamIndex, 1);
  } else {
    const teamPart = `team_id=${newTeamId}`;
    teamIndex >= 0
      ? parts.splice(teamIndex, 1, teamPart) // replace old param
      : parts.push(teamPart); // add new param
  }
  console.log("newQueryParams", parts);

  return !parts.length ? "" : `?${parts.join("&")}`;
};

export const curryRouterReplaceTeamId = ({
  pathname,
  search,
  hash = "",
  router,
}: {
  pathname: string;
  search: string;
  hash?: string;
  router: InjectedRouter;
}) => (teamId: number) => {
  console.log("routerReplaceTeamId called", teamId, pathname, search);
  router.replace(
    pathname.concat(rebuildQueryStringWithTeamId(search, teamId)).concat(hash)
  );
};

export const isValidCurrentTeam = ({
  currentTeam,
  availableTeams,
  includeAllTeams,
  includeNoTeam,
}: {
  currentTeam?: ITeamSummary;
  availableTeams: ITeamSummary[];
  includeAllTeams: boolean;
  includeNoTeam: boolean;
}) => {
  if (
    currentTeam === undefined ||
    (currentTeam.id === ALL_TEAMS_ID && !includeAllTeams) ||
    (currentTeam.id === NO_TEAM_ID && !includeNoTeam) ||
    !availableTeams?.find((t) => t.id === currentTeam.id)
  ) {
    return false;
  }
  return true;
};

export const useTeamIdParam = ({
  location = { pathname: "", search: "", hash: "", query: {} },
  router,
  includeAllTeams,
  includeNoTeam,
}: {
  location?: {
    pathname: string;
    search: string;
    query: QueryParams;
    hash?: string;
  };
  router: InjectedRouter;
  includeAllTeams: boolean;
  includeNoTeam: boolean;
}) => {
  const { hash, pathname, query, search } = location;
  const { currentTeam, availableTeams } = useContext(AppContext);

  const memoizedIsAnyTeamSelected = useMemo(
    () => isAnyTeamSelected(currentTeam),
    [currentTeam]
  );

  const memoizedTeamIdFromLocation = useMemo(
    () => parseTeamIdParam(teamIdParamFromUrlSearch(search)),
    [search]
  );

  const memoizedTeamIdForApi = useMemo(
    () => teamIdForApi({ currentTeam, includeNoTeam }),
    [currentTeam, includeNoTeam]
  );

  const memoizedDefaultTeam = useMemo(() => {
    if (!availableTeams?.length) {
      return undefined;
    }
    return getDefaultTeam(availableTeams, includeAllTeams, includeNoTeam);
  }, [availableTeams, includeAllTeams, includeNoTeam]);

  const handleTeamSelect = useCallback(
    (teamId: number) => {
      curryRouterReplaceTeamId({ pathname, search, hash, router })(teamId);
    },
    [pathname, search, hash, router]
  );

  useEffect(() => {
    console.log("useEffect: validating currentTeam inside useTeamIdParam");
    if (!availableTeams?.length || !memoizedDefaultTeam) {
      console.log("skipping, not ready");
      return;
    }
    if (
      currentTeam?.id === memoizedDefaultTeam.id &&
      memoizedTeamIdFromLocation === memoizedDefaultTeam.id
    ) {
      console.log("skipping, location already default team");
      return;
    }

    if (
      isValidCurrentTeam({
        currentTeam,
        availableTeams,
        includeAllTeams,
        includeNoTeam,
      })
    ) {
      console.log("validated currentTeam");
      return;
    }
    console.log("invalid currentTeam, switching to defaultTeam");
    handleTeamSelect(memoizedDefaultTeam.id);
  }, [
    availableTeams,
    currentTeam,
    includeAllTeams,
    includeNoTeam,
    memoizedDefaultTeam,
    memoizedTeamIdFromLocation,
    pathname,
    query,
    router,
    search,
    handleTeamSelect,
  ]);

  return {
    currentTeamId: currentTeam?.id,
    currentTeamName: currentTeam?.name,
    defaultTeam: memoizedDefaultTeam,
    isAnyTeamSelected: memoizedIsAnyTeamSelected,
    teamIdForApi: memoizedTeamIdForApi,
    handleTeamSelect,
  };
};

export default useTeamIdParam;
