import React from "react";
import classnames from "classnames";

import { ITokenTeam } from "interfaces/mdm";

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
    text = teams[0].name;
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
        .map((team) => team.name)
        .reverse()) ||
    [];

  return teams.length > NUM_TEAMS_IN_TOOLTIP
    ? condensed.concat(`+${teams.length - NUM_TEAMS_IN_TOOLTIP} more`)
    : condensed;
};

const generateTooltipContent = (teams: ITokenTeam[]) => {
  const condensedTeams = condenseTeams(teams);

  return (
    <ul className={`${baseClass}__team-list`}>
      {condensedTeams.map((teamName) => (
        <li key={teamName}>{teamName}</li>
      ))}
    </ul>
  );
};

interface ITeamsCellProps {
  teams: ITokenTeam[] | null;
  className?: string;
}

const TeamsCell = ({ teams, className }: ITeamsCellProps) => {
  const classNames = classnames(baseClass, className);

  if (!teams) {
    return <TextCell value={teams} />;
  }

  if (teams.length === 0) {
    return <TextCell value="All fleets" />;
  }

  if (teams.length === 1) {
    return <TextCell value={teams[0].name} />;
  }

  const cell = generateCell(teams);

  return (
    <TooltipWrapper
      showArrow
      tipContent={generateTooltipContent(teams)}
      underline={false}
      className={classNames}
    >
      {cell}
    </TooltipWrapper>
  );
};

export default TeamsCell;
