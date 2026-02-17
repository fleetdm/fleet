import { ILabelFormData } from "./LabelForm";

export interface ILabelFormValidation {
  isValid: boolean;
  name?: { isValid: boolean; message?: string };
  description?: { isValid: boolean; message?: string };
}

// Matches length in DB
const MAX_LABEL_NAME_LENGTH = 255;
const MAX_LABEL_DESCRIPTION_LENGTH = 255;

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
      {
        name: "notTooLong",
        isValid: (formData) => formData.name.length <= MAX_LABEL_NAME_LENGTH,
        message: `Name may not exceed ${MAX_LABEL_NAME_LENGTH} characters`,
      },
    ],
  },
  description: {
    validations: [
      {
        name: "notTooLong",
        isValid: (formData) =>
          !formData.description ||
          formData.description.length <= MAX_LABEL_DESCRIPTION_LENGTH,
        message: `Description may not exceed ${MAX_LABEL_DESCRIPTION_LENGTH} characters`,
      },
    ],
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

  Object.keys(FORM_VALIDATIONS).forEach((key) => {
    const objKey = key as keyof typeof FORM_VALIDATIONS;
    const failedValidation = FORM_VALIDATIONS[objKey].validations.find(
      (validation) => !validation.isValid(formData, formValidation)
    );

    if (!failedValidation) {
      formValidation[objKey] = { isValid: true };
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
