import { ICertificateAuthorityPartial } from "interfaces/certificates";

import valid_url from "components/forms/validators/valid_url";

import { ISmallstepFormData } from "./SmallstepForm";

// TODO: create a validator abstraction for this and the other form validation files

export interface ISmallstepFormValidation {
  isValid: boolean;
  name?: { isValid: boolean; message?: string };
  scepURL?: { isValid: boolean; message?: string };
  challengeURL?: { isValid: boolean; message?: string };
  username?: { isValid: boolean; message?: string };
  password?: { isValid: boolean; message?: string };
}

type IMessageFunc = (formData: ISmallstepFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<ISmallstepFormValidation, "isValid">;

interface IValidation {
  name: string;
  isValid: (formData: ISmallstepFormData) => boolean;
  message?: IValidationMessage;
}

type IFormValidations = Record<
  IFormValidationKey,
  { validations: IValidation[] }
>;

export const generateFormValidations = (
  smallstepIntegrations: ICertificateAuthorityPartial[],
  isEditing: boolean
) => {
  const FORM_VALIDATIONS: IFormValidations = {
    name: {
      validations: [
        {
          name: "required",
          isValid: (formData: ISmallstepFormData) => {
            return formData.name.length > 0;
          },
        },
        {
          name: "invalidCharacters",
          isValid: (formData: ISmallstepFormData) => {
            return /^[a-zA-Z0-9_]+$/.test(formData.name);
          },
          message:
            "Invalid characters. Only letters, numbers and underscores allowed.",
        },
        {
          name: "unique",
          isValid: (formData: ISmallstepFormData) => {
            return (
              isEditing ||
              smallstepIntegrations.find(
                (cert) => cert.name === formData.name
              ) === undefined
            );
          },
          message: "Name is already used by another custom SCEP CA.",
        },
      ],
    },
    scepURL: {
      validations: [
        {
          name: "required",
          isValid: (formData: ISmallstepFormData) => {
            return formData.scepURL.length > 0;
          },
        },
        {
          name: "validUrl",
          isValid: (formData: ISmallstepFormData) => {
            return valid_url({ url: formData.scepURL });
          },
          message: "Must be a valid URL.",
        },
      ],
    },
    challengeURL: {
      validations: [
        {
          name: "required",
          isValid: (formData: ISmallstepFormData) => {
            return formData.challengeURL.length > 0;
          },
        },
        {
          name: "validUrl",
          isValid: (formData: ISmallstepFormData) => {
            return valid_url({ url: formData.challengeURL });
          },
          message: "Must be a valid URL.",
        },
      ],
    },
    username: {
      validations: [
        {
          name: "required",
          isValid: (formData: ISmallstepFormData) => {
            return formData.username.length > 0;
          },
        },
      ],
    },
    password: {
      validations: [
        {
          name: "required",
          isValid: (formData: ISmallstepFormData) => {
            return formData.password.length > 0;
          },
        },
      ],
    },
  };

  return FORM_VALIDATIONS;
};

const getErrorMessage = (
  formData: ISmallstepFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

// eslint-disable-next-line import/prefer-default-export
export const validateFormData = (
  formData: ISmallstepFormData,
  validationConfig: IFormValidations
) => {
  const formValidation: ISmallstepFormValidation = {
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
