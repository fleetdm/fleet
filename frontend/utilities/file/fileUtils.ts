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

/** Currently only using tar.gz, but keeping the others for future use
 *  and to avoid breaking changes. */
const compoundExtensions = ["tar.gz", "tar.xz", "tar.bz2", "tar.zst"];

/**  Currently only using tgz, but keeping the others for future use
 * and to avoid breaking changes. */
const extensionAliases: Record<string, string> = {
  tgz: "tar.gz",
  tbz2: "tar.bz2",
  tzst: "tar.zst",
  txz: "tar.xz",
};

/** Extract the extension, considering compound extensions like .tar.gz;
 * Aliases like .tgz will be converted to compound extensions like .tar.gz
 */
export const getExtensionFromFileName = (fileName: string) => {
  const lower = fileName.toLowerCase();
  const parts = lower.split(".");

  // Find compound extension
  const compound = compoundExtensions.find((ext) => {
    const extParts = ext.split(".");
    return parts.slice(-extParts.length).join(".") === ext;
  });

  // Choose extension: compound or simple
  let ext: string | undefined;
  if (compound) {
    ext = compound;
  } else if (parts.length > 1) {
    ext = parts.pop();
  }

  // Map aliases if needed
  if (ext && extensionAliases[ext]) {
    ext = extensionAliases[ext];
  }

  return ext as PackageType | undefined;
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
