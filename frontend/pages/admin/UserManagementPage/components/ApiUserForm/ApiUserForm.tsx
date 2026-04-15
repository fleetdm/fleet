import React, { FormEvent, useState, useContext } from "react";

import { AppContext } from "context/app";
import { IApiEndpointRef, endpointKey } from "interfaces/api_endpoint";
import { ITeam, INewTeamUser } from "interfaces/team";
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

export interface IApiUserFormData {
  name: string;
  global_role: UserRole | null;
  fleets: INewTeamUser[];
  api_endpoints?: IApiEndpointRef[] | null;
}

interface IApiUserFormProps {
  onCancel: () => void;
  onSubmit: (formData: IApiUserFormData) => void;
  availableTeams: ITeam[];
  isNewUser?: boolean;
  defaultName?: string;
  defaultGlobalRole?: UserRole | null;
  defaultFleets?: ITeam[];
  defaultApiEndpoints?: IApiEndpointRef[];
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
  defaultGlobalRole,
  defaultFleets = [],
  defaultApiEndpoints,
  formErrors: ancestorErrors = {},
  isSubmitting = false,
}: IApiUserFormProps) => {
  const { isPremiumTier } = useContext(AppContext);

  const defaultRole =
    defaultGlobalRole ?? (isPremiumTier ? "gitops" : "observer");

  const [name, setName] = useState(defaultName);
  const [globalRole, setGlobalRole] = useState<UserRole | null>(defaultRole);
  const [fleets, setFleets] = useState<ITeam[]>(defaultFleets);
  const [isGlobalUser, setIsGlobalUser] = useState(defaultFleets.length === 0);
  const [selectedEndpointKeys, setSelectedEndpointKeys] = useState<string[]>(
    () => (defaultApiEndpoints ? defaultApiEndpoints.map(endpointKey) : [])
  );
  const [isSpecificEndpoints, setIsSpecificEndpoints] = useState(
    () => !!defaultApiEndpoints && defaultApiEndpoints.length > 0
  );
  const [formErrors, setFormErrors] = useState<IUserFormErrors>({});

  const combinedErrors = { ...formErrors, ...ancestorErrors };

  const clearEndpointError = () => {
    if (formErrors.api_endpoints) {
      setFormErrors((prev) => {
        const { api_endpoints: _, ...rest } = prev;
        return rest;
      });
    }
  };

  const handleEndpointSelectionChange = (keys: string[]) => {
    setSelectedEndpointKeys(keys);
    if (keys.length > 0) {
      clearEndpointError();
    }
  };

  const handleAccessTypeChange = (specific: boolean) => {
    setIsSpecificEndpoints(specific);
    if (!specific) {
      clearEndpointError();
    }
  };

  const getErrors = (): IUserFormErrors => {
    const errors: IUserFormErrors = {};
    if (!validatePresence(name)) {
      errors.name = "Name is required";
    }
    if (isSpecificEndpoints && selectedEndpointKeys.length === 0) {
      errors.api_endpoints = "Please select at least one API endpoint";
    }
    return errors;
  };

  const onInputChange = (value: string) => {
    setName(value);
    if (formErrors.name && validatePresence(value)) {
      setFormErrors((prev) => {
        const { name: _n, ...rest } = prev;
        return rest;
      });
    }
  };

  const onInputBlur = () => {
    setFormErrors(getErrors());
  };

  const handleSubmit = (evt: FormEvent) => {
    evt.preventDefault();
    const errs = getErrors();
    if (Object.keys(errs).length > 0) {
      setFormErrors(errs);
      return;
    }

    // Convert endpoint keys ("METHOD /path") back to { method, path } objects.
    // When "All" is selected, send null to clear any specific endpoints (full access).
    const apiEndpoints = isSpecificEndpoints
      ? selectedEndpointKeys.map((key) => {
          const spaceIndex = key.indexOf(" ");
          return {
            method: key.substring(0, spaceIndex),
            path: key.substring(spaceIndex + 1),
          };
        })
      : null;

    onSubmit({
      name,
      global_role: isGlobalUser ? globalRole : null,
      fleets: isGlobalUser
        ? []
        : fleets.map((f) => ({
            id: f.id,
            role: f.role || "observer",
          })),
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
      value={globalRole ?? defaultRole}
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
          label="Global user"
          id="global-user"
          checked={isGlobalUser}
          value={UserTeamType.GlobalUser}
          name="user-team-type"
          onChange={handleIsGlobalUserChange}
        />
        <Radio
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
    <div>
      <form autoComplete="off" onSubmit={handleSubmit}>
        <InputField
          name="name"
          label="Name"
          value={name}
          onChange={onInputChange}
          onBlur={onInputBlur}
          error={combinedErrors.name}
          autofocus
        />
        {isPremiumTier ? renderPermissions() : renderGlobalRoleForm()}
        {isPremiumTier && (
          <ApiAccessSection
            selectedEndpointKeys={selectedEndpointKeys}
            onEndpointSelectionChange={handleEndpointSelectionChange}
            onAccessTypeChange={handleAccessTypeChange}
            error={combinedErrors.api_endpoints}
          />
        )}
        <div className="user-management-form__footer">
          <Button onClick={onCancel} variant="inverse">
            Cancel
          </Button>
          <Button
            type="submit"
            isLoading={isSubmitting}
            disabled={Object.keys(combinedErrors).length > 0}
          >
            {isNewUser ? "Add" : "Save"}
          </Button>
        </div>
      </form>
    </div>
  );
};

export default ApiUserForm;
