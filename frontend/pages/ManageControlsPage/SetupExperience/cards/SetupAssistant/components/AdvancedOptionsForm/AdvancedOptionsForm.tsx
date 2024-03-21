import React, { useState } from "react";

import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";
import Checkbox from "components/forms/fields/Checkbox";
import Button from "components/buttons/Button";

const baseClass = "advanced-options-form";

const AdvancedOptionsForm = () => {
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);
  const [releaseDevice, setReleaseDevice] = useState(false);

  const accordionText = showAdvancedOptions ? "Hide" : "Show";
  const icon = showAdvancedOptions ? "chevron-up" : "chevron-down";

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    console.log("release?", releaseDevice);
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
