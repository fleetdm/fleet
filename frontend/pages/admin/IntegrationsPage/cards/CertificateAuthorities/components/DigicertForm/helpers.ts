import { ICertificatesIntegrationDigicert } from "interfaces/integration";

import valid_url from "components/forms/validators/valid_url";

import { IDigicertFormData } from "./DigicertForm";

// TODO: create a validator abstraction for this and the other form validation files

export interface IDigicertFormValidation {
  isValid: boolean;
  name?: { isValid: boolean; message?: string };
  url?: { isValid: boolean; message?: string };
  apiToken?: { isValid: boolean };
  profileId?: { isValid: boolean };
  commonName?: { isValid: boolean };
  certificateSeatId?: { isValid: boolean };
}

type IMessageFunc = <T>(formData: T) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<IDigicertFormValidation, "isValid">;

type IValidation<T extends Record<string, any>> = {
  name: string;
  isValid: (formData: T) => boolean;
  message?: IValidationMessage;
};

type IFormValidations<T extends Record<string, any>> = Record<
  keyof T,
  { validations: IValidation<T>[] }
>;

export const generateFormValidations = (
  digicertIntegrations: ICertificatesIntegrationDigicert[],
  isEditing: boolean
) => {
  const FORM_VALIDATIONS: IFormValidations<IDigicertFormData> = {
    name: {
      validations: [
        {
          name: "required",
          isValid: (formData: IDigicertFormData) => {
            return formData.name.length > 0;
          },
        },
        {
          name: "invalidCharacters",
          isValid: (formData: IDigicertFormData) => {
            return /^[a-zA-Z0-9_]+$/.test(formData.name);
          },
          message:
            "Invalid characters. Only letters, numbers and underscores allowed.",
        },
        {
          name: "unique",
          isValid: (formData: IDigicertFormData) => {
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
          isValid: (formData: IDigicertFormData) => {
            return formData.url.length > 0;
          },
        },
        {
          name: "validUrl",
          isValid: (formData: IDigicertFormData) => {
            return valid_url({ url: formData.url });
          },
          message: "Must be a valid URL.",
        },
      ],
    },
    apiToken: {
      validations: [
        {
          name: "required",
          isValid: (formData: IDigicertFormData) => {
            return formData.apiToken.length > 0;
          },
        },
      ],
    },
    profileId: {
      validations: [
        {
          name: "required",
          isValid: (formData: IDigicertFormData) => {
            return formData.profileId.length > 0;
          },
        },
      ],
    },
    commonName: {
      validations: [
        {
          name: "required",
          isValid: (formData: IDigicertFormData) => {
            return formData.commonName.length > 0;
          },
        },
      ],
    },
    certificateSeatId: {
      validations: [
        {
          name: "required",
          isValid: (formData: IDigicertFormData) => {
            return formData.certificateSeatId.length > 0;
          },
        },
      ],
    },
  };
  return FORM_VALIDATIONS;
};

const getErrorMessage = <T>(formData: T, message?: IValidationMessage) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

export const validateFormData = (
  formData: IDigicertFormData,
  validationConfig: IFormValidations
) => {
  const formValidation: IDigicertFormValidation = {
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

type IFieldValidation = {
  isValid: boolean;
  message?: string;
};

type IFormValidations2<T extends Record<string, any>> = {
  [K in keyof T]?: { validations: IValidation<T>[] };
};
type IValidationResult<T extends Record<string, any>> = {
  isValid: boolean;
} & {
  [K in keyof T]?: IFieldValidation;
};

export const useFormValidation = <T extends Record<string, any>>(
  validationConfig: IFormValidations2<T>
) => {
  const validateFormData2 = (formData: T) => {
    const formValidation: IValidationResult<T> = {
      isValid: true,
    } as IValidationResult<T>;

    Object.keys(validationConfig).forEach((key) => {
      const objKey = key as keyof typeof validationConfig;
      const failedValidation = validationConfig[objKey]?.validations.find(
        (validation) => !validation.isValid(formData)
      );
      if (!failedValidation) {
        (formValidation[objKey] as IFieldValidation) = {
          isValid: true,
        };
      } else {
        formValidation.isValid = false;
        (formValidation[objKey] as IFieldValidation) = {
          isValid: false,
          message: getErrorMessage(formData, failedValidation.message),
        };
      }
    });
    return formValidation;
  };
  return { validateFormData2 };
};
