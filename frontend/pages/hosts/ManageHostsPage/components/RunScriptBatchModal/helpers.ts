import { parse, isValid } from "date-fns";
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
  isValid: (
    formData: IRunScriptBatchModalScheduleFormData,
    validations?: IRunScriptBatchModalFormValidation
  ) => boolean;
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
        name: "validDate",
        isValid: (formData: IRunScriptBatchModalScheduleFormData) => {
          if (!formData.date.match(/^\d{4}-\d{2}-\d{2}$/)) {
            return false;
          }
          const parsedDate = parse(formData.date, "yyyy-MM-dd", new Date());
          return isValid(parsedDate);
        },
        message: "Date (UTC) must have valid format",
      },
      {
        name: "notInPast",
        isValid: (formData: IRunScriptBatchModalScheduleFormData) => {
          const now = new Date();
          const parsedDate = parse(
            `${formData.date} 23:59:59.999`,
            "yyyy-MM-dd HH:mm:ss.SSS",
            now
          );
          return parsedDate >= now;
        },
        message: `Date (UTC) cannot be in the past`,
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
      {
        name: "validTime",
        isValid: (formData: IRunScriptBatchModalScheduleFormData) => {
          if (!formData.time.match(/^\d{2}:\d{2}$/)) {
            return false;
          }
          const parsedDate = parse(
            `1982-10-13 ${formData.time}`,
            "yyyy-MM-dd HH:mm",
            new Date()
          );
          return isValid(parsedDate);
        },
        message: "Time (UTC) must have valid format",
      },
      {
        name: "notInPast",
        isValid: (
          formData: IRunScriptBatchModalScheduleFormData,
          validations?: IRunScriptBatchModalFormValidation
        ) => {
          if (validations?.date?.isValid === false) {
            return true; // If date is invalid, skip time validation
          }
          const parsedDate = parse(
            `${formData.date} ${formData.time}`,
            "yyyy-MM-dd HH:mm",
            new Date()
          );
          return parsedDate >= new Date();
        },
        message: `Time (UTC) cannot be in the past`,
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
  formData: IRunScriptBatchModalScheduleFormData,
  runMode: "run_now" | "schedule" = "run_now"
) => {
  const formValidation: IRunScriptBatchModalFormValidation = {
    isValid: true,
  };
  if (runMode === "run_now") {
    return formValidation; // No validation needed for run now
  }

  Object.keys(FORM_VALIDATIONS).forEach((key) => {
    const objKey = key as keyof typeof FORM_VALIDATIONS;
    const failedValidation = FORM_VALIDATIONS[objKey].validations.find(
      (validation) => !validation.isValid(formData, formValidation)
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
