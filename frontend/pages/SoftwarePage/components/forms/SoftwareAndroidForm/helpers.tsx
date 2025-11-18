// helpers.ts

import {
  ISoftwareAndroidFormData,
  IFormValidation,
} from "./SoftwareAndroidForm";

interface IValidation {
  name: string;
  isValid: (formData: ISoftwareAndroidFormData) => boolean;
  message?: IValidationMessage;
}

type IMessageFunc = (formData: ISoftwareAndroidFormData) => string;
type IValidationMessage = string | IMessageFunc;

const generateFormValidation = (formData: ISoftwareAndroidFormData) => {
  const formValidation: IFormValidation = {
    isValid: true,
  };

  // Single requirement: applicationID must be present
  if (!formData.applicationID) {
    formValidation.isValid = false;
  }

  return formValidation;
};

export default generateFormValidation;
