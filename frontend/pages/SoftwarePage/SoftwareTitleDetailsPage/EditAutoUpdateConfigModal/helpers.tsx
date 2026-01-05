import { ISoftwareAutoUpdateConfigFormData } from "./EditAutoUpdateConfigModal";

export interface ISoftwareAutoUpdateConfigInputValidation {
  isValid: boolean;
  message?: string;
}

export interface ISoftwareAutoUpdateConfigFormValidation {
  isValid: boolean;
  autoUpdateStartTime?: ISoftwareAutoUpdateConfigInputValidation;
  autoUpdateEndTime?: ISoftwareAutoUpdateConfigInputValidation;
  targets?: ISoftwareAutoUpdateConfigInputValidation;
  windowLength?: ISoftwareAutoUpdateConfigInputValidation;
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

const validateWindowLength = (
  formData: ISoftwareAutoUpdateConfigFormData,
  validations?: ISoftwareAutoUpdateConfigFormValidation
) => {
  if (
    formData.autoUpdateStartTime.length === 0 ||
    formData.autoUpdateEndTime.length === 0 ||
    !validations?.autoUpdateStartTime ||
    !validations.autoUpdateStartTime.isValid ||
    !validations.autoUpdateEndTime ||
    !validations.autoUpdateEndTime.isValid
  ) {
    return true; // Skip this validation if startTime is invalid
  }
  const [startHours, startMinutes] = formData.autoUpdateStartTime
    .split(":")
    .map(Number);
  const [endHours, endMinutes] = formData.autoUpdateEndTime
    .split(":")
    .map(Number);
  const startTotalMinutes = startHours * 60 + startMinutes;
  const endTotalMinutes = endHours * 60 + endMinutes;
  return (
    endTotalMinutes < startTotalMinutes ||
    endTotalMinutes - startTotalMinutes >= 60
  );
};

const FORM_VALIDATIONS: IFormValidations = {
  autoUpdateStartTime: {
    validations: [
      {
        name: "required",
        isValid: (formData: ISoftwareAutoUpdateConfigFormData) => {
          return formData.autoUpdateStartTime.length > 0;
        },
        message: `Earliest start time is required`,
      },
      {
        name: "valid",
        isValid: (formData: ISoftwareAutoUpdateConfigFormData) => {
          if (formData.autoUpdateStartTime.length === 0) {
            return true; // Skip this validation if startTime is empty
          }
          return validateTimeFormat(formData.autoUpdateStartTime);
        },
        message: `Use HH:MM format (24-hour clock)`,
      },
    ],
  },
  autoUpdateEndTime: {
    validations: [
      {
        name: "required",
        isValid: (formData: ISoftwareAutoUpdateConfigFormData) => {
          return formData.autoUpdateEndTime.length > 0;
        },
        message: `Latest start time is required`,
      },
      {
        name: "valid",
        isValid: (formData: ISoftwareAutoUpdateConfigFormData) => {
          if (formData.autoUpdateEndTime.length === 0) {
            return true; // Skip this validation if endTime is empty
          }
          return validateTimeFormat(formData.autoUpdateEndTime);
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
  windowLength: {
    validations: [
      {
        name: "minimum_length",
        isValid: validateWindowLength,
        message: `Update window must be at least 60 minutes long`,
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
    if (!formData.autoUpdateEnabled && key !== "targets") {
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
