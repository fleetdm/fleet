import { ILabelFormData } from "./LabelForm";

export interface ILabelFormValidation {
  isValid: boolean;
  name?: { isValid: boolean; message?: string };
  description?: { isValid: boolean; message?: string };
}

type IMessageFunc = (formData: ILabelFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof ILabelFormData;

interface IValidation {
  name: string;
  isValid: (
    formData: ILabelFormData,
    validations?: ILabelFormValidation
  ) => boolean;
  message?: IValidationMessage;
}

type IFormValidations = Record<
  IFormValidationKey,
  { validations: IValidation[] }
>;

const FORM_VALIDATIONS: IFormValidations = {
  name: {
    validations: [
      {
        name: "required",
        isValid: (formData) => formData.name.trim().length > 0,
        message: "Label name must be present",
      },
    ],
  },
  description: {
    validations: [],
  },
};

const getErrorMessage = (
  formData: ILabelFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

export const validateLabelFormData = (
  formData: ILabelFormData
): ILabelFormValidation => {
  const formValidation: ILabelFormValidation = { isValid: true };

  (Object.keys(FORM_VALIDATIONS) as IFormValidationKey[]).forEach((objKey) => {
    const failedValidation = FORM_VALIDATIONS[objKey].validations.find(
      (validation) => !validation.isValid(formData, formValidation)
    );

    if (!failedValidation) {
      switch (objKey) {
        case "name":
          formValidation.name = { isValid: true };
          break;
        case "description":
          formValidation.description = { isValid: true };
          break;
        default: {
          const _exhaustiveCheck: never = objKey;
          break;
        }
      }
    } else {
      formValidation.isValid = false;
      const message = getErrorMessage(formData, failedValidation.message);
      switch (objKey) {
        case "name":
          formValidation.name = { isValid: false, message };
          break;
        case "description":
          formValidation.description = { isValid: false, message };
          break;
        default: {
          const _exhaustiveCheck: never = objKey;
          break;
        }
      }
    }
  });

  return formValidation;
};
