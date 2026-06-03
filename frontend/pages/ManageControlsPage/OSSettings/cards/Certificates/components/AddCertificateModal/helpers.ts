import { ICertificate } from "services/entities/certificates";
import { IAddCertFormData } from "./AddCertificateModal";

export interface IAddCertFormValidation {
  isValid: boolean;
  name?: { isValid: boolean; message?: string };
  certAuthorityId?: { isValid: boolean; message?: string };
  subjectName?: { isValid: boolean; message?: string };
  subjectAlternativeName?: { isValid: boolean; message?: string };
}

export const INVALID_NAME_MSG =
  "Invalid characters. Only letters, numbers, spaces, dashes, and underscores allowed.";
export const USED_NAME_MSG = "Name is already used by another certificate.";
export const NAME_TOO_LONG_MSG = "Name is too long. Maximum is 255 characters.";
export const NAME_REQUIRED_MSG = "Name must be completed.";
export const CA_REQUIRED_MSG = "Certificate authority must be completed.";
export const SUBJECT_NAME_REQUIRED_MSG = "Subject name must be completed.";

type IMessageFunc = (formData: IAddCertFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<IAddCertFormValidation, "isValid">;

interface IValidation {
  name: string;
  isValid: (formData: IAddCertFormData) => boolean;
  message?: IValidationMessage;
  // required validations only render their message after the first submit attempt;
  // non-required (format) errors render as the user types.
  required?: boolean;
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
          required: true,
          isValid: (formData: IAddCertFormData) => {
            return formData.name.trim().length > 0;
          },
          message: NAME_REQUIRED_MSG,
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
          required: true,
          isValid: (formData: IAddCertFormData) => {
            return formData.certAuthorityId !== "";
          },
          message: CA_REQUIRED_MSG,
        },
      ],
    },
    subjectName: {
      validations: [
        {
          name: "required",
          required: true,
          isValid: (formData: IAddCertFormData) => {
            return formData.subjectName.trim().length > 0;
          },
          message: SUBJECT_NAME_REQUIRED_MSG,
        },
        // accept any value, let the server handle any errors
      ],
    },
    // SAN is optional; format and length are validated server-side and surfaced
    // back to the user via the 422 error path in AddCertificateModal.tsx.
    subjectAlternativeName: { validations: [] },
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
  validationConfig: IFormValidations,
  attemptedSubmit = false
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
      const suppressMessage = failedValidation.required && !attemptedSubmit;
      formValidation[objKey] = {
        isValid: false,
        message: suppressMessage
          ? undefined
          : getErrorMessage(formData, failedValidation.message),
      };
    }
  });

  return formValidation;
};
