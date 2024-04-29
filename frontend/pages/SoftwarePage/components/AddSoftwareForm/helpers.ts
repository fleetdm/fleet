import { getPlatformDisplayName } from "utilities/file/fileUtils";

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
