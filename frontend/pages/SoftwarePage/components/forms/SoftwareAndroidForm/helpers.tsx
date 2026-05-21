// helpers.ts

import {
  ISoftwareAndroidFormData,
  IFormValidation,
} from "./SoftwareAndroidForm";

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
