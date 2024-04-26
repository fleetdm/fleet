import FleetAce from "components/FleetAce";
import RevealButton from "components/buttons/RevealButton";
import Checkbox from "components/forms/fields/Checkbox";
import React, { useState } from "react";

const baseClass = "add-software-advanced-options";

interface IAddSoftwareAdvancedOptionsProps {
  preInstallCondition: string;
  postInstallScript: string;
  onChangePreInstallCondition: (value: string) => void;
  onChangePostInstallScript: (value: string) => void;
}

const AddSoftwareAdvancedOptions = ({
  preInstallCondition,
  postInstallScript,
  onChangePreInstallCondition,
  onChangePostInstallScript,
}: IAddSoftwareAdvancedOptionsProps) => {
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);
  const [showPreInstallCondition, setShowPreInstallCondition] = useState(false);
  const [showPostInstallScript, setShowPostInstallScript] = useState(false);

  const onChangePreInstallCheckbox = () => {
    setShowPreInstallCondition(!showPreInstallCondition);
    onChangePreInstallCondition("");
  };

  const onChangePostInstallCheckbox = () => {
    setShowPostInstallScript(!showPostInstallScript);
    onChangePostInstallScript("");
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
              focus
              value={preInstallCondition}
              label="Query"
              name="preInstallQuery"
              onChange={onChangePreInstallCondition}
              helpText="Software will be installed only if the query returns results "
            />
          )}
          <Checkbox
            value={showPostInstallScript}
            onChange={onChangePostInstallCheckbox}
          >
            Post-install script
          </Checkbox>
          {showPostInstallScript && (
            <FleetAce
              focus
              value={postInstallScript}
              name="postInstallScript"
              onChange={onChangePostInstallScript}
              helpText="Shell (macOS and Linux) or PowerShell (Windows)."
            />
          )}
        </div>
      )}
    </div>
  );
};

export default AddSoftwareAdvancedOptions;
