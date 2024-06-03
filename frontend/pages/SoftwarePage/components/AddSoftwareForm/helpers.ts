import validator from "validator";

// @ts-ignore
import validateQuery from "components/forms/validators/validate_query";

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
    enabledPreInstallCondition?: boolean,
    enabledPostInstallScript?: boolean
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
        isValid: (
          formData: IAddSoftwareFormData,
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

export default generateFormValidation;
