import React, { useContext, useMemo } from "react";
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
  includeAll?: boolean;
  onChange: (newSelectedValue: number) => void;
  onOpen?: () => void;
  onClose?: () => void;
}

const baseClass = "component__team-dropdown";

const TeamsDropdown = ({
  currentUserTeams,
  selectedTeamId,
  includeAll,
  onChange,
  onOpen,
  onClose,
}: ITeamsDropdownProps): JSX.Element => {
  const { isOnGlobalTeam } = useContext(AppContext);

  const teamOptions = useMemo(
    () =>
      generateDropdownOptions(currentUserTeams, isOnGlobalTeam || includeAll),
    [currentUserTeams, isOnGlobalTeam]
  );

  const selectedValue = teamOptions.find(
    (option) => selectedTeamId === option.value
  )
    ? selectedTeamId
    : teamOptions[0]?.value;

  return (
    <div>
      {teamOptions.length && (
        <Dropdown
          value={selectedValue}
          placeholder="All teams"
          className={baseClass}
          options={teamOptions}
          searchable={false}
          onChange={onChange}
          onOpen={onOpen}
          onClose={onClose}
        />
      )}
    </div>
  );
};

export default TeamsDropdown;
