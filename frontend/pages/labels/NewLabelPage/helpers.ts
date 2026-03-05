import { INewLabelFormData } from "./NewLabelPage";

export interface INewLabelFormValidation {
  isValid: boolean;
  name?: { isValid: boolean; message?: string };
  description?: { isValid: boolean; message?: string };
  labelQuery?: { isValid: boolean; message?: string };
  criteria?: { isValid: boolean; message?: string };
}

// Matches DB
const MAX_LABEL_NAME_LENGTH = 255;
const MAX_DESCRIPTION_LENGTH = 255;

type IMessageFunc = (formData: INewLabelFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Pick<
  INewLabelFormData,
  "name" | "description" | "labelQuery" | "vitalValue"
>;

interface IValidation {
  name: string;
  isValid: (
    formData: INewLabelFormData,
    validations?: INewLabelFormValidation
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
          formData.description.length <= MAX_DESCRIPTION_LENGTH,
        message: `Description may not exceed ${MAX_DESCRIPTION_LENGTH} characters`,
      },
    ],
  },
  labelQuery: {
    validations: [
      {
        name: "requiredForDynamic",
        isValid: (formData) => {
          if (formData.type !== "dynamic") {
            return true;
          }
          return formData.labelQuery.trim().length > 0;
        },
        message: "Query text must be present",
      },
    ],
  },
  vitalValue: {
    validations: [
      {
        name: "requiredForHostVitals",
        isValid: (formData) => {
          if (formData.type !== "host_vitals") {
            return true;
          }
          return formData.vitalValue.trim().length > 0;
        },
        message: "Label criteria must be completed",
      },
    ],
  },
};

const getErrorMessage = (
  formData: INewLabelFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

export const validateNewLabelFormData = (
  formData: INewLabelFormData
): INewLabelFormValidation => {
  const formValidation: INewLabelFormValidation = { isValid: true };

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
        case "labelQuery":
          formValidation.labelQuery = { isValid: true };
          break;
        case "vitalValue":
          formValidation.criteria = { isValid: true };
          break;
        default: {
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
        case "labelQuery":
          formValidation.labelQuery = { isValid: false, message };
          break;
        case "vitalValue":
          formValidation.criteria = { isValid: false, message };
          break;
        default: {
          break;
        }
      }
    }
  });

  return formValidation;
};
