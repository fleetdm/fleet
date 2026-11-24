import { isPresent } from "components/forms/validators";
import { IFormErrors } from "hooks/useFormValidation";
import { endpointFingerprint, IApiEndpointRef } from "interfaces/api_endpoint";
import { ITeam } from "interfaces/team";
import { UserRole } from "interfaces/user";

export type ApiUserFormState = {
  name: string;
  global_role: UserRole;
  isGlobalUser: boolean;
  fleets: ITeam[];
  isSpecificEndpoints: boolean;
  api_endpoints: IApiEndpointRef[];
};

export interface IApiUserFormValidationContext {
  isPremiumTier?: boolean;
  apiEndpointCatalog?: IApiEndpointRef[];
}

/**
 * Drops selected endpoints the catalog no longer lists (e.g. a removed route).
 * The selector can't show them, so the user can't remove them, and the server
 * rejects them. A missing or empty catalog means it hasn't loaded (or failed
 * to), so nothing is dropped.
 */
export const dropUnlistedEndpoints = (
  selected: IApiEndpointRef[],
  catalog?: IApiEndpointRef[]
): IApiEndpointRef[] => {
  if (!catalog?.length) {
    return selected;
  }
  const listed = new Set(catalog.map(endpointFingerprint));
  return selected.filter((ep) => listed.has(endpointFingerprint(ep)));
};

export const validateApiUserForm = (
  data: ApiUserFormState,
  { isPremiumTier, apiEndpointCatalog }: IApiUserFormValidationContext
): IFormErrors => {
  const errors: IFormErrors = {};

  if (!isPresent(data.name)) {
    errors.name = "Enter a name";
  }
  // The permissions and API-access selectors only render on premium.
  if (isPremiumTier) {
    if (!data.isGlobalUser && data.fleets.length === 0) {
      errors.fleets = "Select at least one fleet";
    }
    if (
      data.isSpecificEndpoints &&
      dropUnlistedEndpoints(data.api_endpoints, apiEndpointCatalog).length === 0
    ) {
      errors.api_endpoints = "Select at least one API endpoint";
    }
  }

  return errors;
};
