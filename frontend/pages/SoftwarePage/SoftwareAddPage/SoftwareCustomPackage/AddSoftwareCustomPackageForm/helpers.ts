// @ts-ignore
import validateQuery from "components/forms/validators/validate_query";

import {
  ICustomPackageAppFormData,
  IFormValidation,
} from "./AddSoftwareCustomPackageForm";

type IMessageFunc = (formData: ICustomPackageAppFormData) => string;
type IValidationMessage = string | IMessageFunc;

interface IValidation {
  name: string;
  isValid: (formData: ICustomPackageAppFormData) => boolean;
  message?: IValidationMessage;
}

const FORM_VALIDATION_CONFIG: Record<
  "preInstallQuery",
  { validations: IValidation[] }
> = {
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
};

const getErrorMessage = (
  formData: ICustomPackageAppFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

// eslint-disable-next-line import/prefer-default-export
export const generateFormValidation = (formData: ICustomPackageAppFormData) => {
  const formValidation: IFormValidation = {
    isValid: true,
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
