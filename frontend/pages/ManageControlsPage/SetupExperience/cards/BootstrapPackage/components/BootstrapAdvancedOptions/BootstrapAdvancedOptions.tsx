import React, { useState } from "react";

import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import Checkbox from "components/forms/fields/Checkbox";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { notify } from "components/ToastNotification";
import TooltipWrapper from "components/TooltipWrapper";
import mdmAPI from "services/entities/mdm";

const baseClass = "bootstrap-advanced-options";

interface IBootstrapAdvancedOptionsProps {
  currentTeamId: number;
  disableInstallManually: boolean;
  selectManualAgentInstall: boolean;
  onChangeManualAgentInstall: (value: boolean) => void;
  disableManualEnrollmentInstall: boolean;
  selectManualEnrollmentInstall: boolean;
  onChangeManualEnrollmentInstall: (value: boolean) => void;
}

const BootstrapAdvancedOptions = ({
  currentTeamId,
  disableInstallManually,
  selectManualAgentInstall,
  onChangeManualAgentInstall,
  disableManualEnrollmentInstall,
  selectManualEnrollmentInstall,
  onChangeManualEnrollmentInstall,
}: IBootstrapAdvancedOptionsProps) => {
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const onSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setIsSaving(true);
    try {
      await mdmAPI.updateSetupExperienceSettings({
        fleet_id: currentTeamId,
        macos_bootstrap_package_manual_enrollment: selectManualEnrollmentInstall,
        macos_manual_agent_install: selectManualAgentInstall,
      });
      notify.success("Successfully updated.");
    } catch (err) {
      notify.error("Something went wrong. Please try again.", {
        response: err,
      });
    }
    setIsSaving(false);
  };

  const tooltip = (
    <>
      Use this option if you&apos;re deploying a custom fleetd via bootstrap
      package. If enabled, Fleet won&apos;t install fleetd automatically. To use
      this option upload a bootstrap package first, and make sure to not use{" "}
      <b>Install software</b> and <b>Run script</b>.
    </>
  );

  const manualEnrollmentTooltip = (
    <>
      If enabled, Fleet also installs the bootstrap package on macOS hosts that
      enroll manually. If <b>Install Fleet&apos;s agent (fleetd) manually</b> is
      also enabled, Fleet won&apos;t install fleetd on these hosts.
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
      />
      {showAdvancedOptions && (
        <form onSubmit={onSubmit}>
          <GitOpsModeTooltipWrapper
            renderChildren={(gitopsDisable) => (
              <div className={`${baseClass}__advanced-options-controls`}>
                <Checkbox
                  value={selectManualEnrollmentInstall}
                  onChange={onChangeManualEnrollmentInstall}
                  disabled={gitopsDisable || disableManualEnrollmentInstall}
                >
                  <TooltipWrapper
                    tipContent={manualEnrollmentTooltip}
                    disableTooltip={gitopsDisable}
                  >
                    Install on manually enrolled hosts
                  </TooltipWrapper>
                </Checkbox>
                <Checkbox
                  value={selectManualAgentInstall}
                  onChange={onChangeManualAgentInstall}
                  disabled={gitopsDisable || disableInstallManually}
                >
                  <TooltipWrapper
                    tipContent={tooltip}
                    disableTooltip={gitopsDisable}
                  >
                    Install Fleet&apos;s agent (fleetd) manually
                  </TooltipWrapper>
                </Checkbox>
                {/* The wrapper div is needed to keep the button from stretching full width
                 * of the flex container */}
                <div>
                  <Button
                    disabled={
                      gitopsDisable ||
                      disableManualEnrollmentInstall ||
                      isSaving
                    }
                    type="submit"
                    isLoading={isSaving}
                  >
                    Save
                  </Button>
                </div>
              </div>
            )}
          />
        </form>
      )}
    </div>
  );
};

export default BootstrapAdvancedOptions;
