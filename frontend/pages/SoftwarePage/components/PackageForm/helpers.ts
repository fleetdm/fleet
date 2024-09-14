// @ts-ignore
import validateQuery from "components/forms/validators/validate_query";

import { IPackageFormData, IFormValidation } from "./PackageForm";

type IPackageFormValidatorKey = Exclude<
  keyof IPackageFormData,
  "installScript" | "uninstallScript"
>;

type IMessageFunc = (formData: IPackageFormData) => string;
type IValidationMessage = string | IMessageFunc;

interface IValidation {
  name: string;
  isValid: (formData: IPackageFormData) => boolean;
  message?: IValidationMessage;
}

/** configuration defines validations for each filed in the form. It defines rules
 *  to determine if a field is valid, and rules for generating an error message.
 */
const FORM_VALIDATION_CONFIG: Record<
  IPackageFormValidatorKey,
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
  postInstallScript: {
    // no validations related to postInstallScript
    validations: [],
  },
  selfService: {
    // no validations related to self service
    validations: [],
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
  const formValidation: IFormValidation = {
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

export default generateFormValidation;
