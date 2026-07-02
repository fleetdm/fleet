export interface ICustomHostVitalFormValidation {
  isValid: boolean;
  name?: { isValid: boolean; message?: string };
}

interface INameFormData {
  name: string;
}

type IMessageFunc = (formData: INameFormData) => string;
type IValidationMessage = string | IMessageFunc;

interface IValidation {
  name: string;
  isValid: (formData: INameFormData) => boolean;
  message?: IValidationMessage;
}

const NAME_VALIDATIONS: IValidation[] = [
  {
    name: "required",
    isValid: (formData) => formData.name.trim().length > 0,
    message: "Name is required",
  },
  {
    name: "notTooLong",
    isValid: (formData) => formData.name.trim().length <= 255,
    message: "Name may not exceed 255 characters",
  },
];

const getErrorMessage = (
  formData: INameFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

export const validateFormData = (
  formData: INameFormData,
  isSaving = false
): ICustomHostVitalFormValidation => {
  const formValidation: ICustomHostVitalFormValidation = { isValid: true };

  const failedValidation = NAME_VALIDATIONS.find((validation) => {
    if (!isSaving && validation.name === "required") {
      return false; // Skip required check until the user attempts to save.
    }
    return !validation.isValid(formData);
  });

  if (!failedValidation) {
    formValidation.name = { isValid: true };
  } else {
    formValidation.isValid = false;
    formValidation.name = {
      isValid: false,
      message: getErrorMessage(formData, failedValidation.message),
    };
  }

  return formValidation;
};
