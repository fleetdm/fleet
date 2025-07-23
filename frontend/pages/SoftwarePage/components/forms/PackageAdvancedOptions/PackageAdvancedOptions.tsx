import React, { useState } from "react";
import { noop } from "lodash";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { getExtensionFromFileName } from "utilities/file/fileUtils";

import {
  isPackageType,
  isWindowsPackageType,
  isFleetMaintainedPackageType,
  PackageType,
} from "interfaces/package_type";

import CustomLink from "components/CustomLink";
import RevealButton from "components/buttons/RevealButton";

import { IPackageFormData } from "../PackageForm/PackageForm";
import AdvancedOptionsFields from "../AdvancedOptionsFields";

const getSupportedScriptTypeText = (pkgType: PackageType) => {
  return `Currently, ${
    isWindowsPackageType(pkgType) ? "PowerS" : "s"
  }hell scripts are supported.`;
};

const PKG_TYPE_TO_ID_TEXT = {
  pkg: "package IDs",
  deb: "package name",
  rpm: "package name",
  msi: "product code",
  exe: "software name",
} as const;

const getInstallScriptTooltip = (pkgType: PackageType) => {
  if (
    !isFleetMaintainedPackageType(pkgType) &&
    (pkgType === "exe" || pkgType === "tar.gz")
  ) {
    return `Required for ${
      pkgType === "exe" ? ".exe packages" : ".tar.gz archives"
    }.`;
  }
  return undefined;
};

const getInstallHelpText = (pkgType: PackageType) => {
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
  if (
    !isFleetMaintainedPackageType(pkgType) &&
    (pkgType === "exe" || pkgType === "tar.gz")
  ) {
    return `Required for ${
      pkgType === "exe" ? ".exe packages" : ".tar.gz archives"
    }.`;
  }
  return undefined;
};

const getUninstallHelpText = (pkgType: PackageType) => {
  if (isFleetMaintainedPackageType(pkgType)) {
    return "Currently, shell scripts are supported.";
  }

  if (pkgType === "exe") {
    return (
      <>
        For Windows, Fleet only creates uninstall scripts for .msi packages.
        $PACKAGE_ID will be populated with the software name from the .exe file
        after it&apos;s added.
        {getSupportedScriptTypeText(pkgType)}{" "}
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
        Currently, shell scripts are supported.{" "}
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
      />
    );
  };

  return (
    <div className={baseClass}>
      <RevealButton
        className={`${baseClass}__accordion-title`}
        isShowing={showAdvancedOptions}
        showText="Advanced options"
        hideText="Advanced options"
        caretPosition="after"
        onClick={() => setShowAdvancedOptions(!showAdvancedOptions)}
        disabled={!selectedPackage}
        disabledTooltipContent={
          <>
            Choose a file to modify <br />
            advanced options.
          </>
        }
      />
      {(showAdvancedOptions || ext === "exe" || ext === "tar.gz") &&
        !!selectedPackage &&
        renderAdvancedOptions()}
    </div>
  );
};

export default PackageAdvancedOptions;
