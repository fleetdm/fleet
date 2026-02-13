import isUUID from "components/forms/validators/valid_uuid";

import { IAddTenantFormData } from "../AddEntraTenantModal/AddEntraTenantModal";

export interface IAddTenantFormValidation {
  isValid: boolean;
  tenantId?: { isValid: boolean; message?: string };
}

type IMessageFunc = (formData: IAddTenantFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<IAddTenantFormValidation, "isValid">;

interface IValidation {
  name: string;
  isValid: (formData: IAddTenantFormData) => boolean;
  message?: IValidationMessage;
}

type IFormValidations = Record<
  IFormValidationKey,
  { validations: IValidation[] }
>;

const FORM_VALIDATIONS: IFormValidations = {
  tenantId: {
    validations: [
      {
        name: "required",
        isValid: (formData: IAddTenantFormData) => {
          return formData.tenantId.length > 0;
        },
        message: `Tenant ID is required`,
      },
      {
        name: "validUUID",
        isValid: (formData: IAddTenantFormData) => {
          if (formData.tenantId.length === 0) {
            return true; // Skip this validation if name is empty
          }
          return isUUID(formData.tenantId);
        },
        message: "Invalid UUID. Please provide a valid UUID format.",
      },
    ],
  },
};

const getErrorMessage = (
  formData: IAddTenantFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

export const validateFormData = (formData: IAddTenantFormData) => {
  const formValidation: IAddTenantFormValidation = {
    isValid: true,
  };

  Object.keys(FORM_VALIDATIONS).forEach((key) => {
    const objKey = key as keyof typeof FORM_VALIDATIONS;
    const failedValidation = FORM_VALIDATIONS[objKey].validations.find(
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
