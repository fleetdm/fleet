import React, { useContext, useEffect, useState } from "react";
import { InjectedRouter } from "react-router";

import { IConfig } from "interfaces/config";
import { AppContext } from "context/app";

import PremiumFeatureMessage from "components/PremiumFeatureMessage";

import NudgePreview from "./components/NudgePreview";
import TurnOnMdmMessage from "../components/TurnOnMdmMessage/TurnOnMdmMessage";
import CurrentVersionSection from "./components/CurrentVersionSection";
import TargetSection from "./components/TargetSection";

const baseClass = "os-updates";

const getSelectedPlatform = (appConfig: IConfig | null): "mac" | "windows" => {
  // We dont have the data ready yet so we default to mac.
  // This is usually when the users first comes to this page.
  if (appConfig === null) return "mac";

  // if the mac mdm is enable and configured we check the app config to see if
  // the mdm for mac is enabled. If it is, it does not matter if windows is
  // enabled and configured and we will always return "mac".
  return appConfig.mdm.enabled_and_configured ? "mac" : "windows";
};

interface IOSUpdates {
  router: InjectedRouter;
  teamIdForApi?: number;
}

const OSUpdates = ({ router, teamIdForApi }: IOSUpdates) => {
  const { config, isPremiumTier } = useContext(AppContext);

  // the default platform is mac and we later update this value when we have
  // done more checks.
  const [selectedPlatform, setSelectedPlatform] = useState<"mac" | "windows">(
    "mac"
  );

  // we have to use useEffect here as we need to update our selected platform
  // state when the app config is updated. This is usually when we get the app
  // config response from the server and it is no longer `null`.
  useEffect(() => {
    setSelectedPlatform(getSelectedPlatform(config));
  }, [config]);

  if (config === null || teamIdForApi === undefined) return null;

  // mdm is not enabled for mac or windows.
  if (
    !config.mdm.enabled_and_configured &&
    !config.mdm.windows_enabled_and_configured
  ) {
    return <TurnOnMdmMessage router={router} />;
  }

  // Not premium shows premium message
  if (!isPremiumTier) {
    return (
      <PremiumFeatureMessage
        className={`${baseClass}__premium-feature-message`}
      />
    );
  }

  const handleSelectPlatform = (platform: "mac" | "windows") => {
    setSelectedPlatform(platform);
  };

  return (
    <div className={baseClass}>
      <p className={`${baseClass}__description`}>
        Remotely encourage the installation of macOS updates on hosts assigned
        to this team.
      </p>
      <div className={`${baseClass}__content`}>
        <div className={`${baseClass}__form-table-content`}>
          <CurrentVersionSection currentTeamId={teamIdForApi} />
          <TargetSection
            currentTeamId={teamIdForApi}
            onSelectAccordionItem={handleSelectPlatform}
          />
        </div>
        <div className={`${baseClass}__nudge-preview`}>
          <NudgePreview platform={selectedPlatform} />
        </div>
      </div>
    </div>
  );
};

export default OSUpdates;
