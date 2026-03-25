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

// isFleetdPkg checks the binary contents of a .pkg file for the
// com.fleetdm.orbit identifier (embedded in the Distribution XML).
const isFleetdPkg = async (file: File): Promise<boolean> => {
  try {
    const buffer = await file.arrayBuffer();
    const text = new TextDecoder("utf-8", { fatal: false }).decode(buffer);
    return text.includes("com.fleetdm.orbit");
  } catch {
    return false;
  }
};

/*
 * getDefaultInstallScript returns a string with a default script to install
 * the provided software.
 * */
const getDefaultInstallScript = async (file: File): Promise<string> => {
  const extension = getExtensionFromFileName(file.name);

  switch (extension) {
    case "pkg":
      if (await isFleetdPkg(file)) {
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
