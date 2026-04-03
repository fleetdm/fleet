import { IValidationConfig } from "hooks/useFormValidation";
import { IAddSecretFormData } from "./AddSecretModal";

const ADD_SECRET_VALIDATIONS: IValidationConfig<IAddSecretFormData> = {
  name: {
    validations: [
      {
        name: "required",
        isValid: (formData) => {
          return formData.name.length > 0;
        },
        message: "Name is required",
      },
      {
        name: "validName",
        isValid: (formData) => {
          if (formData.name.length === 0) {
            return true; // Skip this validation if name is empty
          }
          return !!formData.name.match(/^[a-zA-Z0-9_]+$/);
        },
        message:
          "Name may only include uppercase letters, numbers, and underscores",
      },
      {
        name: "notTooLong",
        isValid: (formData) => {
          return formData.name.length <= 255;
        },
        message: "Name may not exceed 255 characters",
      },
      {
        name: "doesNotIncludePrefix",
        isValid: (formData) => {
          return !formData.name.match(/^FLEET_SECRET_/);
        },
        message: "Name should not include variable prefix",
      },
    ],
  },
  value: {
    validations: [
      {
        name: "required",
        isValid: (formData) => {
          return formData.value.length > 0;
        },
        message: "Value is required",
      },
    ],
  },
};

export default ADD_SECRET_VALIDATIONS;
