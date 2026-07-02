import { isValidUuid } from "components/forms/validators";

import { IAddClientIdFormData } from "./AddEntraClientIDModal";

export interface IAddClientIdFormValidation {
  isValid: boolean;
  clientId?: { isValid: boolean; message?: string };
}

type IMessageFunc = (formData: IAddClientIdFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<IAddClientIdFormValidation, "isValid">;

interface IValidation {
  name: string;
  isValid: (formData: IAddClientIdFormData) => boolean;
  message?: IValidationMessage;
}

type IFormValidations = Record<
  IFormValidationKey,
  { validations: IValidation[] }
>;

const FORM_VALIDATIONS: IFormValidations = {
  clientId: {
    validations: [
      {
        name: "required",
        isValid: (formData: IAddClientIdFormData) => {
          return (
            formData.clientId !== undefined && formData.clientId.length > 0
          );
        },
        message: (formData: IAddClientIdFormData) =>
          formData.clientId === undefined ? "" : `Client ID is required`,
      },
      {
        name: "validUUID",
        isValid: (formData: IAddClientIdFormData) => {
          if (
            formData.clientId === undefined ||
            formData.clientId.length === 0
          ) {
            return true; // Skip this validation if the value is empty
          }
          return isValidUuid(formData.clientId);
        },
        message: "Invalid UUID. Please provide a valid UUID format.",
      },
    ],
  },
};

const getErrorMessage = (
  formData: IAddClientIdFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

export const validateFormData = (formData: IAddClientIdFormData) => {
  const formValidation: IAddClientIdFormValidation = {
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
