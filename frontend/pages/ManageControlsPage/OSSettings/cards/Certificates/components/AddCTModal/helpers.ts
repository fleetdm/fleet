import { ICertTemplate } from "services/entities/certificates";
import { IAddCTFormData } from "./AddCTModal";

export interface IAddCTFormValidation {
  isValid: boolean;
  name?: { isValid: boolean; message?: string };
  certAuthorityId?: { isValid: boolean; message?: string };
  subjectName?: { isValid: boolean; message?: string };
}

type IMessageFunc = (formData: IAddCTFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<IAddCTFormValidation, "isValid">;

interface IValidation {
  name: string;
  isValid: (formData: IAddCTFormData) => boolean;
  message?: IValidationMessage;
}

type IFormValidations = Record<
  IFormValidationKey,
  { validations: IValidation[] }
>;

export const generateFormValidations = (
  existingCTs: ICertTemplate[]
): IFormValidations => {
  const FORM_VALIDATIONS: IFormValidations = {
    name: {
      validations: [
        {
          name: "required",
          isValid: (formData: IAddCTFormData) => {
            return formData.name.length > 0;
          },
        },
        {
          name: "invalidCharacters",
          isValid: (formData: IAddCTFormData) => {
            return /^[a-zA-Z0-9 \-_]+$/.test(formData.name);
          },
          message:
            "Invalid characters. Only letters, numbers, spaces, dashes, and underscores allowed.",
        },
        {
          name: "unique",
          isValid: (formData: IAddCTFormData) => {
            return (
              existingCTs.find(
                (ct) => ct.name.toLowerCase() === formData.name.toLowerCase()
              ) === undefined
            );
          },
          message: "Name is already used by another certificate template.",
        },
        {
          name: "maxLength",
          isValid: (formData: IAddCTFormData) => {
            return formData.name.length <= 255;
          },
          message: "Name is too long. Maximum is 255 characters.",
        },
      ],
    },
    certAuthorityId: {
      validations: [
        {
          name: "required",
          isValid: (formData: IAddCTFormData) => {
            return formData.certAuthorityId !== null;
          },
          // no error message specified
        },
      ],
    },
    subjectName: {
      validations: [
        {
          name: "required",
          isValid: (formData: IAddCTFormData) => {
            return formData.subjectName.length > 0;
          },
        },
        // accept any value, let the serve handle any errors
      ],
    },
  };
  return FORM_VALIDATIONS;
};

const getErrorMessage = (
  formData: IAddCTFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

export const validateFormData = (
  formData: IAddCTFormData,
  validationConfig: IFormValidations
): IAddCTFormValidation => {
  const formValidation: IAddCTFormValidation = {
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
