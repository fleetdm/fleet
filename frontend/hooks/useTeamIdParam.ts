import { useCallback, useContext, useEffect, useMemo } from "react";
import { InjectedRouter } from "react-router";
import { findLastIndex, trimStart } from "lodash";

import { AppContext } from "context/app";
import {
  ALL_TEAMS_ID,
  isAnyTeamSelected,
  ITeamSummary,
  NO_TEAM_ID,
} from "interfaces/team";

const getDefaultTeam = (
  availableTeams: ITeamSummary[],
  includeAll: boolean,
  includeNoTeam: boolean
) => {
  let defaultTeam: ITeamSummary | undefined = availableTeams[0]; // TODO(sarah): can this be improved?
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

const isValidTeamId = ({
  availableTeams,
  includeAllTeams,
  includeNoTeam,
  teamId,
}: {
  availableTeams: ITeamSummary[];
  includeAllTeams: boolean;
  includeNoTeam: boolean;
  teamId: number;
}) => {
  if (
    (teamId === ALL_TEAMS_ID && !includeAllTeams) ||
    (teamId === NO_TEAM_ID && !includeNoTeam) ||
    !availableTeams?.find((t) => t.id === teamId)
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
    query: { team_id?: string };
    hash?: string;
  };
  router: InjectedRouter;
  includeAllTeams: boolean;
  includeNoTeam: boolean;
}) => {
  const { hash, pathname, query, search } = location;
  const { currentTeam, availableTeams, setCurrentTeam } = useContext(
    AppContext
  );

  const memoizedDefaultTeam = useMemo(() => {
    if (!availableTeams?.length) {
      return undefined;
    }
    return getDefaultTeam(availableTeams, includeAllTeams, includeNoTeam);
  }, [availableTeams, includeAllTeams, includeNoTeam]);

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

  useEffect(() => {
    if (!availableTeams?.length || !memoizedDefaultTeam) {
      return;
    }

    const teamIdString = query?.team_id || "";
    let parsedTeamId = parseInt(teamIdString, 10);

    // test for negative numbers or non-numeric values and redirect to default
    if (teamIdString.length && (isNaN(parsedTeamId) || parsedTeamId < 0)) {
      handleTeamChange(memoizedDefaultTeam.id);
      return;
    }

    // coerce undefined/empty string to -1
    parsedTeamId = !teamIdString.length ? ALL_TEAMS_ID : parsedTeamId;
    if (
      !isValidTeamId({
        availableTeams,
        includeAllTeams,
        includeNoTeam,
        teamId: parsedTeamId,
      })
    ) {
      handleTeamChange(memoizedDefaultTeam.id);
      return;
    }
    if (parsedTeamId !== currentTeam?.id) {
      setCurrentTeam(availableTeams?.find((t) => t.id === parsedTeamId));
    }
  }, [
    availableTeams,
    currentTeam,
    includeAllTeams,
    includeNoTeam,
    memoizedDefaultTeam,
    query,
    search,
    handleTeamChange,
    setCurrentTeam,
  ]);

  return {
    currentTeamId: currentTeam?.id,
    currentTeamName: currentTeam?.name,
    isAnyTeamSelected: isAnyTeamSelected(currentTeam),
    teamIdForApi: teamIdForApi({ currentTeam, includeNoTeam }),
    handleTeamChange,
  };
};

export default useTeamIdParam;
