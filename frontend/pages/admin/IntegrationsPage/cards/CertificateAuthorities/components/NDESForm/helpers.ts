import { isValidURL } from "components/forms/validators";

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
          return isValidURL({ url: formData.scepURL });
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
          return isValidURL({ url: formData.adminURL });
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
