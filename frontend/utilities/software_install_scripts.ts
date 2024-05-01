import { ISoftwareInstallerType } from "interfaces/software_installers";

// @ts-ignore
import installPkg from "../../pkg/file/scripts/install_pkg.sh";
// @ts-ignore
import installMsi from "../../pkg/file/scripts/install_msi.ps1";
// @ts-ignore
import installExe from "../../pkg/file/scripts/install_exe.ps1";
// @ts-ignore
import installDeb from "../../pkg/file/scripts/install_deb.sh";

const replaceVariables = (rawScript: string, installerPath: string): string => {
  return rawScript.replace("$INSTALLER_PATH", installerPath);
};

/*
 * getInstallScript returns a string with a script to install the
 * provided software.
 *
 * Note that we don't do any sanitization of the arguments here,
 * delegating that to the caller which should have the right context
 * about what should be escaped.
 * */
const getInstallScript = (
  filetype: ISoftwareInstallerType,
  path: string
): string => {
  let rawScript: string;
  switch (filetype) {
    case "pkg":
      rawScript = installPkg;
      break;
    case "msi":
      rawScript = installMsi;
      break;
    case "deb":
      rawScript = installDeb;
      break;
    case "exe":
      rawScript = installExe;
      break;
    default:
      // this should never happen as this function is type-guarded
      throw new Error(`unsupported file type: ${filetype}`);
  }

  return replaceVariables(rawScript, path);
};

export default getInstallScript;
