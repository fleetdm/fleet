import { IDropdownOption } from "interfaces/dropdownOption";
import { ISoftwarePackage } from "interfaces/software";

// @ts-ignore
import validateQuery from "components/forms/validators/validate_query";

import { IPackageFormData, IPackageFormValidation } from "./PackageForm";

type IMessageFunc = (formData: IPackageFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<IPackageFormValidation, "isValid">;

interface IValidation {
  name: string;
  isValid: (formData: IPackageFormData) => boolean;
  message?: IValidationMessage;
}

/** configuration defines validations for each field in the form. It defines rules
 *  to determine if a field is valid, and rules for generating an error message.
 */
const FORM_VALIDATION_CONFIG: Record<
  IFormValidationKey,
  { validations: IValidation[] }
> = {
  software: {
    validations: [
      {
        name: "required",
        isValid: (formData) => formData.software !== null,
      },
    ],
  },
  preInstallQuery: {
    validations: [
      {
        name: "invalidQuery",
        isValid: (formData) => {
          const query = formData.preInstallQuery;
          return (
            query === undefined || query === "" || validateQuery(query).valid
          );
        },
        message: (formData) => validateQuery(formData.preInstallQuery).error,
      },
    ],
  },
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

const getErrorMessage = (
  formData: IPackageFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

export const generateFormValidation = (formData: IPackageFormData) => {
  const formValidation: IPackageFormValidation = {
    isValid: true,
    software: {
      isValid: false,
    },
  };

  Object.keys(FORM_VALIDATION_CONFIG).forEach((key) => {
    const objKey = key as keyof typeof FORM_VALIDATION_CONFIG;
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
        message: getErrorMessage(formData, failedValidation.message),
      };
    }
  });

  return formValidation;
};

export default generateFormValidation;

export const CUSTOM_TARGET_OPTIONS: IDropdownOption[] = [
  {
    value: "labelsIncludeAny",
    label: "Include any",
    disabled: false,
  },
  {
    value: "labelsExcludeAny",
    label: "Exclude any",
    disabled: false,
  },
];

export const getTargetType = (softwarePackage: ISoftwarePackage) => {
  if (!softwarePackage) return "All hosts";

  return !softwarePackage.labels_include_any &&
    !softwarePackage.labels_exclude_any
    ? "All hosts"
    : "Custom";
};

export const getCustomTarget = (softwarePackage: ISoftwarePackage) => {
  if (!softwarePackage) return "labelsIncludeAny";

  return softwarePackage.labels_include_any
    ? "labelsIncludeAny"
    : "labelsExcludeAny";
};

export const generateSelectedLabels = (softwarePackage: ISoftwarePackage) => {
  if (
    !softwarePackage ||
    (!softwarePackage.labels_include_any && !softwarePackage.labels_exclude_any)
  ) {
    return {};
  }

  const customTypeKey = softwarePackage.labels_include_any
    ? "labels_include_any"
    : "labels_exclude_any";

  return (
    softwarePackage[customTypeKey]?.reduce<Record<string, boolean>>(
      (acc, label) => {
        acc[label.name] = true;
        return acc;
      },
      {}
    ) ?? {}
  );
};
