import { ICertificatesIntegrationDigicert } from "interfaces/integration";

import valid_url from "components/forms/validators/valid_url";

import { IHydrantFormData } from "./HydrantForm";

// TODO: create a validator abstraction for this and the other form validation files

export interface IHydrantFormValidation {
  isValid: boolean;
  name?: { isValid: boolean; message?: string };
  url?: { isValid: boolean; message?: string };
  clientId?: {
    isValid: boolean;
    message?: string;
  };
  clientSecret?: { isValid: boolean; message?: string };
}

type IMessageFunc = (formData: IHydrantFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<IHydrantFormData, "isValid">;

interface IValidation {
  name: string;
  isValid: (formData: IHydrantFormData) => boolean;
  message?: IValidationMessage;
}

type IFormValidations = Record<
  IFormValidationKey,
  { validations: IValidation[] }
>;

export const generateFormValidations = (
  digicertIntegrations: ICertificatesIntegrationDigicert[],
  isEditing: boolean
) => {
  const FORM_VALIDATIONS: IFormValidations = {
    name: {
      validations: [
        {
          name: "required",
          isValid: (formData: IHydrantFormData) => {
            return formData.name.length > 0;
          },
        },
        {
          name: "invalidCharacters",
          isValid: (formData: IHydrantFormData) => {
            return /^[a-zA-Z0-9_]+$/.test(formData.name);
          },
          message:
            "Invalid characters. Only letters, numbers and underscores allowed.",
        },
        {
          name: "unique",
          isValid: (formData: IHydrantFormData) => {
            return (
              isEditing ||
              digicertIntegrations.find(
                (cert) => cert.name === formData.name
              ) === undefined
            );
          },
          message: "Name is already used by another DigiCert CA.",
        },
      ],
    },
    url: {
      validations: [
        {
          name: "required",
          isValid: (formData: IHydrantFormData) => {
            return formData.url.length > 0;
          },
        },
        {
          name: "validUrl",
          isValid: (formData: IHydrantFormData) => {
            return valid_url({ url: formData.url });
          },
          message: "Must be a valid URL.",
        },
      ],
    },
    clientId: {
      validations: [
        {
          name: "required",
          isValid: (formData: IHydrantFormData) => {
            return formData.clientId.length > 0;
          },
        },
      ],
    },
    clientSecret: {
      validations: [
        {
          name: "required",
          isValid: (formData: IHydrantFormData) => {
            return formData.clientSecret.length > 0;
          },
        },
      ],
    },
  };
  return FORM_VALIDATIONS;
};

const getErrorMessage = (
  formData: IHydrantFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

export const validateFormData = (
  formData: IHydrantFormData,
  validationConfig: IFormValidations
) => {
  const formValidation: IHydrantFormValidation = {
    isValid: true,
  };

  Object.keys(validationConfig).forEach((key) => {
    const objKey = key as keyof typeof validationConfig;
    const failedValidation = validationConfig[objKey].validations.find(
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
