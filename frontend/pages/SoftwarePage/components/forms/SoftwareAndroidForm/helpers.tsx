// helpers.ts

import {
  ISoftwareAndroidFormData,
  IFormValidation,
} from "./SoftwareAndroidForm";

const generateFormValidation = (formData: ISoftwareAndroidFormData) => {
  const formValidation: IFormValidation = {
    isValid: true,
  };

  if (!formData.applicationID) {
    formValidation.isValid = false;
  }

  // A custom target needs at least one label selected
  if (
    formData.targetType === "Custom" &&
    !Object.values(formData.labelTargets).some(Boolean)
  ) {
    formValidation.isValid = false;
  }

  return formValidation;
};

export default generateFormValidation;
