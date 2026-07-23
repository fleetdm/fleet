import React, { useState } from "react";

import mdmAPI from "services/entities/mdm";

import TooltipWrapper from "components/TooltipWrapper";
import Checkbox from "components/forms/fields/Checkbox";
import Button from "components/buttons/Button";
import { notify } from "components/ToastNotification";
import RevealButton from "components/buttons/RevealButton";

const baseClass = "advanced-options-form";

interface IAdvancedOptionsFormProps {
  currentTeamId: number;
  defaultReleaseDevice: boolean;
}

const AdvancedOptionsForm = ({
  currentTeamId,
  defaultReleaseDevice,
}: IAdvancedOptionsFormProps) => {
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);
  const [releaseDevice, setReleaseDevice] = useState(defaultReleaseDevice);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();

    try {
      await mdmAPI.updateReleaseDeviceSetting(currentTeamId, releaseDevice);
      notify.success("Successfully updated.");
    } catch (err) {
      notify.error("Something went wrong. Please try again.", {
        response: err,
      });
    }
  };

  const tooltip = (
    <>
      When enabled, you&apos;re responsible for sending the DeviceConfigured
      command. (Default: <b>Off</b>)
    </>
  );

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
      />
      {showAdvancedOptions && (
        <form onSubmit={handleSubmit}>
          <Checkbox
            value={releaseDevice}
            onChange={() => setReleaseDevice(!releaseDevice)}
          >
            <TooltipWrapper tipContent={tooltip}>
              Release device manually
            </TooltipWrapper>
          </Checkbox>
          <Button type="submit">Save</Button>
        </form>
      )}
    </div>
  );
};

export default AdvancedOptionsForm;
