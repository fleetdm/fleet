import React, { useContext } from "react";
import { InjectedRouter } from "react-router";

import { AppContext } from "context/app";

import PremiumFeatureMessage from "components/PremiumFeatureMessage";

import OsMinVersionForm from "./components/OsMinVersionForm";
import NudgePreview from "./components/NudgePreview";
import TurnOnMdmMessage from "../components/TurnOnMdmMessage/TurnOnMdmMessage";
import CurrentVersionSection from "./components/CurrentVersionSection";
import TargetSection from "./components/TargetSection";

const baseClass = "os-updates";

interface IOSUpdates {
  router: InjectedRouter;
  teamIdForApi: number;
}

const OSUpdates = ({ router, teamIdForApi }: IOSUpdates) => {
  const { config, isPremiumTier } = useContext(AppContext);

  if (!config?.mdm.enabled_and_configured) {
    return <TurnOnMdmMessage router={router} />;
  }

  return isPremiumTier ? (
    <div className={baseClass}>
      <p className={`${baseClass}__description`}>
        Remotely encourage the installation of macOS updates on hosts assigned
        to this team.
      </p>
      <div className={`${baseClass}__content`}>
        <div className={`${baseClass}__form-table-content`}>
          <CurrentVersionSection currentTeamId={teamIdForApi} />
          <TargetSection currentTeamId={teamIdForApi} />
        </div>
        <div className={`${baseClass}__nudge-preview`}>
          <NudgePreview />
        </div>
      </div>
    </div>
  ) : (
    <PremiumFeatureMessage
      className={`${baseClass}__premium-feature-message`}
    />
  );
};

export default OSUpdates;
