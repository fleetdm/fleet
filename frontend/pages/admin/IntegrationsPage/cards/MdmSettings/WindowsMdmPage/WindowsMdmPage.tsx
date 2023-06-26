import React, { useContext } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import configAPI from "services/entities/config";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import MainContent from "components/MainContent/MainContent";
import Button from "components/buttons/Button";
import BackLink from "components/BackLink/BackLink";

const baseClass = "windows-mdm-page";

interface IWindowsMdmOnContentProps {
  router: InjectedRouter;
}

const WindowsMdmOnContent = ({ router }: IWindowsMdmOnContentProps) => {
  const { setConfig } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const turnOnWindowsMdm = async () => {
    try {
      const updatedConfig = await configAPI.update({
        mdm: {
          windows_enabled_and_configured: true,
        },
      });
      setConfig(updatedConfig);
      renderFlash("success", "Windows MDM turned on (servers excluded).");
    } catch {
      renderFlash("error", "Unable to turn on Windows MDM. Please try again.");
    } finally {
      router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
    }
  };

  return (
    <>
      <h1>Turn on Windwos MDM</h1>
      <p>
        This will turn MDM on for Windows hosts with fleetd, overriding existing
        MDM solutions.
      </p>
      <p>MDM won&apos;t be turned on for Windows servers</p>
      <Button onClick={turnOnWindowsMdm}>Turn on</Button>
    </>
  );
};

interface IWindowsMdmOffContentProps {
  router: InjectedRouter;
}

const WindowsMdmOffContent = ({ router }: IWindowsMdmOffContentProps) => {
  const { setConfig } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const turnOnWindowsMdm = async () => {
    try {
      const updatedConfig = await configAPI.update({
        mdm: {
          windows_enabled_and_configured: false,
        },
      });
      setConfig(updatedConfig);
      renderFlash("success", "Windows MDM turned off.");
    } catch {
      renderFlash("error", "Unable to turn off Windows MDM. Please try again.");
    } finally {
      router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
    }
  };

  return (
    <>
      <h1>Turn off Windows MDM</h1>
      <p>This will turn off MDM on each Windows host.</p>
      <Button onClick={turnOnWindowsMdm}>Turn off MDM</Button>
    </>
  );
};

interface IWindowsMdmPageProps {
  router: InjectedRouter;
}

const WindowsMdmPage = ({ router }: IWindowsMdmPageProps) => {
  const { config } = useContext(AppContext);

  const isWindowsMdmEnabled =
    config?.mdm?.windows_enabled_and_configured ?? false;

  return (
    <MainContent className={baseClass}>
      <>
        <BackLink
          text="Back to MDM"
          path={PATHS.ADMIN_INTEGRATIONS_MDM}
          className={`${baseClass}__back-to-mdm`}
        />
        {isWindowsMdmEnabled ? (
          <WindowsMdmOffContent router={router} />
        ) : (
          <WindowsMdmOnContent router={router} />
        )}
      </>
    </MainContent>
  );
};

export default WindowsMdmPage;
