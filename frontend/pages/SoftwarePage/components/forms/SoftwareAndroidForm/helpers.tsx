// helpers.ts

import { IFormErrors } from "hooks/useFormValidation";

import { ISoftwareAndroidFormData } from "./SoftwareAndroidForm";

const validate = (formData: ISoftwareAndroidFormData): IFormErrors => {
  const errors: IFormErrors = {};

  if (!formData.applicationID.trim()) {
    errors.applicationID = "Enter an application ID";
  }

  if (
    formData.targetType === "Custom" &&
    !Object.values(formData.labelTargets).some(Boolean)
  ) {
    errors.labelTargets = "Select at least one label";
  }

  return errors;
};

export default validate;
