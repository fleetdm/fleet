import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import paths from "router/paths";
import { Link } from "react-router";
import { AppContext } from "context/app";
import { find } from "lodash";

import teamsAPI from "services/entities/teams";
import { ITeam } from "interfaces/team";
import { ISoftware } from "interfaces/software";

import TeamsDropdown from "components/TeamsDropdown";
import Button from "components/buttons/Button";
import InfoCard from "./components/InfoCard";
import HostsSummary from "./cards/HostsSummary";
import ActivityFeed from "./cards/ActivityFeed";
import Software from "./cards/Software";
import LearnFleet from "./cards/LearnFleet";
import LinkArrow from "../../../assets/images/icon-arrow-right-vibrant-blue-10x18@2x.png";

interface ITeamsResponse {
  teams: ITeam[];
}

const baseClass = "homepage";

const Homepage = (): JSX.Element => {
  const { MANAGE_HOSTS } = paths;
  const { 
    config, 
    currentTeam, 
    isPremiumTier, 
    isPreviewMode,
    setCurrentTeam,
  } = useContext(AppContext);

  const [isSoftwareModalOpen, setIsSoftwareModalOpen] = useState<boolean>(
    false
  );

  const { data: teams, isLoading: isLoadingTeams } = useQuery<
    ITeamsResponse,
    Error,
    ITeam[]
  >(["teams"], () => teamsAPI.loadAll(), {
    enabled: !!isPremiumTier,
    select: (data: ITeamsResponse) => data.teams,
  });

  const handleTeamSelect = (teamId: number) => {
    const selectedTeam = find(teams, ["id", teamId]);
    setCurrentTeam(selectedTeam);
  };

  const canSeeActivity = !isPreviewMode && (!isPremiumTier || !currentTeam);

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__header-wrap`}>
        <div className={`${baseClass}__header`}>
          {isPremiumTier ? (
            <TeamsDropdown
              currentTeamId={currentTeam?.id || 0}
              isLoading={isLoadingTeams}
              teams={teams || []}
              onChange={(newSelectedValue: number) =>
                handleTeamSelect(newSelectedValue)
              }
            />
          ) : (
            <h1 className={`${baseClass}__title`}>
              <span>{config?.org_name}</span>
            </h1>
          )}
        </div>
      </div>
      <div className={`${baseClass}__section one-column`}>
        <InfoCard
          title="Hosts"
          action={{ type: "link", to: MANAGE_HOSTS, text: "View all hosts" }}
        >
          <HostsSummary />
        </InfoCard>
      </div>
      {!isPreviewMode && (
        <div className={`${baseClass}__section two-column`}>
          <InfoCard title="Welcome to Fleet">
            <LearnFleet />
          </InfoCard>
          <InfoCard title="Learn how to use Fleet">
            <ActivityFeed />
          </InfoCard>
        </div>
      )}
      {canSeeActivity && (
        <div className={`${baseClass}__section one-column`}>
          <InfoCard title="Activity">
            <ActivityFeed />
          </InfoCard>
        </div>
      )}
      {/* TODO: Re-add this commented out section once the /software API is running */}
      {/* <div className={`
        ${baseClass}__section 
        ${currentTeam ? 'one' : 'two'}-column
      `}>
        {!currentTeam && (
          <InfoCard 
            title="Software"
            action={{ 
              type: button, 
              text: "View all software", 
              onClick: () => setIsSoftwareModalOpen(true)
            }}
          >
            <Software
              isModalOpen={isSoftwareModalOpen}
              setIsSoftwareModalOpen={setIsSoftwareModalOpen}
            />
          </InfoCard>
        )}
        <InfoCard title="Activity">
          <ActivityFeed />
        </InfoCard>
      </div> */}
    </div>
  );
};

export default Homepage;
