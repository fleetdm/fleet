import { IFormErrors } from "hooks/useFormValidation";
import { ICertificateAuthorityPartial } from "interfaces/certificates";
import { UNCHANGED_PASSWORD_API_RESPONSE } from "utilities/constants";

import { IGoogleCASFormData } from "./GoogleCASForm";

const NAME_REGEX = /^[a-zA-Z0-9_]+$/;
// The server limits these to the characters Google allows in the IDs, because
// they become path segments of Google API URLs.
const PROJECT_ID_REGEX = /^[a-z0-9][a-z0-9.:-]*$/;
const LOCATION_REGEX = /^[a-z0-9-]+$/;
const CA_POOL_REGEX = /^[a-zA-Z0-9_-]+$/;

const isJSONObject = (value: string) => {
  try {
    const parsed = JSON.parse(value);
    return (
      typeof parsed === "object" && parsed !== null && !Array.isArray(parsed)
    );
  } catch {
    return false;
  }
};

interface IValidateOptions {
  certAuthorities: ICertificateAuthorityPartial[];
  isEditing: boolean;
}

const validateGoogleCASForm = (
  formData: IGoogleCASFormData,
  { certAuthorities, isEditing }: IValidateOptions
): IFormErrors => {
  const errors: IFormErrors = {};
  const name = formData.name.trim();
  const projectId = formData.projectId.trim();
  const caPool = formData.caPool.trim();
  const location = formData.location.trim();
  const apiKeyJson = formData.apiKeyJson.trim();

  if (!name) {
    errors.name = "Enter a name";
  } else if (!NAME_REGEX.test(name)) {
    errors.name = "Enter a name with only letters, numbers, and underscores";
  } else if (
    !isEditing &&
    certAuthorities.some((ca) => ca.type === "google_cas" && ca.name === name)
  ) {
    errors.name = "Enter a name that isn't used by another Google CAS CA";
  }

  if (!projectId) {
    errors.projectId = "Enter a project ID";
  } else if (!PROJECT_ID_REGEX.test(projectId)) {
    errors.projectId =
      "Enter a project ID with only lowercase letters, numbers, hyphens, periods, and colons";
  }

  if (!caPool) {
    errors.caPool = "Enter a CA pool";
  } else if (!CA_POOL_REGEX.test(caPool)) {
    errors.caPool =
      "Enter a CA pool with only letters, numbers, hyphens, and underscores";
  }

  if (!location) {
    errors.location = "Enter a location";
  } else if (!LOCATION_REGEX.test(location)) {
    errors.location =
      "Enter a location with only lowercase letters, numbers, and hyphens";
  }

  if (!apiKeyJson) {
    errors.apiKeyJson = "Enter a service account key";
  } else if (
    // When editing, the masked value the API returns means "keep the saved key".
    !(isEditing && apiKeyJson === UNCHANGED_PASSWORD_API_RESPONSE) &&
    !isJSONObject(apiKeyJson)
  ) {
    errors.apiKeyJson = "Enter the service account key file's JSON";
  }

  return errors;
};

export default validateGoogleCASForm;
