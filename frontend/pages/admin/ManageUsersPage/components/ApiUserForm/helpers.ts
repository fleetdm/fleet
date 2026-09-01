import { IApiEndpointRef } from "interfaces/api_endpoint";
import { ITeam } from "interfaces/team";
import { UserRole } from "interfaces/user";
import { IFormErrors } from "hooks/useFormValidation";
import validatePresence from "components/forms/validators/validate_presence";

export type ApiUserFormState = {
  name: string;
  global_role: UserRole;
  isGlobalUser: boolean;
  fleets: ITeam[];
  isSpecificEndpoints: boolean;
  api_endpoints: IApiEndpointRef[];
};

export const validateApiUserForm = (
  data: ApiUserFormState,
  { isPremiumTier }: { isPremiumTier?: boolean }
): IFormErrors => {
  const errors: IFormErrors = {};

  if (!validatePresence(data.name)) {
    errors.name = "Enter a name";
  }
  // The permissions and API-access selectors only render on premium.
  if (isPremiumTier) {
    if (!data.isGlobalUser && data.fleets.length === 0) {
      errors.fleets = "Select at least one fleet";
    }
    if (data.isSpecificEndpoints && data.api_endpoints.length === 0) {
      errors.api_endpoints = "Select at least one API endpoint";
    }
  }

  return errors;
};
