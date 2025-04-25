import React, { useContext, useState } from "react";

import mdmAPI from "services/entities/mdm";
import { NotificationContext } from "context/notification";

import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "bootstrap-advanced-options";

interface IBootstrapAdvancedOptionsProps {
  currentTeamId: number;
  enableInstallManually: boolean;
  defaultManualInstall: boolean;
}

const BootstrapAdvancedOptions = ({
  currentTeamId,
  enableInstallManually,
  defaultManualInstall,
}: IBootstrapAdvancedOptionsProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);
  const [manualAgentInstall, setManualAgentInstall] = useState(
    defaultManualInstall
  );

  const onSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();

    try {
      await mdmAPI.updateSetupExperienceSettings({
        team_id: currentTeamId,
        manual_agent_install: manualAgentInstall,
      });
      renderFlash("success", "Successfully updated.");
    } catch {
      renderFlash("error", "Something went wrong. Please try again.");
    }
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
            value={manualAgentInstall}
            onChange={() => setManualAgentInstall(!manualAgentInstall)}
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
