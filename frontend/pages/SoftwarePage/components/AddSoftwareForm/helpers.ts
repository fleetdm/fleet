import validator from "validator";

// @ts-ignore
import validateQuery from "components/forms/validators/validate_query";
import { getPlatformDisplayName } from "utilities/file/fileUtils";

import { IAddSoftwareFormData, IFormValidation } from "./AddSoftwareForm";

type IAddSoftwareFormValidatorKey = Exclude<
  keyof IAddSoftwareFormData,
  "installScript"
>;

type IMessageFunc = (formData: IAddSoftwareFormData) => string;
type IValidationMessage = string | IMessageFunc;

interface IValidation {
  name: string;
  isValid: (
    formData: IAddSoftwareFormData,
    showingPreInstallCondition?: boolean,
    showingPostInstallScript?: boolean
  ) => boolean;
  message?: IValidationMessage;
}

/** configuration defines validations for each filed in the form. It defines rules
 *  to determine if a field is valid, and rules for generating an error message.
 */
const FORM_VALIDATION_CONFIG: Record<
  IAddSoftwareFormValidatorKey,
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
        message: (formData) => {
          // we dont want an error message until the user has interacted with
          // the field. This is why we check for undefined here.
          if (formData.preInstallCondition === undefined) {
            return "";
          }
          return "Pre-install condition is required when enabled.";
        },
        isValid: (
          formData: IAddSoftwareFormData,
          showingPreInstallCondition
        ) => {
          if (!showingPreInstallCondition) {
            return true;
          }
          return (
            formData.preInstallCondition !== undefined &&
            !validator.isEmpty(formData.preInstallCondition)
          );
        },
      },
      {
        name: "invalidQuery",
        message: (formData) =>
          validateQuery(formData.preInstallCondition).error,
        isValid: (formData, showingPreInstallCondition) => {
          if (!showingPreInstallCondition) {
            return true;
          }
          return (
            formData.preInstallCondition !== undefined &&
            validateQuery(formData.preInstallCondition).valid
          );
        },
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
        isValid: (formData, _, showingPostInstallScript) => {
          if (!showingPostInstallScript) {
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
};

const getErrorMessage = (
  formData: IAddSoftwareFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

export const generateFormValidation = (
  formData: IAddSoftwareFormData,
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

export const shouldDisableFormSubmit = (
  formData: IAddSoftwareFormData,
  showPreInstallCondition: boolean,
  showPostInstallScript: boolean
) => {
  const preInstallEnabledWithNoValue =
    showPreInstallCondition &&
    (formData.preInstallCondition === undefined ||
      formData.preInstallCondition === "");
  const postInstallEnabledWithNoValue =
    showPostInstallScript &&
    (formData.postInstallScript === undefined ||
      formData.postInstallScript === "");

  return (
    formData.software === null ||
    preInstallEnabledWithNoValue ||
    postInstallEnabledWithNoValue
  );
};

export const getFileDetails = (file: File) => {
  return {
    name: file.name,
    platform: getPlatformDisplayName(file),
  };
};

export const getInstallScript = (file: File) => {
  // TODO: get this dynamically
  return `sudo installer -pkg ${file.name} -target /`;
};
