import { getPlatformDisplayName } from "utilities/file/fileUtils";

import { IAddSoftwareFormData } from "./AddSoftwareForm";

const FORM_VALIDATION_CONFIG = {
  software: {
    isValid: (formData: IAddSoftwareFormData) => formData.software !== null,
  },
  preInstallCondition: {
    isValid: (formData: IAddSoftwareFormData) => formData.software !== null,
  },
  postInstallScript: {
    isValid: (formData: IAddSoftwareFormData) => formData.software !== null,
  },
};

// export const validateForm = () => {};

export const shouldDisableFormSubmit = (
  formData: IAddSoftwareFormData,
  showPreInstallCondition: boolean,
  showPostInstallScript: boolean
) => {
  const preInstallEnabledWithNoValue =
    showPreInstallCondition &&
    (formData.preInstallCondition === undefined ||
      formData.preInstallCondition === "");
  const postInstallEnabledWithNoValue =
    showPostInstallScript &&
    (formData.postInstallScript === undefined ||
      formData.postInstallScript === "");

  return (
    formData.software === null ||
    preInstallEnabledWithNoValue ||
    postInstallEnabledWithNoValue
  );
};

export const getFileDetails = (file: File) => {
  return {
    name: file.name,
    platform: getPlatformDisplayName(file),
  };
};

export const getInstallScript = (file: File) => {
  // TODO: get this dynamically
  return `sudo installer -pkg ${file.name} -target /`;
};
