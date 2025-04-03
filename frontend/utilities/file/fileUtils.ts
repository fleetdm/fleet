import { PackageType } from "interfaces/package_type";

type IPlatformDisplayName = "macOS" | "Windows" | "Linux";

export const FILE_EXTENSIONS_TO_PLATFORM_DISPLAY_NAME: Record<
  string,
  IPlatformDisplayName
> = {
  json: "macOS",
  pkg: "macOS",
  mobileconfig: "macOS",
  exe: "Windows",
  msi: "Windows",
  xml: "Windows",
  deb: "Linux",
  rpm: "Linux",
  "tar.gz": "Linux",
};

/** Extract the extension, considering compound extensions like .tar.gz */
export const getExtensionFromFileName = (fileName: string) => {
  const parts = fileName.split(".");

  if (parts.length <= 1) {
    // No period in the filename, hence no extension
    return undefined;
  }

  const extension =
    parts.length > 1 && parts[parts.length - 2] === "tar"
      ? "tar.gz"
      : parts.pop();
  return extension as PackageType;
};

/** This gets the platform display name from the file. */
export const getPlatformDisplayName = (file: File) => {
  const fileExt = getExtensionFromFileName(file.name);
  if (!fileExt) {
    return undefined;
  }
  return FILE_EXTENSIONS_TO_PLATFORM_DISPLAY_NAME[fileExt];
};

/** This gets the file details from the file. */
export const getFileDetails = (file: File) => {
  return {
    name: file.name,
    platform: getPlatformDisplayName(file),
  };
};

export interface IFileDetails {
  name: string;
  platform?: string;
}
