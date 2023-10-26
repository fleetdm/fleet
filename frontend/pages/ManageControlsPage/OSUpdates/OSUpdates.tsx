import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";

import { APP_CONTEXT_NO_TEAM_ID, ITeamConfig } from "interfaces/team";
import { AppContext } from "context/app";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";

import PremiumFeatureMessage from "components/PremiumFeatureMessage";

import NudgePreview from "./components/NudgePreview";
import TurnOnMdmMessage from "../components/TurnOnMdmMessage/TurnOnMdmMessage";
import CurrentVersionSection from "./components/CurrentVersionSection";
import TargetSection from "./components/TargetSection";

const baseClass = "os-updates";

interface IOSUpdates {
  router: InjectedRouter;
  teamIdForApi?: number;
}

const OSUpdates = ({ router, teamIdForApi }: IOSUpdates) => {
  const { config, isPremiumTier } = useContext(AppContext);

  const [selectedPlatform, setSelectedPlatform] = useState<"mac" | "windows">(
    "mac"
  );

  const { data: teamData, isLoading: isLoadingTeam, isError } = useQuery<
    ILoadTeamResponse,
    Error,
    ITeamConfig
  >(["team-config", teamIdForApi], () => teamsAPI.load(teamIdForApi), {
    refetchOnWindowFocus: false,
    enabled:
      teamIdForApi !== undefined && teamIdForApi > APP_CONTEXT_NO_TEAM_ID,
    select: (data) => data.team,
  });

  if (teamIdForApi === undefined) return null;

  // mdm is not enabled for mac or windows.
  if (
    !config?.mdm.enabled_and_configured &&
    !config?.mdm.windows_enabled_and_configured
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
