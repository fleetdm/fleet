import React, { useContext, useState } from "react";

import mdmAPI from "services/entities/mdm";

import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";
import Checkbox from "components/forms/fields/Checkbox";
import Button from "components/buttons/Button";
import { NotificationContext } from "context/notification";

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
  const { renderFlash } = useContext(NotificationContext);

  const accordionText = showAdvancedOptions ? "Hide" : "Show";
  const icon = showAdvancedOptions ? "chevron-up" : "chevron-down";

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();

    try {
      await mdmAPI.updateReleaseDeviceSetting(currentTeamId, releaseDevice);
      renderFlash("success", "Successfully updated.");
    } catch {
      renderFlash("error", "Something went wrong. Please try again.");
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
      <div
        className={`${baseClass}__accordion-title`}
        onClick={() => setShowAdvancedOptions(!showAdvancedOptions)}
      >
        <span>{accordionText} advanced options</span>
        <Icon name={icon} color="core-fleet-blue" />
      </div>
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
