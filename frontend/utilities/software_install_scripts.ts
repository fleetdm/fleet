// @ts-ignore
import installPkg from "../../pkg/file/scripts/install_pkg.sh";
// @ts-ignore
import installMsi from "../../pkg/file/scripts/install_msi.ps1";
// @ts-ignore
import installExe from "../../pkg/file/scripts/install_exe.ps1";
// @ts-ignore
import installDeb from "../../pkg/file/scripts/install_deb.sh";

/*
 * getInstallScript returns a string with a script to install the
 * provided software.
 * */
const getInstallScript = (fileName: string): string => {
  const extension = fileName.split(".").pop();
  switch (extension) {
    case "pkg":
      return installPkg;
    case "msi":
      return installMsi;
    case "deb":
      return installDeb;
    case "exe":
      return installExe;
    default:
      throw new Error(`unsupported file extension: ${extension}`);
  }
};

export default getInstallScript;
