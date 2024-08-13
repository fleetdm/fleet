import React, { useState } from "react";

import Editor from "components/Editor";
import CustomLink from "components/CustomLink";
import FleetAce from "components/FleetAce";
import RevealButton from "components/buttons/RevealButton";
import Checkbox from "components/forms/fields/Checkbox";

const baseClass = "add-package-advanced-options";

interface IAddPackageAdvancedOptionsProps {
  errors: { preInstallCondition?: string; postInstallScript?: string };
  showPreInstallCondition: boolean;
  showInstallScript: boolean;
  showPostInstallScript: boolean;
  preInstallCondition?: string;
  postInstallScript?: string;
  onTogglePreInstallCondition: (value: boolean) => void;
  onTogglePostInstallScript: (value: boolean) => void;
  onChangePreInstallCondition: (value?: string) => void;
  onChangeInstallScript: (value: string) => void;
  onChangePostInstallScript: (value?: string) => void;
  installScript: string;
}

const AddPackageAdvancedOptions = ({
  errors,
  showPreInstallCondition,
  showInstallScript,
  showPostInstallScript,
  preInstallCondition,
  postInstallScript,
  onTogglePreInstallCondition,
  onTogglePostInstallScript,
  onChangePreInstallCondition,
  onChangeInstallScript,
  onChangePostInstallScript,
  installScript,
}: IAddPackageAdvancedOptionsProps) => {
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);

  const onChangePreInstallCheckbox = () => {
    onTogglePreInstallCondition(!showPreInstallCondition);
  };

  const onChangePostInstallCheckbox = () => {
    onTogglePostInstallScript(!showPostInstallScript);
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
      />
      {showAdvancedOptions && (
        <div className={`${baseClass}__input-fields`}>
          <Checkbox
            value={showPreInstallCondition}
            onChange={onChangePreInstallCheckbox}
          >
            Pre-install condition
          </Checkbox>
          {showPreInstallCondition && (
            <FleetAce
              className="form-field"
              focus
              error={errors.preInstallCondition}
              value={preInstallCondition}
              label="Query"
              name="preInstallQuery"
              maxLines={10}
              onChange={onChangePreInstallCondition}
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
          )}
          {showInstallScript && (
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
                  Fleet will run this script on hosts to install software. Use
                  the
                  <br />
                  $INSTALLER_PATH variable to point to the installer.
                </>
              }
              isFormField
            />
          )}
          <Checkbox
            value={showPostInstallScript}
            onChange={onChangePostInstallCheckbox}
          >
            Post-install script
          </Checkbox>
          {showPostInstallScript && (
            <>
              <Editor
                label="Script"
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
            </>
          )}
        </div>
      )}
    </div>
  );
};

export default AddPackageAdvancedOptions;
