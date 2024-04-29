import { getPlatformDisplayName } from "utilities/file/fileUtils";

import { IAddSoftwareFormData } from "./AddSoftwareForm";

export const getFormSubmitDisabled = (
  formData: IAddSoftwareFormData,
  showPreInstallCondition: boolean,
  showPostInstallScript: boolean
) => {
  const preInstallEnabledWithNoValue =
    showPreInstallCondition && formData.preInstallCondition === "";
  const postInstallEnabledWithNoValue =
    showPostInstallScript && formData.postInstallScript === "";

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
