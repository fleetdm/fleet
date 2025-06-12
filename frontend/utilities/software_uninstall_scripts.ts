import { getExtensionFromFileName } from "./file/fileUtils";

// @ts-ignore
import uninstallPkg from "../../pkg/file/scripts/uninstall_pkg.sh";
// @ts-ignore
import uninstallMsi from "../../pkg/file/scripts/uninstall_msi.ps1";
// @ts-ignore
import uninstallDeb from "../../pkg/file/scripts/uninstall_deb.sh";
// @ts-ignore
import uninstallRPM from "../../pkg/file/scripts/uninstall_rpm.sh";

/*
 * getUninstallScript returns a string with a script to uninstall the
 * provided software.
 * */
const getDefaultUninstallScript = (fileName: string): string => {
  const extension = getExtensionFromFileName(fileName);

  switch (extension) {
    case "pkg":
      return uninstallPkg;
    case "msi":
      return uninstallMsi;
    case "deb":
      return uninstallDeb;
    case "rpm":
      return uninstallRPM;
    case "exe":
      return "";
    case "tar.gz":
      return "";
    default:
      throw new Error(`unsupported file extension: ${extension}`);
  }
};

export default getDefaultUninstallScript;
