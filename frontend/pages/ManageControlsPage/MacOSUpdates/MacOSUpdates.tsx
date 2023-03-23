import React, { useContext } from "react";

import { AppContext } from "context/app";
import { NO_TEAM_ID } from "interfaces/team";

import OperatingSystems from "pages/DashboardPage/cards/OperatingSystems";
import useInfoCard from "pages/DashboardPage/components/InfoCard";

import PremiumFeatureMessage from "components/PremiumFeatureMessage";

import OsMinVersionForm from "./components/OsMinVersionForm";
import NudgePreview from "./components/NudgePreview";

const baseClass = "mac-os-updates";

const MacOSUpdates = () => {
  const { currentTeam, isPremiumTier } = useContext(AppContext);
  const teamId =
    currentTeam?.id === undefined || currentTeam.id < NO_TEAM_ID
      ? 0 // coerce undefined and -1 to 0 for 'No team'
      : currentTeam.id;

  const OperatingSystemCard = useInfoCard({
    title: "macOS versions",
    children: (
      <OperatingSystems
        currentTeamId={teamId}
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
      <>
        <p className={`${baseClass}__description`}>
          Remotely encourage the installation of macOS updates on hosts assigned
          to this team.
        </p>
        <div className={`${baseClass}__content`}>
          <div className={`${baseClass}__form-table-content`}>
            <div className={`${baseClass}__os-versions-card`}>
              {OperatingSystemCard}
            </div>
            <div className={`${baseClass}__os-version-form`}>
              <OsMinVersionForm currentTeamId={teamId} key={teamId} />
            </div>
          </div>
          <div className={`${baseClass}__nudge-preview`}>
            <NudgePreview />
          </div>
        </div>
      </>
    </div>
  ) : (
    <PremiumFeatureMessage />
  );
};

export default MacOSUpdates;
