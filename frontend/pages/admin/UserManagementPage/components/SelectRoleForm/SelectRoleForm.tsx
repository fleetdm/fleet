import React, { useState, useContext } from "react";

import { ITeam } from "interfaces/team";
import { IRole } from "interfaces/role";
import { UserRole } from "interfaces/user";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import { AppContext } from "context/app";

interface ISelectRoleFormProps {
  defaultTeamRole: UserRole;
  currentTeam?: ITeam;
  teams: ITeam[];
  onFormChange: (teams: ITeam[]) => void;
  label: string | string[];
}

const baseClass = "select-role-form";

const roleOptions = (isPremiumTier: boolean): IRole[] => {
  const roles: IRole[] = [
    {
      disabled: false,
      label: "Observer",
      value: "observer",
    },
    {
      disabled: false,
      label: "Maintainer",
      value: "maintainer",
    },
    {
      disabled: false,
      label: "Admin",
      value: "admin",
    },
  ];

  if (isPremiumTier) {
    roles.unshift({
      disabled: false,
      label: "Observer+",
      value: "observer_plus",
    });
  }

  return roles;
};

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
}: ISelectRoleFormProps): JSX.Element => {
  const { isPremiumTier } = useContext(AppContext);

  const [selectedRole, setSelectedRole] = useState(
    defaultTeamRole.toLowerCase()
  );

  const updateSelectedRole = (newRoleValue: UserRole) => {
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
          options={roleOptions(isPremiumTier || false)}
          searchable={false}
          onChange={(newRoleValue: UserRole) =>
            updateSelectedRole(newRoleValue)
          }
          testId={`${name}-checkbox`}
        />
      </div>
    </div>
  );
};

export default SelectRoleForm;
