type IPlatformDisplayName = "macOS" | "Windows" | "linux";

const getFileExtension = (file: File) => {
  const nameParts = file.name.split(".");
  return nameParts.slice(-1)[0];
};

export const FILE_EXTENSIONS_TO_PLATFORM_DISPLAY_NAME: Record<
  IPlatformDisplayName,
  string[]
> = {
  macOS: ["json", "pkg", "mobileconfig"],
  Windows: ["exe", "msi", "xml"],
  linux: ["dev"],
};

/**
 * This gets the platform display name from the file.
 */
export const getPlatformDisplayName = (file: File) => {
  const fileExt = getFileExtension(file);
  const keys = Object.keys(
    FILE_EXTENSIONS_TO_PLATFORM_DISPLAY_NAME
  ) as IPlatformDisplayName[];

  const platformKey = keys.find((key) => {
    const foundExt = FILE_EXTENSIONS_TO_PLATFORM_DISPLAY_NAME[key].find(
      (ext) => ext === fileExt
    );
    return foundExt !== undefined;
  });

  return platformKey ?? "";
};
