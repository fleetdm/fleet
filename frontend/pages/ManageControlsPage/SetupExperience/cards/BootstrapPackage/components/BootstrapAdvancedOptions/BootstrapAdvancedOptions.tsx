import React, { useContext, useEffect, useState } from "react";

import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import Card from "components/Card";
import CustomLink from "components/CustomLink";
import Checkbox from "components/forms/fields/Checkbox";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { notify } from "components/ToastNotification";
import TooltipWrapper from "components/TooltipWrapper";
import { AppContext } from "context/app";
import paths from "router/paths";
import mdmAPI from "services/entities/mdm";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";

const baseClass = "bootstrap-advanced-options";

export interface IBootstrapAdvancedOptionsFormData {
  macos_enable_bootstrap_package_during_manual_enrollment: boolean;
  macos_enable_bootstrap_package_during_migration: boolean;
  macos_manual_agent_install: boolean;
}

interface IBootstrapAdvancedOptionsProps {
  currentTeamId: number;
  initialValues: IBootstrapAdvancedOptionsFormData;
  hasUploadedPackage: boolean;
  hasSetupExperienceSoftwareOrScript: boolean;
  enrollSecret: string | null;
}

const BootstrapAdvancedOptions = ({
  currentTeamId,
  initialValues,
  hasUploadedPackage,
  hasSetupExperienceSoftwareOrScript,
  enrollSecret,
}: IBootstrapAdvancedOptionsProps) => {
  const { config } = useContext(AppContext);
  const [formData, setFormData] = useState<IBootstrapAdvancedOptionsFormData>(
    initialValues
  );
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => setFormData(initialValues), [initialValues]);
  const onChange = (key: keyof IBootstrapAdvancedOptionsFormData) => (
    value: boolean
  ) => {
    setFormData((prev) => ({ ...prev, [key]: value }));
  };

  const onSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setIsSaving(true);
    try {
      await mdmAPI.updateSetupExperienceSettings({
        fleet_id: currentTeamId,
        ...formData,
      });
      notify.success("Successfully updated.");
    } catch (err) {
      notify.error("Something went wrong. Please try again.", {
        response: err,
      });
    } finally {
      setIsSaving(false);
    }
  };

  const manualAgentInstallTooltip = (
    <>
      Use this option if you&apos;re deploying a custom fleetd via bootstrap
      package. If enabled, Fleet won&apos;t install fleetd automatically. To use
      this option upload a bootstrap package first, and make sure to not use{" "}
      <b>Install software</b> and <b>Run script</b>.
    </>
  );

  const manualEnrollmentTooltip =
    enrollSecret != null ? (
      <>
        Installs when hosts turn on MDM manually using the{" "}
        <CustomLink
          url={`${
            config?.server_settings.server_url ?? ""
          }/enroll?enroll_secret=${encodeURIComponent(enrollSecret)}`}
          text="enrollment link"
          newTab
          variant="tooltip-link"
        />{" "}
        (not automatically via Apple Business).
      </>
    ) : (
      <>
        Installs when hosts turn on MDM manually using the enrollment link (not
        automatically via Apple Business). To use the enrollment link,{" "}
        <CustomLink
          text="add an enroll secret"
          variant="tooltip-link"
          url={getPathWithQueryParams(paths.MANAGE_HOSTS, {
            manage_enroll_secrets: 1,
            fleet_id: currentTeamId,
          })}
        />
        .
      </>
    );

  const migrationEnrollmentTooltip = (
    <>
      Installs when hosts migrate from another MDM, including re-enrollment via
      sudo profiles renew.{" "}
      <CustomLink
        text="Learn more"
        newTab
        variant="tooltip-link"
        url={`${LEARN_MORE_ABOUT_BASE_LINK}/mdm-migration`}
      />
    </>
  );

  return (
    <div className={baseClass}>
      <form onSubmit={onSubmit}>
        <GitOpsModeTooltipWrapper
          isInputField
          renderChildren={(gitopsDisable) => (
            <Card className={`${baseClass}__settings-card`} color="white">
              <div className={`${baseClass}__settings-card-content`}>
                <Checkbox
                  value={
                    formData.macos_enable_bootstrap_package_during_manual_enrollment
                  }
                  onChange={onChange(
                    "macos_enable_bootstrap_package_during_manual_enrollment"
                  )}
                  disabled={gitopsDisable || !hasUploadedPackage}
                >
                  <TooltipWrapper
                    tipContent={manualEnrollmentTooltip}
                    disableTooltip={gitopsDisable}
                  >
                    Install during manual enrollment
                  </TooltipWrapper>
                </Checkbox>
                <Checkbox
                  value={
                    formData.macos_enable_bootstrap_package_during_migration
                  }
                  onChange={onChange(
                    "macos_enable_bootstrap_package_during_migration"
                  )}
                  disabled={gitopsDisable || !hasUploadedPackage}
                >
                  <TooltipWrapper
                    tipContent={migrationEnrollmentTooltip}
                    disableTooltip={gitopsDisable}
                  >
                    Install during migration
                  </TooltipWrapper>
                </Checkbox>
                <RevealButton
                  className={`${baseClass}__accordion-title`}
                  isShowing={showAdvancedOptions}
                  showText="Advanced options"
                  hideText="Advanced options"
                  caretPosition="after"
                  onClick={() => setShowAdvancedOptions(!showAdvancedOptions)}
                />
                {showAdvancedOptions && (
                  <Checkbox
                    value={formData.macos_manual_agent_install}
                    onChange={onChange("macos_manual_agent_install")}
                    disabled={
                      gitopsDisable ||
                      !hasUploadedPackage ||
                      hasSetupExperienceSoftwareOrScript
                    }
                  >
                    <TooltipWrapper
                      tipContent={manualAgentInstallTooltip}
                      disableTooltip={gitopsDisable}
                    >
                      Deploy custom agent
                    </TooltipWrapper>
                  </Checkbox>
                )}
              </div>
              <Button
                disabled={gitopsDisable || !hasUploadedPackage || isSaving}
                type="submit"
                isLoading={isSaving}
              >
                Save
              </Button>
            </Card>
          )}
        />
      </form>
    </div>
  );
};

export default BootstrapAdvancedOptions;
