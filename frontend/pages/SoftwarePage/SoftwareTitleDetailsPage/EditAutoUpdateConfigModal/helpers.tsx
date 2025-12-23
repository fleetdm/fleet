import { ISoftwareAutoUpdateConfigFormData } from "./EditAutoUpdateConfigModal";

export interface ISoftwareAutoUpdateConfigInputValidation {
  isValid: boolean;
  message?: string;
}

export interface ISoftwareAutoUpdateConfigFormValidation {
  isValid: boolean;
  startTime?: ISoftwareAutoUpdateConfigInputValidation;
  endTime?: ISoftwareAutoUpdateConfigInputValidation;
  targets?: ISoftwareAutoUpdateConfigInputValidation;
}

type IMessageFunc = (formData: ISoftwareAutoUpdateConfigFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<
  ISoftwareAutoUpdateConfigFormValidation,
  "isValid"
>;

interface IValidation {
  name: string;
  isValid: (
    formData: ISoftwareAutoUpdateConfigFormData,
    validations?: ISoftwareAutoUpdateConfigFormValidation
  ) => boolean;
  message?: IValidationMessage;
}

type IFormValidations = Record<
  IFormValidationKey,
  { validations: IValidation[] }
>;

const validateTimeFormat = (time: string): boolean => {
  if (!time.match(/^[0-9]{2}:[0-9]{2}$/)) {
    return false;
  }
  const [hours, minutes] = time.split(":").map(Number);
  if (hours < 0 || hours > 23 || minutes < 0 || minutes > 59) {
    return false;
  }
  return true;
};

const FORM_VALIDATIONS: IFormValidations = {
  startTime: {
    validations: [
      {
        name: "required",
        isValid: (formData: ISoftwareAutoUpdateConfigFormData) => {
          return formData.startTime.length > 0;
        },
        message: `Earliest start time is required`,
      },
      {
        name: "valid",
        isValid: (formData: ISoftwareAutoUpdateConfigFormData) => {
          if (formData.startTime.length === 0) {
            return true; // Skip this validation if startTime is empty
          }
          return validateTimeFormat(formData.startTime);
        },
        message: `Use HH:MM format (24-hour clock)`,
      },
    ],
  },
  endTime: {
    validations: [
      {
        name: "required",
        isValid: (formData: ISoftwareAutoUpdateConfigFormData) => {
          return formData.endTime.length > 0;
        },
        message: `Latest start time is required`,
      },
      {
        name: "valid",
        isValid: (formData: ISoftwareAutoUpdateConfigFormData) => {
          if (formData.endTime.length === 0) {
            return true; // Skip this validation if endTime is empty
          }
          return validateTimeFormat(formData.endTime);
        },
        message: `Use HH:MM format (24-hour clock)`,
      },
    ],
  },
  targets: {
    validations: [
      {
        name: "custom_labels_selected",
        isValid: (formData: ISoftwareAutoUpdateConfigFormData) => {
          return (
            formData.targetType !== "Custom" ||
            Object.values(formData.labelTargets).filter((v) => v).length > 0
          );
        },
        message: `At least one label target must be selected`,
      },
    ],
  },
};

const getErrorMessage = (
  formData: ISoftwareAutoUpdateConfigFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

export const validateFormData = (
  formData: ISoftwareAutoUpdateConfigFormData,
  isSaving = false
) => {
  const formValidation: ISoftwareAutoUpdateConfigFormValidation = {
    isValid: true,
  };
  // If auto updates are not enabled, skip further validations.
  Object.keys(FORM_VALIDATIONS).forEach((key) => {
    if (!formData.enabled && key !== "targets") {
      return;
    }
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
