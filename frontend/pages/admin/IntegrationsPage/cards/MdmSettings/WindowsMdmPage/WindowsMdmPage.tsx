import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import configAPI from "services/entities/config";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import MainContent from "components/MainContent/MainContent";
import Button from "components/buttons/Button";
import BackLink from "components/BackLink/BackLink";
import Slider from "components/forms/fields/Slider";
import Checkbox from "components/forms/fields/Checkbox";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import { getErrorMessage } from "./helpers";

const baseClass = "windows-mdm-page";

interface ISetWindowsMdmOptions {
  enableMdm: boolean;
  enableAutoMigration: boolean;
  router: InjectedRouter;
}

const useSetWindowsMdm = ({
  enableMdm,
  enableAutoMigration,
  router,
}: ISetWindowsMdmOptions) => {
  const { setConfig } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const turnOnWindowsMdm = async () => {
    try {
      const updatedConfig = await configAPI.updateMDMConfig(
        {
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

  const updateWindowsMdm = useSetWindowsMdm({
    enableMdm: mdmOn,
    enableAutoMigration: autoMigration,
    router,
  });

  const onChangeMdmOn = () => {
    setMdmOn(!mdmOn);
    mdmOn && setAutoMigration(false);
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
        <BackLink
          text="Back to MDM"
          path={PATHS.ADMIN_INTEGRATIONS_MDM}
          className={`${baseClass}__back-to-mdm`}
        />
        <h1>Manage Windows MDM</h1>
        <form>
          <Slider
            value={mdmOn}
            activeText="Windows MDM on"
            inactiveText="Windows MDM off"
            onChange={onChangeMdmOn}
            disabled={gitOpsModeEnabled}
          />
          <p>{descriptionText}</p>
          <Checkbox
            disabled={!isPremiumTier || !mdmOn || gitOpsModeEnabled}
            value={autoMigration}
            onChange={onChangeAutoMigration}
            tooltipContent={
              isPremiumTier ? "" : "This feature is included in Fleet Premium."
            }
          >
            Automatically migrate hosts connected to another MDM solution
          </Checkbox>
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
