import React, { useState } from "react";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import {
  isPackageType,
  isWindowsPackageType,
  PackageType,
} from "interfaces/package_type";

import Editor from "components/Editor";
import CustomLink from "components/CustomLink";
import FleetAce from "components/FleetAce";
import RevealButton from "components/buttons/RevealButton";
import { IAddPackageFormData } from "../AddPackageForm/AddPackageForm";

const getSupportedScriptTypeText = (pkgType: PackageType) => {
  return `Currently, ${
    isWindowsPackageType(pkgType) ? "Power" : ""
  }Shell scripts are supported.`;
};

const PKG_TYPE_TO_ID_TEXT = {
  pkg: "package IDs",
  deb: "package name",
  msi: "product code",
  exe: "software name",
} as const;

const getInstallHelpText = (pkgType: PackageType) => (
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

const getPostInstallHelpText = (pkgType: PackageType) => {
  return getSupportedScriptTypeText(pkgType);
};

const getUninstallHelpText = (pkgType: PackageType) => {
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

const baseClass = "add-package-advanced-options";

interface IAddPackageAdvancedOptionsProps {
  errors: { preInstallQuery?: string; postInstallScript?: string };
  selectedPackage: IAddPackageFormData["software"];
  preInstallQuery?: string;
  installScript: string;
  postInstallScript?: string;
  uninstallScript?: string;
  onChangePreInstallQuery: (value?: string) => void;
  onChangeInstallScript: (value: string) => void;
  onChangePostInstallScript: (value?: string) => void;
  onChangeUninstallScript: (value?: string) => void;
}

const AddPackageAdvancedOptions = ({
  errors,
  selectedPackage,
  preInstallQuery,
  installScript,
  postInstallScript,
  uninstallScript,
  onChangePreInstallQuery,
  onChangeInstallScript,
  onChangePostInstallScript,
  onChangeUninstallScript,
}: IAddPackageAdvancedOptionsProps) => {
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);

  const renderAdvancedOptions = () => {
    const name = selectedPackage?.name || "";
    const ext = name.split(".").pop() as PackageType;
    if (!isPackageType(ext)) {
      // this should never happen
      return null;
    }
    return (
      <div className={`${baseClass}__input-fields`}>
        <FleetAce
          className="form-field"
          focus
          error={errors.preInstallQuery}
          value={preInstallQuery}
          placeholder="SELECT * FROM osquery_info WHERE start_time > 1"
          label="Pre-install query"
          name="preInstallQuery"
          maxLines={10}
          onChange={onChangePreInstallQuery}
          helpText={
            <>
              Software will be installed only if the{" "}
              <CustomLink
                className={`${baseClass}__table-link`}
                text="query returns results"
                url="https://fleetdm.com/tables"
                newTab
              />
            </>
          }
        />
        <Editor
          wrapEnabled
          maxLines={10}
          name="install-script"
          onChange={onChangeInstallScript}
          value={installScript}
          helpText={getInstallHelpText(ext)}
          label="Install script"
          isFormField
        />
        <Editor
          label="Post-install script"
          focus
          error={errors.postInstallScript}
          wrapEnabled
          name="post-install-script-editor"
          maxLines={10}
          onChange={onChangePostInstallScript}
          value={postInstallScript}
          helpText={getPostInstallHelpText(ext)}
          isFormField
        />
        <Editor
          label="Uninstall script"
          focus
          wrapEnabled
          name="uninstall-script-editor"
          maxLines={20}
          onChange={onChangeUninstallScript}
          value={uninstallScript}
          helpText={getUninstallHelpText(ext)}
          isFormField
        />
      </div>
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
          selectedPackage
            ? "Choose a file to modify advanced options."
            : undefined
        }
      />
      {showAdvancedOptions && !!selectedPackage && renderAdvancedOptions()}
    </div>
  );
};

export default AddPackageAdvancedOptions;
