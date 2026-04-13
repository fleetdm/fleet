import React, { FormEvent, useState, useContext } from "react";

import { AppContext } from "context/app";
import { ITeam } from "interfaces/team";
import { IUserFormErrors, UserRole } from "interfaces/user";

import { SingleValue } from "react-select-5";
import Button from "components/buttons/Button";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import validatePresence from "components/forms/validators/validate_presence";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Radio from "components/forms/fields/Radio";
import SelectedTeamsForm from "../SelectedTeamsForm/SelectedTeamsForm";
import ApiAccessSection from "../ApiAccessSection";
import { roleOptions } from "../../helpers/userManagementHelpers";

const baseClass = "api-user-form";

export interface IApiUserFormData {
  name: string;
  global_role: UserRole | null;
  teams: ITeam[];
  api_only: true;
  selectedEndpointKeys?: string[];
}

interface IApiUserFormProps {
  onCancel: () => void;
  onSubmit: (formData: IApiUserFormData) => void;
  availableTeams: ITeam[];
  isNewUser?: boolean;
  defaultName?: string;
  defaultGlobalRole?: UserRole | null;
  defaultTeams?: ITeam[];
  defaultSelectedEndpointKeys?: string[];
  formErrors?: IUserFormErrors;
  isSubmitting?: boolean;
}

enum UserTeamType {
  GlobalUser = "GLOBAL_USER",
  AssignTeams = "ASSIGN_TEAMS",
}

const ApiUserForm = ({
  onCancel,
  onSubmit,
  availableTeams,
  isNewUser = false,
  defaultName = "",
  defaultGlobalRole = "gitops",
  defaultTeams = [],
  defaultSelectedEndpointKeys = [],
  formErrors: ancestorErrors = {},
  isSubmitting = false,
}: IApiUserFormProps) => {
  const { isPremiumTier } = useContext(AppContext);

  const [name, setName] = useState(defaultName);
  const [globalRole, setGlobalRole] = useState<UserRole | null>(
    defaultGlobalRole
  );
  const [teams, setTeams] = useState<ITeam[]>(defaultTeams);
  const [isGlobalUser, setIsGlobalUser] = useState(defaultTeams.length === 0);
  const [selectedEndpointKeys, setSelectedEndpointIds] = useState<string[]>(
    defaultSelectedEndpointKeys
  );
  const [formErrors, setFormErrors] = useState<IUserFormErrors>({});

  const combinedErrors = { ...formErrors, ...ancestorErrors };

  const validate = (): boolean => {
    const errors: IUserFormErrors = {};
    if (!validatePresence(name)) {
      errors.name = "Name is required";
    }
    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = (evt: FormEvent) => {
    evt.preventDefault();
    if (!validate()) return;

    onSubmit({
      name,
      global_role: isGlobalUser ? globalRole : null,
      teams: isGlobalUser ? [] : teams,
      api_only: true,
      selectedEndpointKeys:
        selectedEndpointKeys.length > 0 ? selectedEndpointKeys : undefined,
    });
  };

  const handleRoleChange = (newValue: SingleValue<CustomOptionType>) => {
    if (newValue) {
      setGlobalRole(newValue.value as UserRole);
    }
  };

  const handleTeamChange = (newTeams: ITeam[]) => {
    setTeams(newTeams);
  };

  const handleIsGlobalUserChange = (value: string) => {
    setIsGlobalUser(value === UserTeamType.GlobalUser);
  };

  const renderGlobalRoleForm = () => (
    <DropdownWrapper
      name="Role"
      label="Role"
      value={globalRole ?? "gitops"}
      options={roleOptions({ isPremiumTier, isApiOnly: true })}
      onChange={handleRoleChange}
      isSearchable={false}
    />
  );

  const renderTeamsForm = () => (
    <SelectedTeamsForm
      availableTeams={availableTeams}
      usersCurrentTeams={teams}
      onFormChange={handleTeamChange}
      isApiOnly
    />
  );

  const renderPermissions = () => (
    <>
      <div className="form-field team-field">
        <div className="form-field__label">Permissions</div>
        <Radio
          className={`${baseClass}__radio-input`}
          label="Global user"
          id="global-user"
          checked={isGlobalUser}
          value={UserTeamType.GlobalUser}
          name="user-team-type"
          onChange={handleIsGlobalUserChange}
        />
        <Radio
          className={`${baseClass}__radio-input`}
          label="Assign to fleet(s)"
          id="assign-teams"
          checked={!isGlobalUser}
          value={UserTeamType.AssignTeams}
          name="user-team-type"
          onChange={handleIsGlobalUserChange}
          disabled={!availableTeams.length}
        />
      </div>
      {isGlobalUser ? renderGlobalRoleForm() : renderTeamsForm()}
    </>
  );

  return (
    <div className={baseClass}>
      <form autoComplete="off">
        <InputField
          name="name"
          label="Name"
          value={name}
          onChange={(value: string) => setName(value)}
          error={combinedErrors.name}
          autofocus
        />
        {isPremiumTier ? renderPermissions() : renderGlobalRoleForm()}
        {isPremiumTier && (
          <ApiAccessSection
            selectedEndpointKeys={selectedEndpointKeys}
            onEndpointSelectionChange={setSelectedEndpointIds}
          />
        )}
      </form>
      <div className={`${baseClass}__button-wrap`}>
        <Button onClick={onCancel} variant="inverse">
          Cancel
        </Button>
        <Button
          type="submit"
          onClick={handleSubmit}
          isLoading={isSubmitting}
          disabled={Object.keys(combinedErrors).length > 0}
        >
          {isNewUser ? "Add" : "Save"}
        </Button>
      </div>
    </div>
  );
};

export default ApiUserForm;
