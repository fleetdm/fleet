// @ts-ignore
import validateQuery from "components/forms/validators/validate_query";

import { IFleetMaintainedAppFormData } from "./FleetAppDetailsForm";

type IMessageFunc = (formData: IFleetMaintainedAppFormData) => string;
type IValidationMessage = string | IMessageFunc;

interface IValidation {
  name: string;
  isValid: (formData: IFleetMaintainedAppFormData) => boolean;
  message?: IValidationMessage;
}

const FORM_VALIDATION_CONFIG: Record<
  "preInstallQuery",
  { validations: IValidation[] }
> = {
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
};

const generateFormValidation = (formData: IFleetMaintainedAppFormData) => {
  const formValidation: IFormValidation = {
    isValid: true,
    software: {
      isValid: false,
    },
  };
};
