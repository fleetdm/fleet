import React, { useContext } from "react";
import { AppContext } from "context/app";

import { ITeam } from "interfaces/team";
import {
  generateTeamFilterDropdownOptions,
  getValidatedTeamId,
} from "fleet/helpers";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";

interface ITeamsDropdownProps {
  isLoading: boolean;
  teams: ITeam[];
  currentTeamId: number;
  hideAllTeamsOption?: boolean;
  onChange: (newSelectedValue: number) => void;
  onOpen?: () => void;
  onClose?: () => void;
}

const baseClass = "component__team-dropdown";

const TeamsDropdown = ({
  isLoading,
  teams,
  currentTeamId,
  hideAllTeamsOption = false,
  onChange,
  onOpen,
  onClose,
}: ITeamsDropdownProps) => {
  const { currentUser, isPremiumTier, isOnGlobalTeam } = useContext(AppContext);

  if (isLoading) {
    return null;
  } else if (!isPremiumTier) {
    return <h1>Hosts</h1>;
  }

  const teamOptions = generateTeamFilterDropdownOptions(
    teams,
    currentUser,
    isOnGlobalTeam as boolean,
    hideAllTeamsOption
  );
  const selectedTeamId = getValidatedTeamId(
    teams,
    currentTeamId,
    currentUser,
    isOnGlobalTeam as boolean
  );

  return (
    <div>
      <Dropdown
        value={selectedTeamId}
        placeholder="All teams"
        className={baseClass}
        options={teamOptions}
        searchable={false}
        onChange={onChange}
        onOpen={onOpen}
        onClose={onClose}
      />
    </div>
  );
};

export default TeamsDropdown;
