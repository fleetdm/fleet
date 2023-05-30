import React, { useContext } from "react";

import { AppContext } from "context/app";

import OperatingSystems from "pages/DashboardPage/cards/OperatingSystems";
import useInfoCard from "pages/DashboardPage/components/InfoCard";

import PremiumFeatureMessage from "components/PremiumFeatureMessage";

import OsMinVersionForm from "./components/OsMinVersionForm";
import NudgePreview from "./components/NudgePreview";

const baseClass = "mac-os-updates";

interface IMacOSUpdates {
  teamIdForApi: number;
}

const MacOSUpdates = ({ teamIdForApi }: IMacOSUpdates) => {
  const { isPremiumTier } = useContext(AppContext);

  const OperatingSystemCard = useInfoCard({
    title: "macOS versions",
    children: (
      <OperatingSystems
        currentTeamId={teamIdForApi}
        selectedPlatform="darwin"
        showTitle
        showDescription={false}
        includeNameColumn={false}
        setShowTitle={() => {
          return null;
        }}
      />
    ),
  });

  return isPremiumTier ? (
    <div className={baseClass}>
      <p className={`${baseClass}__description`}>
        Remotely encourage the installation of macOS updates on hosts assigned
        to this team.
      </p>
      <h2 className={`${baseClass}__title`}>MacOS updates settings</h2>
      <div className={`${baseClass}__content`}>
        <div className={`${baseClass}__form-table-content`}>
          <div className={`${baseClass}__os-versions-card`}>
            {OperatingSystemCard}
          </div>
          <div className={`${baseClass}__os-version-form`}>
            <OsMinVersionForm currentTeamId={teamIdForApi} key={teamIdForApi} />
          </div>
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

export default MacOSUpdates;
