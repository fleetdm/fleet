import React from "react";
import { getErrorReason } from "interfaces/errors";
import { IVppApp } from "services/entities/mdm_apple";
import { IVPPAppFormData, IFormValidation } from "./SoftwareVppForm";

const ADD_SOFTWARE_ERROR_PREFIX = "Couldn’t add software.";
const DEFAULT_ERROR_MESSAGE = `${ADD_SOFTWARE_ERROR_PREFIX} Please try again.`;

interface IValidation {
  name: string;
  isValid: (formData: IVPPAppFormData) => boolean;
  message?: IValidationMessage;
}

type IMessageFunc = (formData: IVPPAppFormData) => string;
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

const generateAlreadyAvailableMessage = (msg: string) => {
  // This regex matches the API message where the title already has a software package (non-VPP) available for install.
  const regex = new RegExp(
    `${ADD_SOFTWARE_ERROR_PREFIX} (.+) already.+on the (.+) team.`
  );

  const match = msg.match(regex);
  if (!match) {
    if (msg.includes("VPPApp")) {
      // This is the case where someone already added this VPP app. This should almost never happen
      // because we omit apps that are already available from the list in the UI, but just in case of
      // shenanigans with concurrent requests or something, we'll handle it with a generic message.
      // The list should clear itself up on the next page load.
      return `${ADD_SOFTWARE_ERROR_PREFIX} The software is already available to install on this team.`;
    }
    return DEFAULT_ERROR_MESSAGE;
  }

  return (
    <>
      {ADD_SOFTWARE_ERROR_PREFIX} <b>{match[1]}</b> already has software
      available for install on the <b>{match[2]}</b> team.{" "}
    </>
  );
};

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (e: unknown) => {
  let reason = getErrorReason(e);
  // software is already available for install
  if (reason.toLowerCase().includes("already")) {
    return generateAlreadyAvailableMessage(reason);
  }
  if (reason && !reason.endsWith(".")) {
    reason += ".";
  }
  return reason || DEFAULT_ERROR_MESSAGE;
};

export const getUniqueAppId = (app: IVppApp) =>
  `${app.app_store_id}_${app.platform}`;

// eslint-disable-next-line import/prefer-default-export
export const generateFormValidation = (formData: IVPPAppFormData) => {
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
