import { ICertificate } from "services/entities/certificates";
import { IAddCertFormData } from "./AddCertificateModal";

export interface IAddCertFormValidation {
  isValid: boolean;
  name?: { isValid: boolean; message?: string };
  certAuthorityId?: { isValid: boolean; message?: string };
  subjectName?: { isValid: boolean; message?: string };
}

export const INVALID_NAME_MSG =
  "Invalid characters. Only letters, numbers, spaces, dashes, and underscores allowed.";
export const USED_NAME_MSG = "Name is already used by another certificate.";
export const NAME_TOO_LONG_MSG = "Name is too long. Maximum is 255 characters.";

type IMessageFunc = (formData: IAddCertFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<IAddCertFormValidation, "isValid">;

interface IValidation {
  name: string;
  isValid: (formData: IAddCertFormData) => boolean;
  message?: IValidationMessage;
}

type IFormValidations = Record<
  IFormValidationKey,
  { validations: IValidation[] }
>;

export const generateFormValidations = (
  existingCerts: ICertificate[]
): IFormValidations => {
  const FORM_VALIDATIONS: IFormValidations = {
    name: {
      validations: [
        {
          name: "required",
          isValid: (formData: IAddCertFormData) => {
            return formData.name.trim().length > 0;
          },
        },
        {
          name: "invalidCharacters",
          isValid: (formData: IAddCertFormData) => {
            return /^[a-zA-Z0-9 \-_]+$/.test(formData.name);
          },
          message: INVALID_NAME_MSG,
        },
        {
          name: "unique",
          isValid: (formData: IAddCertFormData) => {
            return (
              existingCerts.find(
                (cert) =>
                  cert.name.toLowerCase() === formData.name.toLowerCase()
              ) === undefined
            );
          },
          message: USED_NAME_MSG,
        },
        {
          name: "maxLength",
          isValid: (formData: IAddCertFormData) => {
            return formData.name.length <= 255;
          },
          message: NAME_TOO_LONG_MSG,
        },
      ],
    },
    certAuthorityId: {
      validations: [
        {
          name: "required",
          isValid: (formData: IAddCertFormData) => {
            return formData.certAuthorityId !== "";
          },
          // no error message specified
        },
      ],
    },
    subjectName: {
      validations: [
        {
          name: "required",
          isValid: (formData: IAddCertFormData) => {
            return formData.subjectName.length > 0;
          },
        },
        // accept any value, let the server handle any errors
      ],
    },
  };
  return FORM_VALIDATIONS;
};

const getErrorMessage = (
  formData: IAddCertFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

export const validateFormData = (
  formData: IAddCertFormData,
  validationConfig: IFormValidations
): IAddCertFormValidation => {
  const formValidation: IAddCertFormValidation = {
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
