import React, { useContext, useMemo } from "react";
import classnames from "classnames";
import { ITeamSummary } from "interfaces/team";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import { AppContext } from "context/app";

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

  if (includeAll) {
    options.unshift({
      disabled: false,
      label: "All teams",
      value: 0,
    });
  }
  if (includeNoTeams) {
    options.unshift({
      disabled: false,
      label: "No team",
      value: 0,
    });
  }
  return options;
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
  isDisabled,
  onChange,
  onOpen,
  onClose,
}: ITeamsDropdownProps): JSX.Element => {
  const { isOnGlobalTeam = false } = useContext(AppContext);

  const teamOptions = useMemo(
    () =>
      generateDropdownOptions(
        currentUserTeams,
        includeAll && isOnGlobalTeam,
        includeNoTeams
      ),
    [currentUserTeams, includeAll, isOnGlobalTeam, includeNoTeams]
  );

  const selectedValue = teamOptions.find(
    (option) => selectedTeamId === option.value
  )
    ? selectedTeamId
    : teamOptions[0]?.value;

  const dropdownWrapperClasses = classnames(`${baseClass}-wrapper`, {
    disabled: isDisabled || undefined,
  });

  return (
    <div className={dropdownWrapperClasses}>
      {teamOptions.length && (
        <Dropdown
          value={selectedValue}
          placeholder="All teams"
          className={baseClass}
          options={teamOptions}
          searchable={false}
          disabled={isDisabled || false}
          onChange={onChange}
          onOpen={onOpen}
          onClose={onClose}
        />
      )}
    </div>
  );
};

export default TeamsDropdown;
