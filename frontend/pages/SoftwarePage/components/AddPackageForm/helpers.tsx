import React from "react";

import validator from "validator";

import { InstallType } from "interfaces/software";

// @ts-ignore
import validateQuery from "components/forms/validators/validate_query";

import { IAddPackageFormData, IFormValidation } from "./AddPackageForm";

type IAddPackageFormValidatorKey = Exclude<
  keyof IAddPackageFormData,
  "installScript" | "installType" | "selectedLabels" | "includeAnyLabels"
>;

type IMessageFunc = (formData: IAddPackageFormData) => string;
type IValidationMessage = string | IMessageFunc;

interface IValidation {
  name: string;
  isValid: (
    formData: IAddPackageFormData,
    enabledPreInstallCondition?: boolean,
    enabledPostInstallScript?: boolean
  ) => boolean;
  message?: IValidationMessage;
}

/** configuration defines validations for each filed in the form. It defines rules
 *  to determine if a field is valid, and rules for generating an error message.
 */
const FORM_VALIDATION_CONFIG: Record<
  IAddPackageFormValidatorKey,
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
  preInstallCondition: {
    validations: [
      {
        name: "required",
        isValid: (
          formData: IAddPackageFormData,
          enabledPreInstallCondition
        ) => {
          if (!enabledPreInstallCondition) {
            return true;
          }
          return (
            formData.preInstallCondition !== undefined &&
            !validator.isEmpty(formData.preInstallCondition)
          );
        },
        message: (formData) => {
          // we dont want an error message until the user has interacted with
          // the field. This is why we check for undefined here.
          if (formData.preInstallCondition === undefined) {
            return "";
          }
          return "Pre-install condition is required when enabled.";
        },
      },
      {
        name: "invalidQuery",
        isValid: (formData, enabledPreInstallCondition) => {
          if (!enabledPreInstallCondition) {
            return true;
          }
          return (
            formData.preInstallCondition !== undefined &&
            validateQuery(formData.preInstallCondition).valid
          );
        },
        message: (formData) =>
          validateQuery(formData.preInstallCondition).error,
      },
    ],
  },
  postInstallScript: {
    validations: [
      {
        name: "required",
        message: (formData) => {
          // we dont want an error message until the user has interacted with
          // the field. This is why we check for undefined here.
          if (formData.postInstallScript === undefined) {
            return "";
          }
          return "Post-install script is required when enabled.";
        },
        isValid: (formData, _, enabledPostInstallScript) => {
          if (!enabledPostInstallScript) {
            return true;
          }
          return (
            formData.postInstallScript !== undefined &&
            !validator.isEmpty(formData.postInstallScript)
          );
        },
      },
    ],
  },
  selfService: {
    // no validations related to self service
    validations: [],
  },
};

const getErrorMessage = (
  formData: IAddPackageFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

export const generateFormValidation = (
  formData: IAddPackageFormData,
  showingPreInstallCondition: boolean,
  showingPostInstallScript: boolean
) => {
  const formValidation: IFormValidation = {
    isValid: true,
    software: {
      isValid: false,
    },
  };

  Object.keys(FORM_VALIDATION_CONFIG).forEach((key) => {
    const objKey = key as keyof typeof FORM_VALIDATION_CONFIG;
    const failedValidation = FORM_VALIDATION_CONFIG[objKey].validations.find(
      (validation) =>
        !validation.isValid(
          formData,
          showingPreInstallCondition,
          showingPostInstallScript
        )
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

export const INSTALL_TYPE_OPTIONS = [
  {
    label: "Automatic",
    value: "automatic",
    helpText: "Install if not installed or an older version is installed.",
  },
  {
    label: "Manual",
    value: "manual",
    helpText: "Go to Host details page and manually install on each host.",
  },
];

export const LABEL_TARGET_MODES = [
  {
    label: "Include any",
    value: "include",
  },
  {
    label: "Exclude any",
    value: "exclude",
  },
];

export const LABEL_HELP_TEXT_CONFIG: Record<
  string,
  Record<InstallType, React.ReactNode>
> = {
  include: {
    automatic: (
      <>
        Software will only be installed on hosts that have <b>any</b> of these
        labels:
      </>
    ),
    manual: (
      <>
        Software will only be available for install on hosts that have{" "}
        <b>any</b> of these labels:
      </>
    ),
  },
  exclude: {
    automatic: (
      <>
        Software will only be installed on hosts that don&apos;t have <b>any</b>{" "}
        of these labels:
      </>
    ),
    manual: (
      <>
        Software will only be available for install on hosts that don&apos;t
        have <b>any</b> of these labels:
      </>
    ),
  },
} as const;

export default generateFormValidation;
