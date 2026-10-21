import { ICertificateAuthorityPartial } from "interfaces/certificates";
import { UNCHANGED_PASSWORD_API_RESPONSE } from "utilities/constants";

import { IGoogleCASFormData } from "./GoogleCASForm";

export interface IGoogleCASFormValidation {
  isValid: boolean;
  name?: { isValid: boolean; message?: string };
  projectId?: { isValid: boolean; message?: string };
  caPool?: { isValid: boolean; message?: string };
  location?: { isValid: boolean; message?: string };
  apiKeyJson?: { isValid: boolean; message?: string };
}

type IFormValidationKey = keyof IGoogleCASFormData;

interface IValidation {
  name: string;
  isValid: (formData: IGoogleCASFormData) => boolean;
  message?: string;
}

type IFormValidations = Record<
  IFormValidationKey,
  // The order of the validations determines the precedence when validating
  { validations: IValidation[] }
>;

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

const required = (field: IFormValidationKey): IValidation => ({
  name: "required",
  isValid: (formData) => formData[field].length > 0,
});

export const generateFormValidations = (
  certAuthorities: ICertificateAuthorityPartial[],
  isEditing: boolean
): IFormValidations => ({
  name: {
    validations: [
      required("name"),
      {
        name: "validCharacters",
        isValid: (formData) => /^[a-zA-Z0-9_]+$/.test(formData.name),
        message:
          "Invalid characters. Only letters, numbers and underscores allowed.",
      },
      {
        name: "unique",
        isValid: (formData) =>
          isEditing ||
          !certAuthorities.some(
            (cert) => cert.type === "google_cas" && cert.name === formData.name
          ),
        message: "Name is already used by another Google CAS CA.",
      },
    ],
  },
  projectId: {
    validations: [
      required("projectId"),
      {
        name: "validCharacters",
        isValid: (formData) => PROJECT_ID_REGEX.test(formData.projectId),
        message:
          "Must contain only lowercase letters, numbers, hyphens, periods and colons.",
      },
    ],
  },
  caPool: {
    validations: [
      required("caPool"),
      {
        name: "validCharacters",
        isValid: (formData) => CA_POOL_REGEX.test(formData.caPool),
        message: "Must contain only letters, numbers, hyphens and underscores.",
      },
    ],
  },
  location: {
    validations: [
      required("location"),
      {
        name: "validCharacters",
        isValid: (formData) => LOCATION_REGEX.test(formData.location),
        message: "Must contain only lowercase letters, numbers and hyphens.",
      },
    ],
  },
  apiKeyJson: {
    validations: [
      required("apiKeyJson"),
      {
        name: "validJSON",
        // The masked value the API returns for a saved key means "keep it".
        isValid: (formData) =>
          formData.apiKeyJson === UNCHANGED_PASSWORD_API_RESPONSE ||
          isJSONObject(formData.apiKeyJson),
        message: "Must be the service account key file's JSON.",
      },
    ],
  },
});

export const validateFormData = (
  formData: IGoogleCASFormData,
  validationConfig: IFormValidations
): IGoogleCASFormValidation => {
  const formValidation: IGoogleCASFormValidation = { isValid: true };

  (Object.keys(validationConfig) as IFormValidationKey[]).forEach((key) => {
    const failed = validationConfig[key].validations.find(
      (validation) => !validation.isValid(formData)
    );
    if (!failed) {
      formValidation[key] = { isValid: true };
      return;
    }
    formValidation.isValid = false;
    formValidation[key] = { isValid: false, message: failed.message };
  });

  return formValidation;
};
