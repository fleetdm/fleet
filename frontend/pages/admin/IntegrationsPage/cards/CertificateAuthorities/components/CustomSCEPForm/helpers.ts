import { getErrorReason } from "interfaces/errors";
import { ICustomSCEPFormData } from "./CustomSCEPForm";

// TODO: create a validator abstraction for this and the other form validation files

export interface ICustomSCEPFormValidation {
  isValid: boolean;
  name?: { isValid: boolean; message?: string };
  scepURL?: { isValid: boolean };
  challenge?: { isValid: boolean };
}

type IMessageFunc = (formData: ICustomSCEPFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<ICustomSCEPFormValidation, "isValid">;

interface IValidation {
  name: string;
  isValid: (formData: ICustomSCEPFormData) => boolean;
  message?: IValidationMessage;
}

const FORM_VALIDATIONS: Record<
  IFormValidationKey,
  { validations: IValidation[] }
> = {
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
          "Inalid characters. Only letters, numbers and underscores allowed.",
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
    ],
  },
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
export const validateFormData = (formData: ICustomSCEPFormData) => {
  const formValidation: ICustomSCEPFormValidation = {
    isValid: true,
  };

  Object.keys(FORM_VALIDATIONS).forEach((key) => {
    const objKey = key as keyof typeof FORM_VALIDATIONS;
    const failedValidation = FORM_VALIDATIONS[objKey].validations.find(
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

const BAD_SCEP_URL_ERROR = "Invalid SCEP URL. Please correct and try again.";
const BAD_CREDENTIALS_ERROR =
  "Couldn't add. Admin URL or credentials are invalid.";
const CACHE_ERROR =
  "The NDES password cache is full. Please increase the number of cached passwords in NDES and try again. By default, NDES caches 5 passwords and they expire 60 minutes after they are created.";
const INSUFFICIENT_PERMISSIONS_ERROR =
  "Couldn't add. This account doesn't have sufficient permissions. Please use the account with enroll permission.";
const SCEP_URL_TIMEOUT_ERROR =
  "Couldn't add. Request to NDES (SCEP URL) timed out. Please try again.";
const DEFAULT_ERROR =
  "Something went wrong updating your SCEP server. Please try again.";

// export const getErrorMessage = (
//   err: unknown,
//   formData: ICustomSCEPFormData
// ) => {
//   const reason = getErrorReason(err);

//   if (reason.includes("invalid admin URL or credentials")) {
//     return BAD_CREDENTIALS_ERROR;
//   } else if (reason.includes("the password cache is full")) {
//     return CACHE_ERROR;
//   } else if (reason.includes("does not have sufficient permissions")) {
//     INSUFFICIENT_PERMISSIONS_ERROR;
//   } else if (
//     reason.includes(formData.scepURL) &&
//     reason.includes("context deadline exceeded")
//   ) {
//     return SCEP_URL_TIMEOUT_ERROR;
//   } else if (reason.includes("invalid SCEP URL")) {
//     return BAD_SCEP_URL_ERROR;
//   }

//   return DEFAULT_ERROR;
// };
