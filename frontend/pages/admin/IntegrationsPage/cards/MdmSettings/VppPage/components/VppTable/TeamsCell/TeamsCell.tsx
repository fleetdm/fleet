import React from "react";
import classnames from "classnames";

import { ITokenTeam } from "interfaces/mdm";
import { APP_CONTEXT_NO_TEAM_ID, APP_CONTEXT_NO_TEAM_SUMMARY } from "interfaces/team";

import TextCell from "components/TableContainer/DataTable/TextCell";
import { uniqueId } from "lodash";
import ReactTooltip from "react-tooltip";

const baseClass = "teams-cell";

const NUM_TEAMS_IN_TOOLTIP = 3;

/** Returns the display name for a token team, normalizing "No team" (team_id 0)
 *  to "Unassigned" to match Fleet's UI convention. */
const getTeamDisplayName = (team: ITokenTeam): string => {
  if (team.team_id === APP_CONTEXT_NO_TEAM_ID) {
    return APP_CONTEXT_NO_TEAM_SUMMARY.name;
  }
  return team.name;
};

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
    return <TextCell value="All fleets" />;
  }

  if (teams.length === 1) {
    return <TextCell value={getTeamDisplayName(teams[0])} />;
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
