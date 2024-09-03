import React, { useState } from "react";

import Editor from "components/Editor";
import CustomLink from "components/CustomLink";
import FleetAce from "components/FleetAce";
import RevealButton from "components/buttons/RevealButton";
import { IAddPackageFormData } from "../AddPackageForm/AddPackageForm";

const baseClass = "add-package-advanced-options";

interface IAddPackageAdvancedOptionsProps {
  errors: { preInstallQuery?: string; postInstallScript?: string };
  selectedPackage: IAddPackageFormData["software"];
  preInstallQuery?: string;
  installScript: string;
  postInstallScript?: string;
  onChangePreInstallQuery: (value?: string) => void;
  onChangeInstallScript: (value: string) => void;
  onChangePostInstallScript: (value?: string) => void;
}

const AddPackageAdvancedOptions = ({
  errors,
  selectedPackage,
  preInstallQuery,
  installScript,
  postInstallScript,
  onChangePreInstallQuery,
  onChangeInstallScript,
  onChangePostInstallScript,
}: IAddPackageAdvancedOptionsProps) => {
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);

  const noPackage = selectedPackage === null;

  return (
    <div className={baseClass}>
      <RevealButton
        className={`${baseClass}__accordion-title`}
        isShowing={showAdvancedOptions}
        showText="Advanced options"
        hideText="Advanced options"
        caretPosition="after"
        onClick={() => setShowAdvancedOptions(!showAdvancedOptions)}
        disabled={noPackage}
        disabledTooltipContent={
          noPackage ? "Choose a file to modify advanced options." : undefined
        }
      />
      {showAdvancedOptions && (
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
            helpText="Shell (macOS and Linux) or PowerShell (Windows)."
            label="Install script"
            labelTooltip={
              <>
                Fleet will run this script on hosts to install software. Use the
                <br />
                $INSTALLER_PATH variable to point to the installer.
              </>
            }
            isFormField
          />
          <Editor
            label="Post-install script"
            labelTooltip="Fleet will run this script after install."
            focus
            error={errors.postInstallScript}
            wrapEnabled
            name="post-install-script-editor"
            maxLines={10}
            onChange={onChangePostInstallScript}
            value={postInstallScript}
            helpText="Shell (macOS and Linux) or PowerShell (Windows)."
            isFormField
          />
        </div>
      )}
    </div>
  );
};

export default AddPackageAdvancedOptions;
