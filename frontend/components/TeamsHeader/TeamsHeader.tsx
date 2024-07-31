import React from "react";

import { ITeamSummary } from "interfaces/team";

import TeamsDropdown from "components/TeamsDropdown";

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
        <TeamsDropdown
          currentUserTeams={userTeams}
          selectedTeamId={currentTeamId}
          onChange={onTeamChange}
          includeNoTeams
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
