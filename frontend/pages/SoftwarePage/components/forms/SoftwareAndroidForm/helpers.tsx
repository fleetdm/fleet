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
type IFormValidationKey = keyof Omit<IFormValidation, "isValid">;

const FORM_VALIDATION_CONFIG: Record<
  IFormValidationKey,
  { validations: IValidation[] }
> = {
  customTarget: {
    validations: [
      {
        name: "requiredLabelTargets",
        isValid: (formData) => {
          if (formData.targetType === "All hosts") return true;
          // there must be at least one label target selected
          return (
            Object.keys(formData.labelTargets).find(
              (key) => formData.labelTargets[key]
            ) !== undefined
          );
        },
      },
    ],
  },
};

const generateFormValidation = (formData: ISoftwareAndroidFormData) => {
  const formValidation: IFormValidation = {
    isValid: true,
  };

  Object.keys(FORM_VALIDATION_CONFIG).forEach((key) => {
    const objKey = key as IFormValidationKey;
    const failedValidation = FORM_VALIDATION_CONFIG[objKey].validations.find(
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
      };
    }
  });

  return formValidation;
};

export default generateFormValidation;
