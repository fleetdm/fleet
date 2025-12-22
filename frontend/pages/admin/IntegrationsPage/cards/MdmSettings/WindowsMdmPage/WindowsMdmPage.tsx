import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import configAPI from "services/entities/config";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import MainContent from "components/MainContent/MainContent";
import Button from "components/buttons/Button";
import BackButton from "components/BackButton";
import Slider from "components/forms/fields/Slider";
import Checkbox from "components/forms/fields/Checkbox";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Radio from "components/forms/fields/Radio";
import CustomLink from "components/CustomLink";

import { getErrorMessage } from "./helpers";

const baseClass = "windows-mdm-page";

interface ISetWindowsMdmOptions {
  enableMdm: boolean;
  enableAutoMigration: boolean;
  enrollmentType: "automatic" | "manual" | null;
  router: InjectedRouter;
}

const useSetWindowsMdm = ({
  enableMdm,
  enableAutoMigration,
  enrollmentType,
  router,
}: ISetWindowsMdmOptions) => {
  const { setConfig } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const turnOnWindowsMdm = async () => {
    try {
      const updatedConfig = await configAPI.updateMDMConfig(
        {
          enable_turn_on_windows_mdm_manually:
            enrollmentType !== null && enrollmentType === "manual",
          windows_enabled_and_configured: enableMdm,
          windows_migration_enabled: enableAutoMigration,
        },
        true
      );
      setConfig(updatedConfig);
      renderFlash("success", "Windows MDM settings successfully updated.", {
        persistOnPageChange: true,
      });
    } catch (e) {
      renderFlash("error", getErrorMessage(e), {
        persistOnPageChange: true,
      });
    }

    router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
  };

  return turnOnWindowsMdm;
};

interface IWindowsMdmPageProps {
  router: InjectedRouter;
}

const WindowsMdmPage = ({ router }: IWindowsMdmPageProps) => {
  const { config, isPremiumTier } = useContext(AppContext);
  const gitOpsModeEnabled = config?.gitops.gitops_mode_enabled;

  const [mdmOn, setMdmOn] = useState(
    config?.mdm?.windows_enabled_and_configured ?? false
  );
  const [autoMigration, setAutoMigration] = useState(
    config?.mdm?.windows_migration_enabled ?? false
  );
  const [enrollmentType, setEnrollmentType] = useState<
    "automatic" | "manual" | null
  >(() => {
    if (!config?.mdm?.windows_enabled_and_configured) return null;
    return config?.mdm?.enable_turn_on_windows_mdm_manually
      ? "manual"
      : "automatic";
  });

  const updateWindowsMdm = useSetWindowsMdm({
    enableMdm: mdmOn,
    enableAutoMigration: autoMigration,
    enrollmentType,
    router,
  });

  const onChangeMdmOn = () => {
    setMdmOn(!mdmOn);
    // if we are toggling off mdm we want to clear enrollment type. If we are toggling
    // it on, we want to set enrollment type to automatic by default
    !mdmOn ? setEnrollmentType("automatic") : setEnrollmentType(null);

    // if we are turning mdm off, also turn off auto migration
    mdmOn && setAutoMigration(false);
  };

  const onChangeEnrollmentType = (value: string) => {
    setAutoMigration(false);
    setEnrollmentType(value === "automaticEnrollment" ? "automatic" : "manual");
  };

  const onChangeAutoMigration = () => {
    setAutoMigration(!autoMigration);
  };

  const onSaveMdm = () => {
    updateWindowsMdm();
  };

  const descriptionText = mdmOn
    ? "Turns on MDM for Windows hosts that enroll to Fleet (excluding servers)."
    : "Hosts with MDM already turned on will not have MDM removed.";

  return (
    <MainContent className={baseClass}>
      <>
        <div className={`${baseClass}__header-links`}>
          <BackButton
            text="Back to MDM"
            path={PATHS.ADMIN_INTEGRATIONS_MDM}
            className={`${baseClass}__back-to-mdm`}
          />
        </div>
        <h1>Windows MDM</h1>
        <form>
          <Slider
            value={mdmOn}
            activeText="Windows MDM on"
            inactiveText="Windows MDM off"
            onChange={onChangeMdmOn}
            disabled={gitOpsModeEnabled}
          />
          {!isPremiumTier && <p>{descriptionText}</p>}
          {isPremiumTier && (
            // NOTE: first time using fieldset and legend. if we use this more we should make
            // a reusable component
            <fieldset
              disabled={!mdmOn}
              className={`${baseClass}__enrollment-type-fieldset form-field`}
            >
              {/* NOTE: we use this wrapper div to style the legend since legend
               does not work well with flexbox. the wrapper div helps the gap styling apply. */}
              <div>
                <legend className="form-field__label">
                  End user experience
                </legend>
              </div>
              <Radio
                id="automatic-enrollment"
                label="Automatic"
                value="automaticEnrollment"
                name="enrollmentType"
                checked={enrollmentType === "automatic"}
                onChange={onChangeEnrollmentType}
                disabled={!mdmOn}
                helpText="MDM is turned on when Fleet's agent is installed on Windows hosts (excluding servers)."
              />
              <Radio
                id="manual-enrollment"
                label="Manual"
                value="manualEnrollment"
                name="enrollmentType"
                checked={enrollmentType === "manual"}
                onChange={onChangeEnrollmentType}
                disabled={!mdmOn}
                helpText={
                  <>
                    Requires{" "}
                    <CustomLink
                      text="connecting Fleet to Microsoft Entra."
                      url={
                        PATHS.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT_WINDOWS
                      }
                    />{" "}
                    End users have to manually turn on MDM in{" "}
                    <b>Settings &gt; Access work or school.</b>
                  </>
                }
              />
            </fieldset>
          )}
          {isPremiumTier && enrollmentType !== "manual" && (
            <Checkbox
              disabled={!mdmOn || gitOpsModeEnabled}
              value={autoMigration}
              onChange={onChangeAutoMigration}
            >
              Automatically migrate hosts connected to another MDM solution
            </Checkbox>
          )}
          <GitOpsModeTooltipWrapper
            tipOffset={8}
            renderChildren={(disableChildren) => (
              <Button onClick={onSaveMdm} disabled={disableChildren}>
                Save
              </Button>
            )}
          />
        </form>
      </>
    </MainContent>
  );
};

export default WindowsMdmPage;
