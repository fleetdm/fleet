import React from "react";

import { ITokenTeam } from "interfaces/mdm";
import { getTeamDisplayName } from "interfaces/team";

import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "teams-cell";

const NUM_TEAMS_IN_TOOLTIP = 3;

const generateCell = (teams: ITokenTeam[] | null) => {
  if (!teams) {
    return <TextCell value="---" grey />;
  }

  if (teams.length === 0) {
    return <TextCell value="All fleets" />;
  }

  let text = "";
  let italicize = true;
  if (teams.length === 1) {
    italicize = false;
    text = getTeamDisplayName(teams[0]);
  } else {
    text = `${teams.length} fleets`;
  }

  return <TextCell value={text} italic={italicize} />;
};

const condenseTeams = (teams: ITokenTeam[]) => {
  const condensed =
    (teams?.length &&
      teams
        .slice(-NUM_TEAMS_IN_TOOLTIP)
        .map((team) => getTeamDisplayName(team))
        .reverse()) ||
    [];

  return teams.length > NUM_TEAMS_IN_TOOLTIP
    ? condensed.concat(`+${teams.length - NUM_TEAMS_IN_TOOLTIP} more`)
    : condensed;
};

interface ITeamsCellProps {
  teams: ITokenTeam[] | null;
}

const TeamsCell = ({ teams }: ITeamsCellProps) => {
  if (!teams) {
    return <TextCell value={teams} />;
  }

  if (teams.length === 0) {
    return <TextCell value="All fleets" />;
  }

  if (teams.length === 1) {
    return <TextCell value={getTeamDisplayName(teams[0])} />;
  }

  const cell = generateCell(teams);
  const condensedTeams = condenseTeams(teams);

  return (
    <TooltipWrapper
      tipContent={
        <ul className={`${baseClass}__team-list`}>
          {condensedTeams.map((teamName) => {
            return <li key={teamName}>{teamName}</li>;
          })}
        </ul>
      }
      underline={false}
      showArrow
      position="top"
      className={`${baseClass}__team-text-with-tooltip`}
      tipOffset={8}
    >
      {cell}
    </TooltipWrapper>
  );
};

export default TeamsCell;
