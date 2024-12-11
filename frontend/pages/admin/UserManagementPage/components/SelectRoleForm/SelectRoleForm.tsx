import React, { useState, useContext } from "react";
import { ITeam } from "interfaces/team";
import { UserRole } from "interfaces/user";
import { SingleValue } from "react-select-5";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import { AppContext } from "context/app";
import { roleOptions } from "../../helpers/userManagementHelpers";

interface ISelectRoleFormProps {
  defaultTeamRole: UserRole;
  currentTeam?: ITeam;
  teams: ITeam[];
  onFormChange: (teams: ITeam[]) => void;
  isApiOnly?: boolean;
}

const generateSelectedTeamData = (
  allTeams: ITeam[],
  updatedTeam?: Partial<ITeam>
): ITeam[] => {
  return allTeams.map(
    (teamItem): ITeam => ({
      ...teamItem,
      role: teamItem.id === updatedTeam?.id ? updatedTeam.role! : teamItem.role,
    })
  );
};

const SelectRoleForm = ({
  defaultTeamRole,
  currentTeam,
  teams,
  onFormChange,
  isApiOnly,
}: ISelectRoleFormProps): JSX.Element => {
  const { isPremiumTier } = useContext(AppContext);

  const [selectedRole, setSelectedRole] = useState<CustomOptionType>({
    value: defaultTeamRole.toLowerCase(),
    label: defaultTeamRole,
  });

  const updateSelectedRole = (newRoleValue: SingleValue<CustomOptionType>) => {
    if (newRoleValue) {
      const updatedTeam = {
        ...currentTeam,
        role: newRoleValue.value as UserRole,
      };
      onFormChange(generateSelectedTeamData(teams, updatedTeam));
      setSelectedRole(newRoleValue);
    }
  };

  return (
    <DropdownWrapper
      name="Team role"
      label="Role"
      options={roleOptions({ isPremiumTier, isApiOnly })}
      value={selectedRole}
      onChange={updateSelectedRole}
      isSearchable={false}
    />
  );
};

export default SelectRoleForm;
