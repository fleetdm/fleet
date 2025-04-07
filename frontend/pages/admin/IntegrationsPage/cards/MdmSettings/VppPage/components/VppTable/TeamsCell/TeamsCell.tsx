import React from "react";
import classnames from "classnames";

import { ITokenTeam } from "interfaces/mdm";

import TextCell from "components/TableContainer/DataTable/TextCell";
import { uniqueId } from "lodash";
import ReactTooltip from "react-tooltip";

const baseClass = "teams-cell";

const NUM_TEAMS_IN_TOOLTIP = 3;

const generateCell = (teams: ITokenTeam[] | null) => {
  if (!teams) {
    return <TextCell value="---" grey />;
  }

  if (teams.length === 0) {
    return <TextCell value="All teams" />;
  }

  let text = "";
  let italicize = true;
  if (teams.length === 1) {
    italicize = false;
    text = teams[0].name;
  } else {
    text = `${teams.length} teams`;
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

const generateTooltip = (teams: ITokenTeam[] | null, tooltipId: string) => {
  if (teams === null || teams.length <= 1) {
    return null;
  }

  const condensedTeams = condenseTeams(teams);

  return (
    <ReactTooltip
      effect="solid"
      backgroundColor="#3e4771"
      id={tooltipId}
      data-html
    >
      <ul className={`${baseClass}__team-list`}>
        {condensedTeams.map((teamName) => {
          return <li key={teamName}>{teamName}</li>;
        })}
      </ul>
    </ReactTooltip>
  );
};

interface ITeamsCellProps {
  teams: ITokenTeam[] | null;
  className?: string;
}

const TeamsCell = ({ teams, className }: ITeamsCellProps) => {
  const tooltipId = uniqueId();
  const classNames = classnames(baseClass, className);

  if (!teams) {
    return <TextCell value={teams} />;
  }

  if (teams.length === 0) {
    return <TextCell value="All teams" />;
  }

  if (teams.length === 1) {
    return <TextCell value={teams[0].name} />;
  }

  const cell = generateCell(teams);
  const tooltip = generateTooltip(teams, tooltipId);

  return (
    <>
      <div
        className={`${baseClass}__team-text-with-tooltip`}
        data-tip
        data-for={tooltipId}
      >
        {cell}
      </div>
      {tooltip}
    </>
  );
};

export default TeamsCell;
