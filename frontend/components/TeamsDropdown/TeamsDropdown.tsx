import React, { useMemo } from "react";
import classnames from "classnames";
import {
  APP_CONTEXT_ALL_TEAMS_SUMMARY,
  ITeamSummary,
  APP_CONTEX_NO_TEAM_SUMMARY,
} from "interfaces/team";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import ReactTooltip from "react-tooltip";
import { uniqueId } from "lodash";

const generateDropdownOptions = (
  teams: ITeamSummary[] | undefined,
  includeAll: boolean,
  includeNoTeams?: boolean
) => {
  if (!teams) {
    return [];
  }

  const options = teams.map((team) => ({
    disabled: false,
    label: team.name,
    value: team.id,
  }));

  const filtered = options.filter(
    (o) =>
      !(
        (o.label === APP_CONTEX_NO_TEAM_SUMMARY.name && !includeNoTeams) ||
        (o.label === APP_CONTEXT_ALL_TEAMS_SUMMARY.name && !includeAll)
      )
  );

  return filtered;
};
interface ITeamsDropdownProps {
  currentUserTeams: ITeamSummary[];
  selectedTeamId?: number;
  includeAll?: boolean; // Include the "All Teams" option;
  includeNoTeams?: boolean;
  isDisabled?: boolean;
  onChange: (newSelectedValue: number) => void;
  onOpen?: () => void;
  onClose?: () => void;
}

const baseClass = "component__team-dropdown";

const TeamsDropdown = ({
  currentUserTeams,
  selectedTeamId,
  includeAll = true,
  includeNoTeams = false,
  isDisabled = false,
  onChange,
  onOpen,
  onClose,
}: ITeamsDropdownProps): JSX.Element => {
  const teamOptions = useMemo(
    () => generateDropdownOptions(currentUserTeams, includeAll, includeNoTeams),
    [currentUserTeams, includeAll, includeNoTeams]
  );

  const selectedValue = teamOptions.find(
    (option) => selectedTeamId === option.value
  )
    ? selectedTeamId
    : teamOptions[0]?.value;

  const dropdownWrapperClasses = classnames(`${baseClass}-wrapper`, {
    disabled: isDisabled || undefined,
  });

  const renderDropdown = () => {
    if (teamOptions.length) {
      return (
        <Dropdown
          value={selectedValue}
          placeholder="All teams"
          className={baseClass}
          options={teamOptions}
          searchable={false}
          disabled={isDisabled}
          onChange={onChange}
          onOpen={onOpen}
          onClose={onClose}
        />
      );
    }
  };

  return <div className={dropdownWrapperClasses}>{renderDropdown()}</div>;
};

export default TeamsDropdown;
