import { useContext, useMemo } from "react";
import { AppContext } from "context/app";
import { isAndroid } from "interfaces/platform";
import {
  ISoftwareTitleDetails,
  ISoftwarePackage,
  IAppStoreApp,
  isSoftwarePackage,
  isIpadOrIphoneSoftwareSource,
  InstallerType,
} from "interfaces/software";
import {
  getInstallerCardInfo,
  InstallerCardInfo,
} from "pages/SoftwarePage/SoftwareTitleDetailsPage/helpers";
import { isAndroidWebApp } from "pages/SoftwarePage/helpers";
import { compareVersions } from "utilities/helpers";

export interface SoftwareInstallerMeta {
  installerType: InstallerType;
  /** Includes both Google Play Store apps and Google Play Store web apps */
  isAndroidPlayStoreApp: boolean;
  /** Only includes Google Play Store web apps */
  isAndroidPlayStoreWebApp: boolean;
  isFleetMaintainedApp: boolean;
  isLatestFmaVersion: boolean;
  isCustomPackage: boolean;
  isIosOrIpadosApp: boolean;
  sha256?: string;
  androidPlayStoreId?: string;
  automaticInstallPolicies:
    | ISoftwarePackage["automatic_install_policies"]
    | IAppStoreApp["automatic_install_policies"];
  gitOpsModeEnabled: boolean;
  repoURL?: string;
  canManageSoftware: boolean;
  /** Raw ISoftwarePackage | IAppStoreApp data */
  softwareInstaller: ISoftwarePackage | IAppStoreApp;
}

export interface UseSoftwareInstallerResult {
  cardInfo: InstallerCardInfo;
  meta: SoftwareInstallerMeta;
}

/** This is used to extract software installer data
 * (FMA, VPP, Google Playstore Apps, custom packages)
 * from ISoftwareTitleDetails to be used in the UI  */
export const useSoftwareInstaller = (
  softwareTitle: ISoftwareTitleDetails
): UseSoftwareInstallerResult | undefined => {
  const appContext = useContext(AppContext);

  return useMemo(() => {
    if (!softwareTitle.software_package && !softwareTitle.app_store_app) {
      return undefined;
    }

    const cardInfo = getInstallerCardInfo(softwareTitle);
    const { softwareInstaller, source } = cardInfo;

    const isIosOrIpadosApp = isIpadOrIphoneSoftwareSource(source);

    const installerType: InstallerType = isSoftwarePackage(softwareInstaller)
      ? "package"
      : "app-store";

    const isAndroidPlayStoreApp =
      "platform" in softwareInstaller && isAndroid(softwareInstaller.platform);

    const isAndroidPlayStoreWebApp =
      isAndroidPlayStoreApp && "app_store_id" in softwareInstaller
        ? isAndroidWebApp(softwareInstaller.app_store_id)
        : false;

    const isFleetMaintainedApp =
      "fleet_maintained_app_id" in softwareInstaller &&
      !!softwareInstaller.fleet_maintained_app_id;

    const isLatestFmaVersion =
      isFleetMaintainedApp &&
      "fleet_maintained_versions" in softwareInstaller &&
      !!softwareInstaller.fleet_maintained_versions &&
      softwareInstaller.fleet_maintained_versions.every(
        (fma) =>
          // Verify that the installer version is not older than any known
          // Fleet‑maintained version by requiring compareVersions to return
          // 0 (equal) or 1 (greater) for every entry.
          compareVersions(softwareInstaller.version ?? "", fma.version ?? "") >=
          0
      );

    const fmaVersions =
      isFleetMaintainedApp && "fleet_maintained_versions" in softwareInstaller
        ? softwareInstaller.fleet_maintained_versions
        : [];

    const isCustomPackage =
      installerType === "package" && !isFleetMaintainedApp;

    const sha256 =
      ("hash_sha256" in softwareInstaller && softwareInstaller.hash_sha256) ||
      undefined;

    const androidPlayStoreId =
      isAndroidPlayStoreApp && "app_store_id" in softwareInstaller
        ? softwareInstaller?.app_store_id
        : undefined;

    const {
      automatic_install_policies: automaticInstallPolicies,
    } = softwareInstaller;

    const {
      isGlobalAdmin,
      isGlobalMaintainer,
      isTeamAdmin,
      isTeamMaintainer,
      config,
    } = appContext;

    const {
      gitops_mode_enabled: configGitOpsModeEnabled,
      repository_url: repoURL,
    } = config?.gitops || {};

    const gitOpsModeEnabled = !!configGitOpsModeEnabled;

    const canManageSoftware = !!(
      isGlobalAdmin ||
      isGlobalMaintainer ||
      isTeamAdmin ||
      isTeamMaintainer
    );

    return {
      cardInfo,
      meta: {
        installerType,
        isAndroidPlayStoreApp,
        isAndroidPlayStoreWebApp,
        isFleetMaintainedApp,
        isLatestFmaVersion,
        fmaVersions,
        isCustomPackage,
        isIosOrIpadosApp,
        sha256,
        androidPlayStoreId,
        automaticInstallPolicies,
        gitOpsModeEnabled,
        repoURL,
        canManageSoftware,
        softwareInstaller,
      },
    };
  }, [softwareTitle, appContext]);
};
