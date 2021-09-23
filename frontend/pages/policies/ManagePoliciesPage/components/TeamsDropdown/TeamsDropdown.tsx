import React, { useContext, useMemo } from "react";
import { isEmpty } from "lodash";
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

const TeamsDropdown = (dropdownProps: {
  currentUserTeams: ITeam[];
  onChange: (id: number) => void;
  selectedTeam: number;
}): JSX.Element => {
  const { currentUserTeams, onChange, selectedTeam } = dropdownProps;
  const { isOnGlobalTeam } = useContext(AppContext);

  const dropdownOptions = useMemo(
    () => generateDropdownOptions(currentUserTeams, isOnGlobalTeam),
    [currentUserTeams, isOnGlobalTeam]
  );

  const selectedValue = dropdownOptions.find(
    (option) => selectedTeam === option.value
  )
    ? selectedTeam
    : dropdownOptions[0]?.value;

  return isEmpty(currentUserTeams) ? (
    <h1>Policies</h1>
  ) : (
    <div>
      <Dropdown
        value={selectedValue}
        placeholder={"All teams"}
        className="teams-dropdown"
        options={dropdownOptions}
        searchable={false}
        onChange={onChange}
      />
    </div>
  );
};
export default TeamsDropdown;
