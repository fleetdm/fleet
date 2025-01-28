// @ts-ignore
import uninstallPkg from "../../pkg/file/scripts/uninstall_pkg.sh";
// @ts-ignore
import uninstallMsi from "../../pkg/file/scripts/uninstall_msi.ps1";
// @ts-ignore
import uninstallExe from "../../pkg/file/scripts/uninstall_exe.ps1";
// @ts-ignore
import uninstallDeb from "../../pkg/file/scripts/uninstall_deb.sh";
// @ts-ignore
import uninstallRPM from "../../pkg/file/scripts/uninstall_rpm.sh";

/*
 * getUninstallScript returns a string with a script to uninstall the
 * provided software.
 * */
const getDefaultUninstallScript = (fileName: string): string => {
  const extension = fileName.split(".").pop();
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
      return uninstallExe;
    default:
      throw new Error(`unsupported file extension: ${extension}`);
  }
};

export default getDefaultUninstallScript;
