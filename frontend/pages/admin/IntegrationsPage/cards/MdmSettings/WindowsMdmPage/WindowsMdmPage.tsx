import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import { isAxiosError } from "axios";

import PATHS from "router/paths";
import configAPI from "services/entities/config";
import { getErrorReason } from "interfaces/errors";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import MainContent from "components/MainContent/MainContent";
import Button from "components/buttons/Button";
import BackLink from "components/BackLink/BackLink";
import Slider from "components/forms/fields/Slider";
import Checkbox from "components/forms/fields/Checkbox";

const baseClass = "windows-mdm-page";

interface ISetWindowsMdmOptions {
  enableMdm: boolean;
  enableAutoMigration: boolean;
  successMessage: string;
  errorMessage: string;
  router: InjectedRouter;
}

const useSetWindowsMdm = ({
  enableMdm,
  enableAutoMigration,
  successMessage,
  errorMessage,
  router,
}: ISetWindowsMdmOptions) => {
  const { setConfig } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const turnOnWindowsMdm = async () => {
    let flashErrMsg = "";
    try {
      const updatedConfig = await configAPI.updateMDMConfig(
        {
          windows_enabled_and_configured: enableMdm,
          windows_migration_enabled: enableAutoMigration,
        },
        true
      );
      setConfig(updatedConfig);
    } catch (e) {
      if (enableMdm && isAxiosError(e) && e.response?.status === 422) {
        flashErrMsg =
          getErrorReason(e, {
            nameEquals: "mdm.windows_enabled_and_configured",
          }) || errorMessage;
      } else {
        flashErrMsg = errorMessage;
      }
    } finally {
      router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
      if (flashErrMsg) {
        renderFlash("error", flashErrMsg);
      } else {
        renderFlash("success", successMessage);
      }
    }
  };

  return turnOnWindowsMdm;
};

interface IWindowsMdmPageProps {
  router: InjectedRouter;
}

const WindowsMdmPage = ({ router }: IWindowsMdmPageProps) => {
  const { config } = useContext(AppContext);

  const [mdmOn, setMdmOn] = useState(
    config?.mdm?.windows_enabled_and_configured ?? false
  );
  const [autoMigration, setAutoMigration] = useState(
    config?.mdm?.windows_migration_enabled ?? false
  );

  const updateWindowsMdm = useSetWindowsMdm({
    enableMdm: mdmOn,
    enableAutoMigration: autoMigration,
    successMessage: "Windows MDM settings successfully updated.",
    errorMessage: "Unable to update Windows MDM. Please try again.",
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
          />
          <p>{descriptionText}</p>
          <Checkbox
            disabled={!mdmOn}
            value={autoMigration}
            onChange={onChangeAutoMigration}
          >
            Automatically migrate hosts connected to another MDM solution
          </Checkbox>

          <Button variant="brand" onClick={onSaveMdm}>
            Save
          </Button>
        </form>
      </>
    </MainContent>
  );
};

export default WindowsMdmPage;
