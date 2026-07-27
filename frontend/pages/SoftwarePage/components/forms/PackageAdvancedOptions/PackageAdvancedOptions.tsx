import React, { useState } from "react";
import { noop } from "lodash";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { getExtensionFromFileName } from "utilities/file/fileUtils";

import {
  isPackageType,
  isWindowsPackageType,
  isFleetMaintainedPackageType,
  isScriptOnlyPackageType,
  PackageType,
} from "interfaces/package_type";

import CustomLink from "components/CustomLink";
import RevealButton from "components/buttons/RevealButton";

import { IPackageFormData } from "../PackageForm/PackageForm";
import AdvancedOptionsFields from "../AdvancedOptionsFields";

const getSupportedScriptTypeText = (pkgType: PackageType) => {
  // .ps1 is a script-only package type, not a "windows package type", but it's
  // still PowerShell.
  const isPowerShell = isWindowsPackageType(pkgType) || pkgType === "ps1";
  return `Currently, ${
    isPowerShell ? "PowerS" : "s"
  }hell scripts are supported.`;
};

const PKG_TYPE_TO_ID_TEXT = {
  pkg: "package IDs",
  deb: "package name",
  rpm: "package name",
  msi: "product code",
  exe: "software name",
  zip: "software name",
  sh: "package name",
  ps1: "package name",
  ipa: "software name",
} as const;

const getInstallScriptTooltip = (pkgType: PackageType) => {
  if (pkgType === "exe" || pkgType === "tar.gz") {
    if (pkgType === "exe") {
      return "Required for .exe packages.";
    }
    return "Required for .tar.gz archives.";
  }
  if (pkgType === "zip" && isWindowsPackageType(pkgType)) {
    return "Required for .zip packages.";
  }
  return undefined;
};

const getInstallHelpText = (pkgType: PackageType) => {
  if (isScriptOnlyPackageType(pkgType)) {
    return "The uploaded script's contents are used as the install script. To change it, upload a new file.";
  }

  if (pkgType === "exe") {
    return (
      <>
        For Windows, Fleet only creates install scripts for .msi packages. Use
        the $INSTALLER_PATH variable to point to the installer.{" "}
        {getSupportedScriptTypeText(pkgType)}{" "}
        <CustomLink
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/exe-install-scripts`}
          text="Learn more"
          newTab
        />
      </>
    );
  }

  if (pkgType === "zip") {
    return (
      <>
        For Windows, Fleet only creates install scripts for .msi packages. Use
        the $INSTALLER_PATH variable to point to the installer.{" "}
        {getSupportedScriptTypeText(pkgType)}{" "}
        <CustomLink
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/exe-install-scripts`}
          text="Learn more"
          newTab
        />
      </>
    );
  }

  return (
    <>
      Use the $INSTALLER_PATH variable to point to the installer.{" "}
      {getSupportedScriptTypeText(pkgType)}{" "}
      <CustomLink
        url={`${LEARN_MORE_ABOUT_BASE_LINK}/install-scripts`}
        text="Learn more about install scripts"
        newTab
      />
    </>
  );
};

const getPostInstallHelpText = (pkgType: PackageType) => {
  return getSupportedScriptTypeText(pkgType);
};

const getUninstallScriptTooltip = (pkgType: PackageType) => {
  if (pkgType === "exe" || pkgType === "tar.gz") {
    if (pkgType === "exe") {
      return "Required for .exe packages.";
    }
    return "Required for .tar.gz archives.";
  }
  if (pkgType === "zip" && isWindowsPackageType(pkgType)) {
    return "Required for .zip packages.";
  }
  return undefined;
};

const getUninstallHelpText = (pkgType: PackageType) => {
  // Script-only packages have no installer metadata, so there's no $PACKAGE_ID
  // to populate; the uninstall script runs as-is.
  if (isScriptOnlyPackageType(pkgType)) {
    return getSupportedScriptTypeText(pkgType);
  }

  // Check for Windows zip files first (before isFleetMaintainedPackageType check)
  if (pkgType === "zip" && isWindowsPackageType(pkgType)) {
    return (
      <>
        For Windows, Fleet only creates uninstall scripts for .msi packages.
        $PACKAGE_ID will be populated with the software name from the .zip file
        after it&apos;s added. {getSupportedScriptTypeText(pkgType)}{" "}
        <CustomLink
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/exe-install-scripts`}
          text="Learn more"
          newTab
        />
      </>
    );
  }

  if (isFleetMaintainedPackageType(pkgType)) {
    return "Currently, only shell scripts are supported.";
  }

  if (pkgType === "exe") {
    return (
      <>
        For Windows, Fleet only creates uninstall scripts for .msi packages.
        $PACKAGE_ID will be populated with the software name from the .exe file
        after it&apos;s added. {getSupportedScriptTypeText(pkgType)}{" "}
        <CustomLink
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/exe-install-scripts`}
          text="Learn more"
          newTab
        />
      </>
    );
  }

  if (pkgType === "tar.gz") {
    return (
      <>
        Currently, only shell scripts are supported.{" "}
        <CustomLink
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/uninstall-scripts`}
          text="Learn more about uninstall scripts"
          newTab
        />
      </>
    );
  }

  if (pkgType === "msi") {
    return (
      <>
        $UPGRADE_CODE will be populated with the .msi&apos;s upgrade code if
        available, and $PACKAGE_ID will be populated with its product code,
        after the software is added. {getSupportedScriptTypeText(pkgType)}{" "}
        <CustomLink
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/uninstall-scripts`}
          text="Learn more about uninstall scripts"
          newTab
        />
      </>
    );
  }

  return (
    <>
      $PACKAGE_ID will be populated with the {PKG_TYPE_TO_ID_TEXT[pkgType]} from
      the .{pkgType} file after the software is added.{" "}
      {getSupportedScriptTypeText(pkgType)}{" "}
      <CustomLink
        url={`${LEARN_MORE_ABOUT_BASE_LINK}/uninstall-scripts`}
        text="Learn more about uninstall scripts"
        newTab
      />
    </>
  );
};

