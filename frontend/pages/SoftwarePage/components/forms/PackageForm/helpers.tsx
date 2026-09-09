import React from "react";

import { validateQuery } from "components/forms/validators/validate_query";
import { listNamesFromSelectedLabels } from "services/entities/labels";
import { getExtensionFromFileName } from "utilities/file/fileUtils";
import { encodeScriptBase64 } from "utilities/scripts_encoding";
import { getGitOpsModeTipContent } from "utilities/helpers";
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
        // Allow all SQL including empty SQL: this field never blocks form submission Request: #35058
        isValid: () => true,
        message: (formData) => {
          const query = formData.preInstallQuery;
          if (!query) {
            return "";
          }

          const { error } = validateQuery(query);
          // Return error text (or empty string)
          return error || "";
        },
      },
    ],
  },

  installScript: {
    validations: [
      {
        name: "requiredForExe",
        isValid: (formData) => {
          if (
            formData.software?.type === "exe" ||
            getExtensionFromFileName(formData.software?.name || "") === "exe"
          ) {
            // Handle undefined safely with nullish coalescing
            return (formData.installScript ?? "").trim().length > 0;
          }
          return true;
        },
        message: "Install script is required for .exe packages.",
      },
      {
        name: "requiredForZip",
        isValid: (formData) => {
          if (
            formData.software?.type === "zip" ||
            getExtensionFromFileName(formData.software?.name || "") === "zip"
          ) {
            // Handle undefined safely with nullish coalescing
            return (formData.installScript ?? "").trim().length > 0;
          }
          return true;
        },
        message: "Install script is required for .zip packages.",
      },
      {
        name: "requiredForTgz",
        isValid: (formData) => {
          if (
            formData.software?.name &&
            getExtensionFromFileName(formData.software.name) === "tar.gz"
          ) {
            // Handle undefined safely with nullish coalescing
            return (formData.installScript ?? "").trim().length > 0;
          }
          return true;
        },
        message: "Install script is required for .tar.gz archives.",
      },
    ],
  },
  uninstallScript: {
    validations: [
      {
        name: "requiredForExe",
        isValid: (formData) => {
          if (
            formData.software?.type === "exe" ||
            getExtensionFromFileName(formData.software?.name || "") === "exe"
          ) {
            // Handle undefined safely with nullish coalescing
            return (formData.uninstallScript ?? "").trim().length > 0;
          }
          return true;
        },
        message: "Uninstall script is required for .exe packages.",
      },
      {
        name: "requiredForZip",
        isValid: (formData) => {
          if (
            formData.software?.type === "zip" ||
            getExtensionFromFileName(formData.software?.name || "") === "zip"
          ) {
            // Handle undefined safely with nullish coalescing
            return (formData.uninstallScript ?? "").trim().length > 0;
          }
          return true;
        },
        message: "Uninstall script is required for .zip packages.",
      },
      {
        name: "requiredForTgz",
        isValid: (formData) => {
          if (
            formData.software?.name &&
            getExtensionFromFileName(formData.software.name) === "tar.gz"
          ) {
            // Handle undefined safely with nullish coalescing
            return (formData.uninstallScript ?? "").trim().length > 0;
          }
          return true;
        },
        message: "Uninstall script is required for .tar.gz archives.",
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
        // still compute error message for preInstallQuery since it can have warnings
        // of bad SQL but still allow form submission
        ...(objKey === "preInstallQuery" && {
          message: getErrorMessage(
            formData,
            FORM_VALIDATION_CONFIG[objKey].validations[0].message
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
  });

  return formValidation;
};

export const createTooltipContent = (
  formValidation: IPackageFormValidation,
  repoURL?: string,
  disabledFieldsForGitOps?: boolean
) => {
  if (disabledFieldsForGitOps && repoURL) {
    return getGitOpsModeTipContent(repoURL);
  }
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

/** Calculates the size of the payload, because the server limits the whole
 * request and not just the installer file. Not all fields are accounted for
 * in this calculation so if the final payload sent is over the size limit,
 * the server will reject it.
 */
export const estimateUploadSize = (formData: IPackageFormData) => {
  const scripts = [
    formData.installScript,
    formData.uninstallScript,
    formData.preInstallQuery,
    formData.postInstallScript,
  ];

  // The scripts are base64 encoded on the way out, so encode them to get the
  // length that actually goes over the wire.
  let scriptsSize = 0;
  scripts.forEach((script) => {
    scriptsSize += encodeScriptBase64(script)?.length || 0;
  });

  // The two flags are sent as "true" or "false".
  let fieldsSize =
    String(formData.selfService).length +
    String(formData.automaticInstall).length;

  // Names are user written, so count bytes. Anything outside ASCII takes more
  // than one.
  const encoder = new TextEncoder();

  formData.categories.forEach((category) => {
    fieldsSize += encoder.encode(category).length;
  });

  // Labels are only sent when the target is Custom, and only the selected ones.
  if (formData.targetType === "Custom") {
    listNamesFromSelectedLabels(formData.labelTargets).forEach((label) => {
      fieldsSize += encoder.encode(label).length;
    });
  }

  return (formData.software?.size || 0) + scriptsSize + fieldsSize;
};

export default generateFormValidation;
