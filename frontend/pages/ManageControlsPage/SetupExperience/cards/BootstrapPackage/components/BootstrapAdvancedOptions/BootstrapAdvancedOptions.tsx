import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";
import React, { useState } from "react";

const baseClass = "bootstrap-advanced-options";

interface IBootstrapAdvancedOptionsProps {
  enableInstallManually: boolean;
  defaultInstallManually: boolean;
}

const BootstrapAdvancedOptions = ({
  enableInstallManually,
  defaultInstallManually,
}: IBootstrapAdvancedOptionsProps) => {
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);
  const [releaseDevice, setReleaseDevice] = useState(defaultInstallManually);

  const onSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();

    // try {
    //   await mdmAPI.updateReleaseDeviceSetting(currentTeamId, releaseDevice);
    //   renderFlash("success", "Successfully updated.");
    // } catch {
    //   renderFlash("error", "Something went wrong. Please try again.");
    // }
  };

  const tooltip = (
    <>
      Use this option if you&apos;re deploying a custom fleetd via bootstrap
      package. If enabled, Fleet won&apos;t install fleetd automatically. To use
      this option upload a bootstrap package first.
    </>
  );

  return (
    <div className={baseClass}>
      <RevealButton
        className={`${baseClass}__accordion-title`}
        isShowing={showAdvancedOptions}
        showText="Show advanced options"
        hideText="Hide advanced options"
        caretPosition="after"
        onClick={() => setShowAdvancedOptions(!showAdvancedOptions)}
      />
      {showAdvancedOptions && (
        <form onSubmit={onSubmit}>
          <Checkbox
            value={releaseDevice}
            onChange={() => setReleaseDevice(!releaseDevice)}
            disabled={!enableInstallManually}
          >
            <TooltipWrapper tipContent={tooltip}>
              Install Fleet&apos;s agent (fleetd) manually
            </TooltipWrapper>
          </Checkbox>
          <Button disabled={!enableInstallManually} type="submit">
            Save
          </Button>
        </form>
      )}
    </div>
  );
};

export default BootstrapAdvancedOptions;
