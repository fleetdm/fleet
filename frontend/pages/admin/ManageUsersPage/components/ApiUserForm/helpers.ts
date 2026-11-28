import { isPresent } from "components/forms/validators";
import { IFormErrors } from "hooks/useFormValidation";
import { endpointFingerprint, IApiEndpointRef } from "interfaces/api_endpoint";
import { IExternalIdentity } from "interfaces/external_identity";
import { ITeam } from "interfaces/team";
import { UserRole } from "interfaces/user";

import { externalIdentityErrorKey } from "../ExternalIdentitiesSection/ExternalIdentitiesSection";

export type ApiUserFormState = {
  name: string;
  global_role: UserRole;
  isGlobalUser: boolean;
  fleets: ITeam[];
  isSpecificEndpoints: boolean;
  api_endpoints: IApiEndpointRef[];
  external_identities: IExternalIdentity[];
};

export interface IApiUserFormValidationContext {
  isPremiumTier?: boolean;
  apiEndpointCatalog?: IApiEndpointRef[];
}

// Subjects are matched byte-for-byte, so whitespace or a repeat is almost
// certainly a mistake; the server rejects both.
const validateExternalIdentities = (
  identities: IExternalIdentity[],
  errors: IFormErrors
) => {
  const seen = new Set<string>();
  identities.forEach(({ issuer, subject }, index) => {
    const key = externalIdentityErrorKey(index);
    const identityKey = JSON.stringify([issuer, subject]);
    if (!subject) {
      errors[key] = "Enter a subject";
    } else if (subject.trim() !== subject) {
      errors[key] = "Remove leading and trailing spaces";
    } else if (seen.has(identityKey)) {
      errors[key] = "Remove the duplicate identity";
    }
    seen.add(identityKey);
  });
};

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
  validateExternalIdentities(data.external_identities, errors);

  return errors;
};
