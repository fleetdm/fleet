type IPlatformDisplayName = "macOS" | "Windows" | "Linux";

const getFileExtension = (file: File) => {
  const nameParts = file.name.split(".");
  return nameParts.slice(-1)[0];
};

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
};

/**
 * This gets the platform display name from the file.
 */
export const getPlatformDisplayName = (file: File) => {
  const fileExt = getFileExtension(file);
  return FILE_EXTENSIONS_TO_PLATFORM_DISPLAY_NAME[fileExt];
};

/**
 * This gets the file details from the file.
 */
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
