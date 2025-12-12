import { IAddSecretModalScheduleFormData } from "./AddSecretModal";

// TODO: create a validator abstraction for this and the other form validation files

export interface IAddSecretModalFormValidation {
  isValid: boolean;
  name?: { isValid: boolean; message?: string };
  value?: { isValid: boolean; message?: string };
}

type IMessageFunc = (formData: IAddSecretModalScheduleFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<
  IAddSecretModalScheduleFormData,
  "isValid"
>;

interface IValidation {
  name: string;
  isValid: (
    formData: IAddSecretModalScheduleFormData,
    validations?: IAddSecretModalFormValidation
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
        isValid: (formData: IAddSecretModalScheduleFormData) => {
          return formData.name.length > 0;
        },
        message: `Name is required`,
      },
      {
        name: "validName",
        isValid: (formData: IAddSecretModalScheduleFormData) => {
          if (formData.name.length === 0) {
            return true; // Skip this validation if name is empty
          }
          return !!formData.name.match(/^[a-zA-Z0-9_]+$/);
        },
        message:
          "Name may only include uppercase letters, numbers, and underscores",
      },
      {
        name: "notTooLong",
        isValid: (formData: IAddSecretModalScheduleFormData) => {
          return formData.name.length <= 255;
        },
        message: "Name may not exceed 255 characters",
      },
      {
        name: "doesNotIncludePrefix",
        isValid: (formData: IAddSecretModalScheduleFormData) => {
          return !formData.name.match(/^FLEET_SECRET_/);
        },
        message: `Name should not include variable prefix`,
      },
    ],
  },
  value: {
    validations: [
      {
        name: "required",
        isValid: (formData: IAddSecretModalScheduleFormData) => {
          return formData.value.length > 0;
        },
        message: `Value is required`,
      },
    ],
  },
};

const getErrorMessage = (
  formData: IAddSecretModalScheduleFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

export const validateFormData = (
  formData: IAddSecretModalScheduleFormData,
  isSaving = false
) => {
  const formValidation: IAddSecretModalFormValidation = {
    isValid: true,
  };
  Object.keys(FORM_VALIDATIONS).forEach((key) => {
    const objKey = key as keyof typeof FORM_VALIDATIONS;
    const failedValidation = FORM_VALIDATIONS[objKey].validations.find(
      (validation) => {
        if (!isSaving && validation.name === "required") {
          return false; // Skip this validation if not saving
        }
        return !validation.isValid(formData, formValidation);
      }
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
