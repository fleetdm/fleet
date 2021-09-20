import React, {
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useQuery, useMutation } from "react-query";
import { isEmpty } from "lodash";
import { AppContext } from "context/app";

import { ITeam } from "interfaces/team";

import sortUtils from "utilities/sort";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";

const sortDropdownOptions = (
  teams: ITeam[] | undefined,
  includeAll: boolean | undefined
) => {
  if (!teams) {
    return [];
  }
  const sortedOptions = teams
    .map((team) => ({
      disabled: false,
      label: team.name,
      value: team.id,
    }))
    .sort((a, b) => sortUtils.caseInsensitiveAsc(b.label, a.label));
  if (includeAll) {
    sortedOptions.unshift({
      disabled: false,
      label: "All teams",
      value: 0,
    });
  }
  return sortedOptions;
};

const TeamsDropdown = (dropdownProps: {
  currentUserTeams: ITeam[] | undefined;
  onChange: (id: number) => void;
  selectedTeam: number;
}): JSX.Element | null => {
  const { currentUserTeams, onChange, selectedTeam } = dropdownProps;
  const {
    currentUser,
    isGlobalAdmin,
    isGlobalMaintainer,
    isOnGlobalTeam,
    isOnlyObserver,
    isPremiumTier,
  } = useContext(AppContext);

  const dropdownOptions = useMemo(
    () => sortDropdownOptions(currentUserTeams, isOnGlobalTeam),
    [currentUserTeams, isOnGlobalTeam]
  );

  if (isEmpty(currentUserTeams)) {
    return <h1>Policies</h1>;
  }

  return (
    <div>
      <Dropdown
        value={selectedTeam}
        placeholder={"All teams"}
        className="teams-dropdown"
        options={dropdownOptions}
        searchable={false}
        onChange={onChange}
      />
      {/* <h1>Policies</h1> */}
    </div>
  );
};
export default TeamsDropdown;
