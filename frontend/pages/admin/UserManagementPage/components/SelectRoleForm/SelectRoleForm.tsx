import React, { useState } from "react";

import { ITeam } from "interfaces/team";
import { IRole } from "interfaces/role";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";

interface ISelectRoleFormProps {
  defaultTeamRole: string;
  currentTeam: ITeam;
  teams: ITeam[];
  onFormChange: (teams: ITeam[]) => void;
}

const baseClass = "select-role-form";

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

const generateSelectedTeamData = (
  allTeams: ITeam[],
  updatedTeam: ITeam
): ITeam[] => {
  const filtered = allTeams.map(
    (teamItem): ITeam => {
      const teamRole =
        teamItem.id === updatedTeam.id ? updatedTeam.role : teamItem.role;
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

const SelectRoleForm = (props: ISelectRoleFormProps): JSX.Element => {
  const { defaultTeamRole, currentTeam, teams, onFormChange } = props;

  const [selectedRole, setSelectedRole] = useState<string>(
    defaultTeamRole.toLowerCase()
  );

  const updateSelectedRole = (newRoleValue: string) => {
    const updatedTeam = { ...currentTeam };

    updatedTeam.role = newRoleValue;

    onFormChange(generateSelectedTeamData(teams, updatedTeam));

    setSelectedRole(newRoleValue);
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__select-role`}>
        <Dropdown
          value={selectedRole}
          className={`${baseClass}__role-dropdown`}
          options={roles}
          searchable={false}
          onChange={(newRoleValue: string) => updateSelectedRole(newRoleValue)}
          testId={`${name}-checkbox`}
        />
      </div>
    </div>
  );
};

export default SelectRoleForm;
