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

type IMessageFunc = (formData: IDigicertFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<IDigicertFormValidation, "isValid">;

interface IValidation {
  name: string;
  isValid: (formData: IDigicertFormData) => boolean;
  message?: IValidationMessage;
}

const FORM_VALIDATIONS: Record<
  IFormValidationKey,
  { validations: IValidation[] }
> = {
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
          "Inalid characters. Only letters, numbers and underscores allowed.",
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
        message: (formData: IDigicertFormData) =>
          `${formData.url} is not a valid URL`,
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

const getErrorMessage = (
  formData: IDigicertFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

// eslint-disable-next-line import/prefer-default-export
export const validateFormData = (formData: IDigicertFormData) => {
  const formValidation: IDigicertFormValidation = {
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
