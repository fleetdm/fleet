import { ICertificateAuthorityPartial } from "interfaces/certificates";

import valid_url from "components/forms/validators/valid_url";

import { ICustomESTFormData } from "./CustomESTForm";

export interface ICustomESTFormValidation {
  isValid: boolean;
  name?: { isValid: boolean; message?: string };
  url?: { isValid: boolean; message?: string };
  username?: { isValid: boolean; message?: string };
  password?: { isValid: boolean; message?: string };
}

type IMessageFunc = (formData: ICustomESTFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<ICustomESTFormValidation, "isValid">;

interface IValidation {
  name: string;
  isValid: (formData: ICustomESTFormData) => boolean;
  message?: IValidationMessage;
}

type IFormValidations = Record<
  IFormValidationKey,
  { validations: IValidation[] }
>;

export const generateFormValidations = (
  customESTIntegrations: ICertificateAuthorityPartial[],
  isEditing: boolean
) => {
  const FORM_VALIDATIONS: IFormValidations = {
    name: {
      validations: [
        {
          name: "required",
          isValid: (formData: ICustomESTFormData) => {
            return formData.name.length > 0;
          },
        },
        {
          name: "invalidCharacters",
          isValid: (formData: ICustomESTFormData) => {
            return /^[a-zA-Z0-9_]+$/.test(formData.name);
          },
          message:
            "Invalid characters. Only letters, numbers and underscores allowed.",
        },
        {
          name: "unique",
          isValid: (formData: ICustomESTFormData) => {
            return (
              isEditing ||
              customESTIntegrations.find(
                (cert) => cert.name === formData.name
              ) === undefined
            );
          },
          message: "Name is already used by another custom EST CA.",
        },
      ],
    },
    url: {
      validations: [
        {
          name: "required",
          isValid: (formData: ICustomESTFormData) => {
            return formData.url.length > 0;
          },
        },
        {
          name: "validUrl",
          isValid: (formData: ICustomESTFormData) => {
            return valid_url({ url: formData.url });
          },
          message: "Must be a valid URL.",
        },
      ],
    },
    username: {
      validations: [
        {
          name: "required",
          isValid: (formData: ICustomESTFormData) => {
            return formData.username.length > 0;
          },
        },
      ],
    },
    password: {
      validations: [
        {
          name: "required",
          isValid: (formData: ICustomESTFormData) => {
            return formData.password.length > 0;
          },
        },
      ],
    },
  };

  return FORM_VALIDATIONS;
};

const getErrorMessage = (
  formData: ICustomESTFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

// eslint-disable-next-line import/prefer-default-export
export const validateFormData = (
  formData: ICustomESTFormData,
  validationConfig: IFormValidations
) => {
  const formValidation: ICustomESTFormValidation = {
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
