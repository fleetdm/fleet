import { ISoftwareInstallerType } from "interfaces/software";

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
 *
 * Note that we don't do any sanitization of the arguments here,
 * delegating that to the caller which should have the right context
 * about what should be escaped.
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
      // this should never happen as this function is type-guarded
      throw new Error(`unsupported file type: ${filetype}`);
  }
};

export default getInstallScript;