const baseClass = "package-advanced-options";

interface IPackageAdvancedOptionsProps {
  errors: { preInstallQuery?: string; postInstallScript?: string };
  selectedPackage: IPackageFormData["software"];
  preInstallQuery?: string;
  installScript: string;
  postInstallScript?: string;
  uninstallScript?: string;
  showSchemaButton?: boolean;
  onClickShowSchema?: () => void;
  onChangePreInstallQuery: (value?: string) => void;
  onChangeInstallScript: (value: string) => void;
  onChangePostInstallScript: (value?: string) => void;
  onChangeUninstallScript: (value?: string) => void;
  /** Currently for editing FMA only, users cannot edit */
  gitopsCompatible?: boolean;
  gitOpsModeEnabled?: boolean;
}

const PackageAdvancedOptions = ({
  showSchemaButton = false,
  errors,
  selectedPackage,
  preInstallQuery,
  installScript,
  postInstallScript,
  uninstallScript,
  onClickShowSchema = noop,
  onChangePreInstallQuery,
  onChangeInstallScript,
  onChangePostInstallScript,
  onChangeUninstallScript,
  gitopsCompatible = false,
  gitOpsModeEnabled = false,
}: IPackageAdvancedOptionsProps) => {
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);
  const name = selectedPackage?.name || "";
  const ext = getExtensionFromFileName(name);

  const renderAdvancedOptions = () => {
    if (!isPackageType(ext)) {
      // this should never happen
      return null;
    }

    return (
      <AdvancedOptionsFields
        className={`${baseClass}__input-fields`}
        showSchemaButton={showSchemaButton}
        installScriptTooltip={getInstallScriptTooltip(ext)}
        installScriptHelpText={getInstallHelpText(ext)}
        installScriptReadOnly={isScriptOnlyPackageType(ext)}
        postInstallScriptHelpText={getPostInstallHelpText(ext)}
        uninstallScriptTooltip={getUninstallScriptTooltip(ext)}
        uninstallScriptHelpText={getUninstallHelpText(ext)}
        errors={errors}
        preInstallQuery={preInstallQuery}
        installScript={installScript}
        postInstallScript={postInstallScript}
        uninstallScript={uninstallScript}
        onClickShowSchema={onClickShowSchema}
        onChangePreInstallQuery={onChangePreInstallQuery}
        onChangeInstallScript={onChangeInstallScript}
        onChangePostInstallScript={onChangePostInstallScript}
        onChangeUninstallScript={onChangeUninstallScript}
        gitopsCompatible={gitopsCompatible}
        gitOpsModeEnabled={gitOpsModeEnabled}
      />
    );
  };

  const requiresAdvancedOptions =
    ext === "exe" || ext === "zip" || ext === "tar.gz";

  return (
    <div className={baseClass}>
      <RevealButton
        className={`${baseClass}__accordion-title`}
        isShowing={showAdvancedOptions}
        showText="Advanced options"
        hideText="Advanced options"
        caretPosition="after"
        onClick={() => setShowAdvancedOptions(!showAdvancedOptions)}
        variant="secondary"
        disabled={!selectedPackage || requiresAdvancedOptions}
        disabledTooltipContent={
          requiresAdvancedOptions ? (
            <>Install and uninstall scripts are required for .{ext} packages.</>
          ) : (
            <>
              Choose a file to modify <br />
              advanced options.
            </>
          )
        }
      />
      {(showAdvancedOptions ||
        ext === "exe" ||
        ext === "zip" ||
        ext === "tar.gz") &&
        !!selectedPackage &&
        renderAdvancedOptions()}
    </div>
  );
};

export default PackageAdvancedOptions;
