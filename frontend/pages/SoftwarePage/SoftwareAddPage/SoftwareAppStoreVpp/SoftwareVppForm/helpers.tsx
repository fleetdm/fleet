import { IVppApp } from "services/entities/mdm_apple";
import { ISoftwareVppFormData, IFormValidation } from "./SoftwareVppForm";

interface IValidation {
  name: string;
  isValid: (formData: ISoftwareVppFormData) => boolean;
  message?: IValidationMessage;
}

type IMessageFunc = (formData: ISoftwareVppFormData) => string;
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

export const getUniqueAppId = (app: IVppApp) =>
  `${app.app_store_id}_${app.platform}`;

export const generateFormValidation = (formData: ISoftwareVppFormData) => {
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
