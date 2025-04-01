import React from "react";

// @ts-ignore
import validateQuery from "components/forms/validators/validate_query";

import { IPackageFormData, IPackageFormValidation } from "./PackageForm";

type IMessageFunc = (formData: IPackageFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<IPackageFormValidation, "isValid">;

interface IValidation {
  name: string;
  isValid: (formData: IPackageFormData) => boolean;
  message?: IValidationMessage;
}

/** configuration defines validations for each field in the form. It defines rules
 *  to determine if a field is valid, and rules for generating an error message.
 */
const FORM_VALIDATION_CONFIG: Record<
  IFormValidationKey,
  { validations: IValidation[] }
> = {
  software: {
    validations: [
      {
        name: "required",
        isValid: (formData) => formData.software !== null,
      },
    ],
  },
  preInstallQuery: {
    validations: [
      {
        name: "invalidQuery",
        isValid: (formData) => {
          const query = formData.preInstallQuery;
          return (
            query === undefined || query === "" || validateQuery(query).valid
          );
        },
        message: (formData) => validateQuery(formData.preInstallQuery).error,
      },
    ],
  },
  installScript: {
    validations: [
      {
        name: "requiredForExe",
        isValid: (formData) => {
          if (formData.software?.type === "exe") {
            // Handle undefined safely with nullish coalescing
            return (formData.installScript ?? "").trim().length > 0;
          }
          return true;
        },
        message: "Install script is required for .exe files.",
      },
    ],
  },
  uninstallScript: {
    validations: [
      {
        name: "requiredForExe",
        isValid: (formData) => {
          if (formData.software?.type === "exe") {
            // Handle undefined safely with nullish coalescing
            return (formData.uninstallScript ?? "").trim().length > 0;
          }
          return true;
        },
        message: "Uninstall script is required for .exe files.",
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
  formData: IPackageFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

export const generateFormValidation = (formData: IPackageFormData) => {
  const formValidation: IPackageFormValidation = {
    isValid: true,
    software: {
      isValid: false,
    },
  };

  Object.keys(FORM_VALIDATION_CONFIG).forEach((key) => {
    const objKey = key as keyof typeof FORM_VALIDATION_CONFIG;
    const failedValidation = FORM_VALIDATION_CONFIG[objKey].validations.find(
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

export const createTooltipContent = (
  formValidation: IPackageFormValidation
) => {
  const messages = Object.values(formValidation)
    .filter((field) => field.isValid === false && field.message)
    .map((field) => field.message);

  if (messages.length === 0) {
    return null;
  }

  return (
    <>
      {messages.map((message, index) => (
        <>
          {message}
          {index < messages.length - 1 && <br />}
        </>
      ))}
    </>
  );
};

export default generateFormValidation;
