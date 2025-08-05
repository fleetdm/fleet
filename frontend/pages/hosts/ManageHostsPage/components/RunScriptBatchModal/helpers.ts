import { IRunScriptBatchModalScheduleFormData } from "./RunScriptBatchModal";

// TODO: create a validator abstraction for this and the other form validation files

export interface IRunScriptBatchModalFormValidation {
  isValid: boolean;
  date?: { isValid: boolean; message?: string };
  time?: { isValid: boolean; message?: string };
}

type IMessageFunc = (formData: IRunScriptBatchModalScheduleFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<
  IRunScriptBatchModalScheduleFormData,
  "isValid"
>;

interface IValidation {
  name: string;
  isValid: (formData: IRunScriptBatchModalScheduleFormData) => boolean;
  message?: IValidationMessage;
}

type IFormValidations = Record<
  IFormValidationKey,
  { validations: IValidation[] }
>;

const FORM_VALIDATIONS: IFormValidations = {
  date: {
    validations: [
      {
        name: "required",
        isValid: (formData: IRunScriptBatchModalScheduleFormData) => {
          return formData.date.length > 0;
        },
      },
      {
        name: "invalidCharacters",
        isValid: (formData: IRunScriptBatchModalScheduleFormData) => {
          return /^[a-zA-Z0-9_]+$/.test(formData.date);
        },
        message:
          "Invalid characters. Only letters, numbers and underscores allowed.",
      },
    ],
  },
  time: {
    validations: [
      {
        name: "required",
        isValid: (formData: IRunScriptBatchModalScheduleFormData) => {
          return formData.time.length > 0;
        },
      },
    ],
  },
};

const getErrorMessage = (
  formData: IRunScriptBatchModalScheduleFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

export const validateFormData = (
  formData: IRunScriptBatchModalScheduleFormData
) => {
  const formValidation: IRunScriptBatchModalFormValidation = {
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
