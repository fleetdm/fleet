import React, { useContext, useState } from "react";

import mdmAPI from "services/entities/mdm";
import { NotificationContext } from "context/notification";

import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

const baseClass = "bootstrap-advanced-options";

interface IBootstrapAdvancedOptionsProps {
  currentTeamId: number;
  disableInstallManually: boolean;
  selectManualAgentInstall: boolean;
  onChange: (value: boolean) => void;
}

const BootstrapAdvancedOptions = ({
  currentTeamId,
  disableInstallManually,
  selectManualAgentInstall,
  onChange,
}: IBootstrapAdvancedOptionsProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);

  const onSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    try {
      await mdmAPI.updateSetupExperienceSettings({
        team_id: currentTeamId,
        manual_agent_install: selectManualAgentInstall,
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
      this option upload a bootstrap package first, and make sure to not use{" "}
      <b>Install software</b> and <b>Run script</b>.
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
          <GitOpsModeTooltipWrapper
            renderChildren={(gitopsDisable) => (
              <div className={`${baseClass}__advanced-options-controls`}>
                <Checkbox
                  value={selectManualAgentInstall}
                  onChange={onChange}
                  disabled={gitopsDisable || disableInstallManually}
                >
                  <TooltipWrapper
                    tipContent={tooltip}
                    disableTooltip={gitopsDisable}
                  >
                    Install Fleet&apos;s agent (fleetd) manually
                  </TooltipWrapper>
                </Checkbox>
                <Button
                  disabled={gitopsDisable || disableInstallManually}
                  type="submit"
                >
                  Save
                </Button>
              </div>
            )}
          />
        </form>
      )}
    </div>
  );
};

export default BootstrapAdvancedOptions;
