import React from "react";

import { ITeamSummary } from "interfaces/team";

import FleetsDropdown from "components/FleetsDropdown";

interface ITeamsHeader {
  isOnGlobalTeam?: boolean;
  currentTeamId?: number;
  userTeams?: ITeamSummary[];
  onTeamChange: (teamId: number) => void;
}

const TeamsHeader = ({
  isOnGlobalTeam,
  currentTeamId,
  userTeams = [],
  onTeamChange,
}: ITeamsHeader) => {
  if (userTeams) {
    if (userTeams.length > 1 || isOnGlobalTeam) {
      return (
        <FleetsDropdown
          currentUserTeams={userTeams}
          selectedFleetId={currentTeamId}
          onChange={onTeamChange}
          includeUnassigned
        />
      );
    }
    if (userTeams.length === 1 && !isOnGlobalTeam) {
      return <h1>{userTeams[0].name}</h1>;
    }
  }
  return <></>;
};

export default TeamsHeader;
