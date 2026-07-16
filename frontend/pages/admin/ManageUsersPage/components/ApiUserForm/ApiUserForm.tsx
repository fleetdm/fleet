import React, { FormEvent, useState } from "react";

import { IApiEndpointRef } from "interfaces/api_endpoint";
import { ITeam } from "interfaces/team";
import { IUserFormErrors, UserRole } from "interfaces/user";

import { SingleValue } from "react-select-5";
import Button from "components/buttons/Button";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import validatePresence from "components/forms/validators/validate_presence";
import InputField from "components/forms/fields/InputField";
import Radio from "components/forms/fields/Radio";

import SelectedTeamsForm from "../SelectedTeamsForm/SelectedTeamsForm";
import ApiAccessSection from "../ApiAccessSection";
import { roleOptions } from "../../helpers/userManagementHelpers";

export interface IApiUserFormData {
  name: string;
  global_role: UserRole | null;
  fleets: ITeam[];
  api_endpoints?: IApiEndpointRef[] | null;
}

interface IApiUserFormProps {
  onCancel: () => void;
  onSubmit: (formData: IApiUserFormData) => void;
  availableTeams: ITeam[];
  defaultData?: IApiUserFormData;
  isSubmitting?: boolean;
  isPremiumTier?: boolean;
}

enum UserTeamType {
  GlobalUser = "GLOBAL_USER",
  AssignTeams = "ASSIGN_TEAMS",
}

const ApiUserForm = ({
  isPremiumTier,
  onCancel,
  onSubmit,
  availableTeams,
  defaultData,
  isSubmitting = false,
}: IApiUserFormProps) => {
  const isNewUser = defaultData === undefined;

  const [name, setName] = useState(defaultData?.name ?? "");
  const [globalRole, setGlobalRole] = useState<UserRole>(
    () =>
      (defaultData?.global_role ??
        (isPremiumTier ? "gitops" : "observer")) as UserRole
  );
  const [fleets, setFleets] = useState<ITeam[]>(defaultData?.fleets ?? []);
  const [isGlobalUser, setIsGlobalUser] = useState(
    !defaultData?.fleets?.length
  );

  const [selectedEndpoints, setSelectedEndpoints] = useState<IApiEndpointRef[]>(
    () => defaultData?.api_endpoints ?? []
  );

  // null (all endpoints) and undefined (field not set / free tier) are both treated as "all endpoints"
  const [isSpecificEndpoints, setIsSpecificEndpoints] = useState(
    () => !!defaultData?.api_endpoints && defaultData.api_endpoints.length > 0
  );
  const [formErrors, setFormErrors] = useState<IUserFormErrors>({});

  const clearEndpointError = () => {
    if (formErrors.api_endpoints) {
      setFormErrors((prev) => {
        const { api_endpoints: _, ...rest } = prev;
        return rest;
      });
    }
  };

  const handleEndpointSelectionChange = (endpoints: IApiEndpointRef[]) => {
    setSelectedEndpoints(endpoints);
    if (endpoints.length > 0) {
      clearEndpointError();
    }
  };

  const handleAccessTypeChange = (specific: boolean) => {
    setIsSpecificEndpoints(specific);
    if (!specific) {
      setSelectedEndpoints([]);
      clearEndpointError();
    }
  };

  const getErrors = (): IUserFormErrors => {
    const errors: IUserFormErrors = {};
    if (!validatePresence(name)) {
      errors.name = "Name is required";
    }
    if (!isGlobalUser && fleets.length === 0) {
      errors.teams = "Please select at least one fleet";
    }
    if (isSpecificEndpoints && selectedEndpoints.length === 0) {
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
    if (!validatePresence(name)) {
      setFormErrors((prev) => ({ ...prev, name: "Name is required" }));
    }
  };

  const handleSubmit = (evt: FormEvent) => {
    evt.preventDefault();
    const errs = getErrors();
    if (Object.keys(errs).length > 0) {
      setFormErrors(errs);
      return;
    }

    // Omit api_endpoints on free tier to avoid clearing a value set by a premium instance.
    // When "All" is selected, send null to signal full access.
    let apiEndpoints: IApiEndpointRef[] | null | undefined;
    if (isPremiumTier) {
      apiEndpoints = isSpecificEndpoints ? selectedEndpoints : null;
    }

    onSubmit({
      name,
      global_role: isGlobalUser ? globalRole : null,
      fleets: isGlobalUser
        ? []
        : fleets.map((f) => ({ ...f, role: f.role || "observer" })),
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
    if (newFleets.length > 0 && formErrors.teams) {
      setFormErrors((prev) => {
        const { teams: _, ...rest } = prev;
        return rest;
      });
    }
  };

  const handleIsGlobalUserChange = (value: string) => {
    const isGlobal = value === UserTeamType.GlobalUser;
    setIsGlobalUser(isGlobal);
    if (isGlobal && formErrors.teams) {
      setFormErrors((prev) => {
        const { teams: _, ...rest } = prev;
        return rest;
      });
    }
  };

  const renderGlobalRoleForm = () => (
    <DropdownWrapper
      name="Role"
      label="Role"
      value={globalRole}
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
      {isGlobalUser ? (
        renderGlobalRoleForm()
      ) : (
        <>
          {renderTeamsForm()}
          {formErrors.teams && (
            <div className="form-field__label form-field__label--error">
              {formErrors.teams}
            </div>
          )}
        </>
      )}
    </>
  );

  return (
    <>
      <div>
        <form autoComplete="off" onSubmit={handleSubmit}>
          <InputField
            name="name"
            label="Name"
            value={name}
            onChange={onInputChange}
            onBlur={onInputBlur}
            error={formErrors.name}
            autofocus
          />
          {isPremiumTier ? renderPermissions() : renderGlobalRoleForm()}
          {isPremiumTier && (
            <ApiAccessSection
              isSpecificEndpoints={isSpecificEndpoints}
              onAccessTypeChange={handleAccessTypeChange}
              selectedEndpoints={selectedEndpoints}
              onEndpointSelectionChange={handleEndpointSelectionChange}
              error={formErrors.api_endpoints}
            />
          )}
          <div className="user-management-form__footer">
            <Button onClick={onCancel} variant="inverse">
              Cancel
            </Button>
            <Button
              type="submit"
              isLoading={isSubmitting}
              disabled={isSubmitting}
            >
              {isNewUser ? "Add" : "Save"}
            </Button>
          </div>
        </form>
      </div>
    </>
  );
};

export default ApiUserForm;
