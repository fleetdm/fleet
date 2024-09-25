import React from "react";

import { isWindowsPackageType, PackageType } from "interfaces/package_type";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

// @ts-ignore
import validateQuery from "components/forms/validators/validate_query";
import CustomLink from "components/CustomLink";

import {
  ICustomPackageAppFormData,
  IFormValidation,
} from "./AddSoftwareCustomPackageForm";

type IMessageFunc = (formData: ICustomPackageAppFormData) => string;
type IValidationMessage = string | IMessageFunc;

interface IValidation {
  name: string;
  isValid: (formData: ICustomPackageAppFormData) => boolean;
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

const getErrorMessage = (
  formData: ICustomPackageAppFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

// eslint-disable-next-line import/prefer-default-export
export const generateFormValidation = (formData: ICustomPackageAppFormData) => {
  const formValidation: IFormValidation = {
    isValid: true,
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

export const getSupportedScriptTypeText = (pkgType: PackageType) => {
  return `Currently, ${
    isWindowsPackageType(pkgType) ? "PowerS" : "s"
  }hell scripts are supported.`;
};

const PKG_TYPE_TO_ID_TEXT = {
  pkg: "package IDs",
  deb: "package name",
  msi: "product code",
  exe: "software name",
} as const;

export const getInstallHelpText = (pkgType: PackageType) => (
  <>
    Use the $INSTALLER_PATH variable to point to the installer.{" "}
    {getSupportedScriptTypeText(pkgType)}{" "}
    <CustomLink
      url={`${LEARN_MORE_ABOUT_BASE_LINK}/install-scripts`}
      text="Learn more about install scripts"
      newTab
    />
  </>
);

export const getPostInstallHelpText = (pkgType: PackageType) => {
  return getSupportedScriptTypeText(pkgType);
};

export const getUninstallHelpText = (pkgType: PackageType) => {
  return (
    <>
      $PACKAGE_ID will be populated with the {PKG_TYPE_TO_ID_TEXT[pkgType]} from
      the .{pkgType} file after the software is added.{" "}
      {getSupportedScriptTypeText(pkgType)}{" "}
      <CustomLink
        url={`${LEARN_MORE_ABOUT_BASE_LINK}/uninstall-scripts`}
        text="Learn more about uninstall scripts"
        newTab
      />
    </>
  );
};
