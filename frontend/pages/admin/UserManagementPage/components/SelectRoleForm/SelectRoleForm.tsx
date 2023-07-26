import React, { useState, useContext } from "react";

import { ITeam } from "interfaces/team";
import { UserRole } from "interfaces/user";
import { Role } from "interfaces/role";

// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import { AppContext } from "context/app";
import { roleOptions } from "../../helpers/userManagementHelpers";

interface ISelectRoleFormProps {
  defaultTeamRole: UserRole;
  currentTeam?: ITeam;
  teams: ITeam[];
  onFormChange: (teams: ITeam[]) => void;
  label: string | string[];
  isApiOnly?: boolean;
}

const baseClass = "select-role-form";

const generateSelectedTeamData = (
  allTeams: ITeam[],
  updatedTeam?: any
): ITeam[] => {
  const filtered = allTeams.map(
    (teamItem): ITeam => {
      const teamRole =
        teamItem.id === updatedTeam?.id ? updatedTeam.role : teamItem.role;
      return {
        description: teamItem.description,
        id: teamItem.id,
        host_count: teamItem.host_count,
        user_count: teamItem.user_count,
        name: teamItem.name,
        role: teamRole,
      };
    }
  );
  return filtered;
};

const SelectRoleForm = ({
  defaultTeamRole,
  currentTeam,
  teams,
  onFormChange,
  label,
  isApiOnly,
}: ISelectRoleFormProps): JSX.Element => {
  const { isPremiumTier } = useContext(AppContext);

  const [selectedRole, setSelectedRole] = useState(
    defaultTeamRole.toLowerCase()
  );

  const updateSelectedRole = (newRoleValue: Role) => {
    const updatedTeam = { ...currentTeam };

    updatedTeam.role = newRoleValue;

    onFormChange(generateSelectedTeamData(teams, updatedTeam));

    setSelectedRole(newRoleValue);
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__select-role`}>
        <Dropdown
          label={label}
          value={selectedRole}
          className={`${baseClass}__role-dropdown`}
          options={roleOptions({ isPremiumTier, isApiOnly })}
          searchable={false}
          onChange={(newRoleValue: Role) => updateSelectedRole(newRoleValue)}
          testId={`${name}-checkbox`}
        />
      </div>
    </div>
  );
};

export default SelectRoleForm;
