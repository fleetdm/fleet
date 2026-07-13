import {
  CUSTOM_HOST_VITAL_CRITERION,
  LabelHostVitalsCriterion,
} from "interfaces/label";

import { INewLabelFormData } from "./NewLabelPage";

// The criteria dropdown needs a single string per option, but custom host
// vitals all share the `custom_host_vital` criterion value, so the definition
// id is encoded into the option value and decoded on selection.
export const buildCriterionOptionValue = (customHostVitalId: number) =>
  `${CUSTOM_HOST_VITAL_CRITERION}:${customHostVitalId}`;

export const parseCriterionOptionValue = (
  optionValue: string
): { vital: LabelHostVitalsCriterion; customHostVitalId?: number } => {
  if (optionValue.startsWith(`${CUSTOM_HOST_VITAL_CRITERION}:`)) {
    // Guard against a malformed/missing id: an unparseable value would become
    // NaN, later pass `!= null`, and serialize to `null` in the request body.
    const parsedId = Number(optionValue.split(":")[1]);
    return {
      vital: CUSTOM_HOST_VITAL_CRITERION,
      customHostVitalId: Number.isFinite(parsedId) ? parsedId : undefined,
    };
  }
  return { vital: optionValue as LabelHostVitalsCriterion };
};

export const getVitalValuePlaceholder = (vital: LabelHostVitalsCriterion) => {
  if (vital === "end_user_idp_group") {
    return "IT admins";
  }
  if (vital === "end_user_idp_department") {
    return "Engineering";
  }
  return "Value";
};

export const getCriterionHelpText = (vital: LabelHostVitalsCriterion) => {
  if (vital === "end_user_idp_group") {
    return "Label criteria is based on the end user's IdP group.";
  }
  if (vital === "end_user_idp_department") {
    return "Label criteria is based on the end user's IdP department.";
  }
  return "Label criteria is based on the selected custom host vital.";
};

export interface INewLabelFormValidation {
  isValid: boolean;
  name?: { isValid: boolean; message?: string };
  description?: { isValid: boolean; message?: string };
  labelQuery?: { isValid: boolean; message?: string };
  criteria?: { isValid: boolean; message?: string };
}

type IMessageFunc = (formData: INewLabelFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Pick<
  INewLabelFormData,
  "name" | "description" | "labelQuery" | "vitalValue"
>;

interface IValidation {
  name: string;
  isValid: (
    formData: INewLabelFormData,
    validations?: INewLabelFormValidation
  ) => boolean;
  message?: IValidationMessage;
}

type IFormValidations = Record<
  IFormValidationKey,
  { validations: IValidation[] }
>;

const FORM_VALIDATIONS: IFormValidations = {
  name: {
    validations: [
      {
        name: "required",
        isValid: (formData) => formData.name.trim().length > 0,
        message: "Label name must be present",
      },
    ],
  },
  description: {
    validations: [],
  },
  labelQuery: {
    validations: [
      {
        name: "requiredForDynamic",
        isValid: (formData) => {
          if (formData.type !== "dynamic") {
            return true;
          }
          return formData.labelQuery.trim().length > 0;
        },
        message: "Query text must be present",
      },
    ],
  },
  vitalValue: {
    validations: [
      {
        name: "requiredForHostVitals",
        isValid: (formData) => {
          if (formData.type !== "host_vitals") {
            return true;
          }
          return formData.vitalValue.trim().length > 0;
        },
        message: "Label criteria must be completed",
      },
      {
        // A custom-vital criterion is incomplete without a selected definition id.
        name: "customVitalRequiresId",
        isValid: (formData) => {
          if (
            formData.type !== "host_vitals" ||
            formData.vital !== CUSTOM_HOST_VITAL_CRITERION
          ) {
            return true;
          }
          return formData.customHostVitalId != null;
        },
        message: "Label criteria must be completed",
      },
    ],
  },
};

const getErrorMessage = (
  formData: INewLabelFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

export const validateNewLabelFormData = (
  formData: INewLabelFormData
): INewLabelFormValidation => {
  const formValidation: INewLabelFormValidation = { isValid: true };

  (Object.keys(FORM_VALIDATIONS) as IFormValidationKey[]).forEach((objKey) => {
    const failedValidation = FORM_VALIDATIONS[objKey].validations.find(
      (validation) => !validation.isValid(formData, formValidation)
    );

    if (!failedValidation) {
      switch (objKey) {
        case "name":
          formValidation.name = { isValid: true };
          break;
        case "description":
          formValidation.description = { isValid: true };
          break;
        case "labelQuery":
          formValidation.labelQuery = { isValid: true };
          break;
        case "vitalValue":
          formValidation.criteria = { isValid: true };
          break;
        default: {
          break;
        }
      }
    } else {
      formValidation.isValid = false;
      const message = getErrorMessage(formData, failedValidation.message);
      switch (objKey) {
        case "name":
          formValidation.name = { isValid: false, message };
          break;
        case "description":
          formValidation.description = { isValid: false, message };
          break;
        case "labelQuery":
          formValidation.labelQuery = { isValid: false, message };
          break;
        case "vitalValue":
          formValidation.criteria = { isValid: false, message };
          break;
        default: {
          break;
        }
      }
    }
  });

  return formValidation;
};
