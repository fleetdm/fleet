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

export interface IApiEndpointRef {
  method: string;
  path: string;
}

export interface IApiUserFormData {
  name: string;
  global_role: UserRole | null;
  fleets: ITeam[];
  api_endpoints?: IApiEndpointRef[];
}

interface IApiUserFormProps {
  onCancel: () => void;
  onSubmit: (formData: IApiUserFormData) => void;
  availableTeams: ITeam[];
  isNewUser?: boolean;
  defaultName?: string;
  defaultGlobalRole?: UserRole | null;
  defaultFleets?: ITeam[];
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
  defaultFleets = [],
  formErrors: ancestorErrors = {},
  isSubmitting = false,
}: IApiUserFormProps) => {
  const { isPremiumTier } = useContext(AppContext);

  const [name, setName] = useState(defaultName);
  const [globalRole, setGlobalRole] = useState<UserRole | null>(
    defaultGlobalRole
  );
  const [fleets, setFleets] = useState<ITeam[]>(defaultFleets);
  const [isGlobalUser, setIsGlobalUser] = useState(defaultFleets.length === 0);
  const [selectedEndpointKeys, setSelectedEndpointKeys] = useState<string[]>(
    []
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

    // Convert endpoint keys ("METHOD /path") back to { method, path } objects
    const apiEndpoints =
      selectedEndpointKeys.length > 0
        ? selectedEndpointKeys.map((key) => {
            const spaceIndex = key.indexOf(" ");
            return {
              method: key.substring(0, spaceIndex),
              path: key.substring(spaceIndex + 1),
            };
          })
        : undefined;

    onSubmit({
      name,
      global_role: isGlobalUser ? globalRole : null,
      fleets: isGlobalUser ? [] : fleets,
      api_endpoints: apiEndpoints,
    });
  };

  const handleRoleChange = (newValue: SingleValue<CustomOptionType>) => {
    if (newValue) {
      setGlobalRole(newValue.value as UserRole);
    }
  };

  const handleFleetChange = (newFleets: ITeam[]) => {
    setFleets(newFleets);
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
      usersCurrentTeams={fleets}
      onFormChange={handleFleetChange}
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
            onEndpointSelectionChange={setSelectedEndpointKeys}
          />
        )}
      </form>
      <div className="user-management-form__footer">
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
