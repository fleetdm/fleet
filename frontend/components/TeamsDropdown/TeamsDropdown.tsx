import React, { useContext, useMemo } from "react";
import classnames from "classnames";
import { AppContext } from "context/app";
import { ITeam } from "interfaces/team";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";

const generateDropdownOptions = (
  teams: ITeam[] | undefined,
  includeAll: boolean | undefined
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

  return options;
};

interface ITeamsDropdownProps {
  currentUserTeams: ITeam[];
  selectedTeamId: number;
  includeAll?: boolean; // Include "All Teams" option for all users
  disableAll?: boolean; // Disable "All Teams" option for global users
  isDisabled?: boolean;
  onChange: (newSelectedValue: number) => void;
  onOpen?: () => void;
  onClose?: () => void;
}

const baseClass = "component__team-dropdown";

const TeamsDropdown = ({
  currentUserTeams,
  selectedTeamId,
  includeAll = false,
  disableAll = false,
  isDisabled,
  onChange,
  onOpen,
  onClose,
}: ITeamsDropdownProps): JSX.Element => {
  const { isOnGlobalTeam } = useContext(AppContext);

  const teamOptions = useMemo(
    () =>
      generateDropdownOptions(
        currentUserTeams,
        (isOnGlobalTeam && !disableAll) || includeAll
      ),
    [currentUserTeams, isOnGlobalTeam]
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
