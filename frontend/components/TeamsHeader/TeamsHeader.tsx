import React from "react";

import { ITeamSummary } from "interfaces/team";

import TeamsDropdown from "components/TeamsDropdown";

interface ITeamsHeader {
  isOnGlobalTeam?: boolean;
  currentTeamId?: number;
  userTeams?: ITeamSummary[];
  isSandboxMode?: boolean;
  onTeamChange: (teamId: number) => void;
}

const TeamsHeader = ({
  isOnGlobalTeam,
  currentTeamId,
  userTeams = [],
  isSandboxMode = false,
  onTeamChange,
}: ITeamsHeader) => {
  if (userTeams) {
    if (userTeams.length > 1 || isOnGlobalTeam) {
      return (
        <TeamsDropdown
          currentUserTeams={userTeams}
          selectedTeamId={currentTeamId}
          onChange={onTeamChange}
          isSandboxMode={isSandboxMode}
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
