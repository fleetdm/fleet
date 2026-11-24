import { validateQuery } from "components/forms/validators/validate_query";

import {
  IFleetMaintainedAppFormData,
  IFormValidation,
} from "./FleetAppDetailsForm";

type IMessageFunc = (formData: IFleetMaintainedAppFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<IFormValidation, "isValid">;

interface IValidation {
  name: string;
  isValid: (formData: IFleetMaintainedAppFormData) => boolean;
  message?: IValidationMessage;
}

const FORM_VALIDATION_CONFIG: Record<
  IFormValidationKey,
  { validations: IValidation[] }
> = {
  preInstallQuery: {
    validations: [
      {
        name: "invalidQuery",
        // Advisory only: a SQL syntax error surfaces the inline warning but
        // does not block submission, matching PackageForm's pre-install query.
        isValid: () => true,
        message: (formData) => {
          const query = formData.preInstallQuery ?? "";

          if (query.trim() === "") {
            return "";
          }

          const { error } = validateQuery(query);
          return error ?? "";
        },
      },
    ],
  },
  customTarget: {
    validations: [
      {
        name: "requiredLabelTargets",
        isValid: (formData) => {
          if (formData.targetType === "All hosts") return true;
          // there must be at least one label target selected
          return (
            Object.keys(formData.labelTargets).find(
              (key) => formData.labelTargets[key]
            ) !== undefined
          );
        },
      },
    ],
  },
};

const getErrorMessage = (
  formData: IFleetMaintainedAppFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

// eslint-disable-next-line import/prefer-default-export
export const generateFormValidation = (
  formData: IFleetMaintainedAppFormData
) => {
  const formValidation: IFormValidation = {
    isValid: true,
  };

  (Object.keys(FORM_VALIDATION_CONFIG) as IFormValidationKey[]).forEach(
    (objKey) => {
      const failedValidation = FORM_VALIDATION_CONFIG[objKey].validations.find(
        (validation) => !validation.isValid(formData)
      );

      if (!failedValidation) {
        formValidation[objKey] = {
          isValid: true,
          // preInstallQuery is advisory — compute the message anyway so a
          // SQL syntax error still shows inline without blocking submission.
          ...(objKey === "preInstallQuery" && {
            message: getErrorMessage(
              formData,
              FORM_VALIDATION_CONFIG.preInstallQuery.validations[0].message
            ),
          }),
        };
      } else {
        formValidation.isValid = false;
        formValidation[objKey] = {
          isValid: false,
          message: getErrorMessage(formData, failedValidation.message),
        };
      }
    }
  );

  return formValidation;
};
