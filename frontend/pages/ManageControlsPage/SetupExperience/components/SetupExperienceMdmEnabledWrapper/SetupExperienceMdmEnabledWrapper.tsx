import React, { useContext } from "react";
import { InjectedRouter } from "react-router";
import { AppContext } from "context/app";

import TurnOnMdmMessage from "components/TurnOnMdmMessage";

type SetupExperienceConfigCondition =
  | "appleMdmConfigured"
  | "abmConfigured"
  | "androidMdmConfigured";
interface ISetupExperienceMdmEnabledWrapperProps {
  router: InjectedRouter;
  children: React.ReactNode;
  conditions?: SetupExperienceConfigCondition[];
  supportedPlatformPrefix?: string;
}
const SetupMdmEnabledWrapper = ({
  router,
  children,
  conditions = [],
  supportedPlatformPrefix = "",
}: ISetupExperienceMdmEnabledWrapperProps) => {
  const { config } = useContext(AppContext);

  const conditionMap: Record<SetupExperienceConfigCondition, boolean> = {
    appleMdmConfigured: !!config?.mdm.enabled_and_configured,
    abmConfigured: !!config?.mdm.apple_bm_enabled_and_configured,
    androidMdmConfigured: !!config?.mdm.android_enabled_and_configured,
  };

  conditions.forEach((condition) => {
    if (!conditionMap[condition]) {
      return (
        <TurnOnMdmMessage
          header="Additional configuration required"
          info={`${supportedPlatformPrefix}To customize, first turn on automatic enrollment.`}
          buttonText="Turn on"
          router={router}
        />
      );
    }
  });

  return <>{children}</>;
};

export default SetupMdmEnabledWrapper;
