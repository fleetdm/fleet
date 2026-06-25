import { ICertificateAuthorityPartial } from "interfaces/certificates";

import { UNCHANGED_PASSWORD_API_RESPONSE } from "utilities/constants";
import { isValidURL } from "components/forms/validators";

import { ICustomSCEPFormData } from "./CustomSCEPForm";

// Windows encodes the SCEP challenge password as an ASN.1 PrintableString, so a challenge with any character outside that set (most
// commonly "_") fails. Keep in sync with printableStringChallengeRegexp in ee/server/service/certificate_authorities.go.
const PRINTABLE_STRING_REGEX = /^[A-Za-z0-9'()+,./:=?-]*$/;

// TODO: create a validator abstraction for this and the other form validation files

export interface ICustomSCEPFormValidation {
  isValid: boolean;
  name?: { isValid: boolean; message?: string };
  scepURL?: { isValid: boolean; message?: string };
  challenge?: { isValid: boolean; message?: string };
}

type IMessageFunc = (formData: ICustomSCEPFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<ICustomSCEPFormValidation, "isValid">;

interface IValidation {
  name: string;
  isValid: (formData: ICustomSCEPFormData) => boolean;
  message?: IValidationMessage;
}

type IFormValidations = Record<
  IFormValidationKey,
  { validations: IValidation[] }
>;

export const generateFormValidations = (
  certAuthorities: ICertificateAuthorityPartial[],
  isEditing: boolean
) => {
  const FORM_VALIDATIONS: IFormValidations = {
    name: {
      validations: [
        {
          name: "required",
          isValid: (formData: ICustomSCEPFormData) => {
            return formData.name.length > 0;
          },
        },
        {
          name: "invalidCharacters",
          isValid: (formData: ICustomSCEPFormData) => {
            return /^[a-zA-Z0-9_]+$/.test(formData.name);
          },
          message:
            "Invalid characters. Only letters, numbers and underscores allowed.",
        },
        {
          name: "unique",
          isValid: (formData: ICustomSCEPFormData) => {
            return (
              isEditing ||
              certAuthorities.find(
                (cert) =>
                  cert.type === "custom_scep_proxy" &&
                  cert.name === formData.name
              ) === undefined
            );
          },
          message: "Name is already used by another custom SCEP CA.",
        },
      ],
    },
    scepURL: {
      validations: [
        {
          name: "required",
          isValid: (formData: ICustomSCEPFormData) => {
            return formData.scepURL.length > 0;
          },
        },
        {
          name: "isValidURL",
          isValid: (formData: ICustomSCEPFormData) => {
            return isValidURL({ url: formData.scepURL });
          },
          message: "Must be a valid URL.",
        },
      ],
    },
    challenge: {
      validations: [
        {
          name: "required",
          isValid: (formData: ICustomSCEPFormData) => {
            return formData.challenge.length > 0;
          },
        },
        {
          name: "printableCharacters",
          isValid: (formData: ICustomSCEPFormData) => {
            // Skip an unchanged (masked) challenge, so editing a CA whose challenge predates this validation isn't blocked.
            return (
              formData.challenge === UNCHANGED_PASSWORD_API_RESPONSE ||
              PRINTABLE_STRING_REGEX.test(formData.challenge)
            );
          },
          message:
            "Invalid characters. Certificate enrollment only supports letters, numbers, and ' ( ) + , - . / : = ?",
        },
      ],
    },
  };

  return FORM_VALIDATIONS;
};

const getErrorMessage = (
  formData: ICustomSCEPFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

// eslint-disable-next-line import/prefer-default-export
export const validateFormData = (
  formData: ICustomSCEPFormData,
  validationConfig: IFormValidations
) => {
  const formValidation: ICustomSCEPFormValidation = {
    isValid: true,
  };

  Object.keys(validationConfig).forEach((key) => {
    const objKey = key as keyof typeof validationConfig;
    const failedValidation = validationConfig[objKey].validations.find(
      (validation) => !validation.isValid(formData)
    );

    if (!failedValidation) {
      formValidation[objKey] = {
        isValid: true,
      };
    } else {
      formValidation.isValid = false;
      formValidation[objKey] = {
        isValid: false,
        message: getErrorMessage(formData, failedValidation.message),
      };
    }
  });

  return formValidation;
};
