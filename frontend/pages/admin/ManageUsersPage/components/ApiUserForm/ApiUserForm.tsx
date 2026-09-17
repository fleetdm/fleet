import React from "react";

import { IApiEndpointRef } from "interfaces/api_endpoint";
import { ITeam } from "interfaces/team";
import { UserRole } from "interfaces/user";
import { MAX_ENTITY_CHAR_LENGTH } from "utilities/constants";
import useFormValidation from "hooks/useFormValidation";

import { SingleValue } from "react-select-5";
import Button from "components/buttons/Button";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import InputField from "components/forms/fields/InputField";
import Radio from "components/forms/fields/Radio";

import SelectedTeamsForm from "../SelectedTeamsForm/SelectedTeamsForm";
import ApiAccessSection from "../ApiAccessSection";
import { roleOptions } from "../../helpers/userManagementHelpers";
import { ApiUserFormState, validateApiUserForm } from "./helpers";

export interface IApiUserFormData {
  name: string;
  global_role: UserRole | null;
  fleets: ITeam[];
  api_endpoints?: IApiEndpointRef[] | null;
}

interface IApiUserFormProps {
  onCancel: () => void;
  onSubmit: (formData: IApiUserFormData) => void | Promise<unknown>;
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
  isSubmitting: isSubmittingProp = false,
}: IApiUserFormProps) => {
  const isNewUser = defaultData === undefined;

  const validate = (data: ApiUserFormState) =>
    validateApiUserForm(data, { isPremiumTier });

  const {
    formData,
    setField,
    commitFields,
    getError,
    clearFieldError,
    validateField,
    handleSubmit,
    isSubmitting,
  } = useFormValidation<ApiUserFormState>({
    initialFormData: {
      name: defaultData?.name ?? "",
      global_role: (defaultData?.global_role ??
        (isPremiumTier ? "gitops" : "observer")) as UserRole,
      isGlobalUser: !defaultData?.fleets?.length,
      fleets: defaultData?.fleets ?? [],
      // null (all endpoints) and undefined (field not set / free tier) both mean
      // "all endpoints"
      isSpecificEndpoints: !!defaultData?.api_endpoints?.length,
      api_endpoints: defaultData?.api_endpoints ?? [],
    },
    validate,
    isSubmitting: isSubmittingProp,
  });

  const onValidSubmit = (data: ApiUserFormState) => {
    // Omit api_endpoints on free tier to avoid clearing a value set by a premium
    // instance. When "All" is selected, send null to signal full access.
    let apiEndpoints: IApiEndpointRef[] | null | undefined;
    if (isPremiumTier) {
      apiEndpoints = data.isSpecificEndpoints ? data.api_endpoints : null;
    }

    return onSubmit({
      name: data.name,
      global_role: data.isGlobalUser ? data.global_role : null,
      fleets: data.isGlobalUser
        ? []
        : data.fleets.map((f) => ({ ...f, role: f.role || "observer" })),
      api_endpoints: apiEndpoints,
    });
  };

  const onRoleChange = (newValue: SingleValue<CustomOptionType>) => {
    if (newValue) {
      commitFields({ global_role: newValue.value as UserRole });
    }
  };

  const onIsGlobalUserChange = (value: string) => {
    commitFields({ isGlobalUser: value === UserTeamType.GlobalUser });
  };

  // Narrowing to "all endpoints" discards the selection.
  const onAccessTypeChange = (isSpecific: boolean) => {
    commitFields(
      isSpecific
        ? { isSpecificEndpoints: true }
        : { isSpecificEndpoints: false, api_endpoints: [] }
    );
  };

  const renderGlobalRoleForm = () => (
    <DropdownWrapper
      name="Role"
      label="Role"
      value={formData.global_role}
      options={roleOptions({ isPremiumTier, isApiOnly: true })}
      onChange={onRoleChange}
      isSearchable={false}
      isDisabled={isSubmitting}
    />
  );

  const renderPermissions = () => (
    <>
      <div className="form-field team-field">
        <div className="form-field__label">Permissions</div>
        <Radio
          label="Global user"
          id="global-user"
          checked={formData.isGlobalUser}
          value={UserTeamType.GlobalUser}
          name="user-team-type"
          onChange={onIsGlobalUserChange}
          disabled={isSubmitting}
        />
        <Radio
          label="Assign to fleet(s)"
          id="assign-teams"
          checked={!formData.isGlobalUser}
          value={UserTeamType.AssignTeams}
          name="user-team-type"
          onChange={onIsGlobalUserChange}
          disabled={isSubmitting || !availableTeams.length}
        />
      </div>
      {formData.isGlobalUser ? (
        renderGlobalRoleForm()
      ) : (
        <>
          <SelectedTeamsForm
            availableTeams={availableTeams}
            usersCurrentTeams={formData.fleets}
            onFormChange={(fleets: ITeam[]) => commitFields({ fleets })}
            isApiOnly
            disabled={isSubmitting}
          />
          {getError("fleets") && (
            <div className="form-field__label form-field__label--error">
              {getError("fleets")}
            </div>
          )}
        </>
      )}
    </>
  );

  return (
    <div>
      <form autoComplete="off" onSubmit={handleSubmit(onValidSubmit)}>
        <InputField
          name="name"
          label="Name"
          value={formData.name}
          onChange={(value: string) => setField("name", value)}
          onFocus={() => clearFieldError("name")}
          onBlur={() => validateField("name")}
          error={getError("name")}
          disabled={isSubmitting}
          autofocus
          inputOptions={{ maxLength: MAX_ENTITY_CHAR_LENGTH }}
        />
        {isPremiumTier ? renderPermissions() : renderGlobalRoleForm()}
        {isPremiumTier && (
          <ApiAccessSection
            isSpecificEndpoints={formData.isSpecificEndpoints}
            onAccessTypeChange={onAccessTypeChange}
            selectedEndpoints={formData.api_endpoints}
            onEndpointSelectionChange={(api_endpoints: IApiEndpointRef[]) =>
              commitFields({ api_endpoints })
            }
            error={getError("api_endpoints")}
            disabled={isSubmitting}
          />
        )}
        <div className="user-management-form__footer">
          <Button onClick={onCancel} variant="secondary">
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
  );
};

export default ApiUserForm;
