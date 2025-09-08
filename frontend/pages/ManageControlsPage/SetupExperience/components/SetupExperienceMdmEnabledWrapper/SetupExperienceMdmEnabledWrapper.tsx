import React, { useContext } from "react";
import { InjectedRouter } from "react-router";
import { AppContext } from "context/app";

import TurnOnMdmMessage from "components/TurnOnMdmMessage";

import { SetupEmptyState } from "../../SetupExperience";

interface ISetupExperienceMdmEnabledWrapperProps {
  router: InjectedRouter;
  children: React.ReactNode;
}
/** Gates children with empty states if Apple MDM or ABM not enabled */
const SetupMdmEnabledWrapper = ({
  router,
  children,
}: ISetupExperienceMdmEnabledWrapperProps) => {
  const { config } = useContext(AppContext);
  if (!config?.mdm.enabled_and_configured) {
    return (
      <TurnOnMdmMessage
        header="Manage setup experience for macOS"
        info="To install software and run scripts when Macs first boot, first turn on automatic enrollment."
        buttonText="Turn on"
        router={router}
      />
    );
  }
  if (!config?.mdm.apple_bm_enabled_and_configured) {
    return <SetupEmptyState router={router} />;
  }
  return <>{children}</>;
};

export default SetupMdmEnabledWrapper;
