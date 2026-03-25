import { getExtensionFromFileName } from "./file/fileUtils";

// @ts-ignore
import installPkg from "../../pkg/file/scripts/install_pkg.sh";
// @ts-ignore
import installPkgFleetd from "../../pkg/file/scripts/install_pkg_fleetd.sh";
// @ts-ignore
import installMsi from "../../pkg/file/scripts/install_msi.ps1";
// @ts-ignore
import installDeb from "../../pkg/file/scripts/install_deb.sh";
// @ts-ignore
import installRPM from "../../pkg/file/scripts/install_rpm.sh";

/*
 * getInstallScript returns a string with a script to install the
 * provided software.
 * */
const getDefaultInstallScript = (fileName: string): string => {
  const extension = getExtensionFromFileName(fileName);

  switch (extension) {
    case "pkg":
      if (fileName.toLowerCase().includes("fleet-osquery")) {
        return installPkgFleetd;
      }
      return installPkg;
    case "msi":
      return installMsi;
    case "deb":
      return installDeb;
    case "rpm":
      return installRPM;
    case "exe":
    case "zip":
    case "tar.gz":
    case "sh":
    case "ps1":
    case "ipa":
      return "";
    default:
      throw new Error(`unsupported file extension: ${extension}`);
  }
};

export default getDefaultInstallScript;
