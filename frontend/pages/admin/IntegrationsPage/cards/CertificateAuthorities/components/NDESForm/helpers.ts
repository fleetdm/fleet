import { getErrorReason } from "interfaces/errors";

import valid_url from "components/forms/validators/valid_url";

import { INDESFormData } from "./NDESForm";

// TODO: create a validator abstraction for this and the other form validation files

export interface INDESFormValidation {
  isValid: boolean;
  scepURL?: { isValid: boolean; message?: string };
  adminURL?: { isValid: boolean; message?: string };
  username?: { isValid: boolean };
  password?: { isValid: boolean };
}

type IMessageFunc = (formData: INDESFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<INDESFormValidation, "isValid">;

interface IValidation {
  name: string;
  isValid: (formData: INDESFormData) => boolean;
  message?: IValidationMessage;
}

const FORM_VALIDATIONS: Record<
  IFormValidationKey,
  { validations: IValidation[] }
> = {
  scepURL: {
    validations: [
      {
        name: "required",
        isValid: (formData: INDESFormData) => {
          return formData.scepURL.length > 0;
        },
      },
      {
        name: "validUrl",
        isValid: (formData: INDESFormData) => {
          return valid_url({ url: formData.scepURL });
        },
        message: "Must be a valid URL.",
      },
    ],
  },
  adminURL: {
    validations: [
      {
        name: "required",
        isValid: (formData: INDESFormData) => {
          return formData.adminURL.length > 0;
        },
      },
      {
        name: "validUrl",
        isValid: (formData: INDESFormData) => {
          return valid_url({ url: formData.adminURL });
        },
        message: "Must be a valid URL",
      },
    ],
  },
  username: {
    validations: [
      {
        name: "required",
        isValid: (formData: INDESFormData) => {
          return formData.username.length > 0;
        },
      },
    ],
  },
  password: {
    validations: [
      {
        name: "required",
        isValid: (formData: INDESFormData) => {
          return formData.password.length > 0;
        },
      },
    ],
  },
};

const getValifationErrorMessage = (
  formData: INDESFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

// eslint-disable-next-line import/prefer-default-export
export const validateFormData = (formData: INDESFormData) => {
  const formValidation: INDESFormValidation = {
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
        message: getValifationErrorMessage(formData, failedValidation.message),
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
const ADMIN_URL_TIMEOUT_ERROR =
  "Couldn't add. Request to NDES (admin URL) timed out. Please try again.";
const DEFAULT_ERROR =
  "Something went wrong updating your SCEP server. Please try again.";

export const getErrorMessage = (err: unknown, formData: INDESFormData) => {
  const reason = getErrorReason(err);

  if (reason.includes("invalid admin URL or credentials")) {
    return BAD_CREDENTIALS_ERROR;
  } else if (reason.includes("the password cache is full")) {
    return CACHE_ERROR;
  } else if (reason.includes("does not have sufficient permissions")) {
    INSUFFICIENT_PERMISSIONS_ERROR;
  } else if (
    reason.includes(formData.scepURL) &&
    reason.includes("context deadline exceeded")
  ) {
    return SCEP_URL_TIMEOUT_ERROR;
  } else if (
    reason.includes(formData.adminURL) &&
    reason.includes("context deadline exceeded")
  ) {
    return ADMIN_URL_TIMEOUT_ERROR;
  } else if (reason.includes("invalid SCEP URL")) {
    return BAD_SCEP_URL_ERROR;
  }

  return DEFAULT_ERROR;
};
