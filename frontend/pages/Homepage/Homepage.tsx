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
import HostsSummary from "./HostsSummary";
import ActivityFeed from "./ActivityFeed";
import LinkArrow from "../../../assets/images/icon-arrow-right-vibrant-blue-10x18@2x.png";
import Software from "./Software";

interface ITeamsResponse {
  teams: ITeam[];
}

const baseClass = "homepage";

const Homepage = (): JSX.Element => {
  const { MANAGE_HOSTS } = paths;
  const { config, currentTeam, isPremiumTier, setCurrentTeam } = useContext(
    AppContext
  );

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

  const canSeeActivity = !isPremiumTier || !currentTeam;

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
        <div className={`${baseClass}__info-card`}>
          <div className={`${baseClass}__section-title`}>
            <h2>Hosts</h2>
            <Link to={MANAGE_HOSTS} className={`${baseClass}__action-button`}>
              <span>View all hosts</span>
              <img src={LinkArrow} alt="link arrow" id="link-arrow" />
            </Link>
          </div>
          <HostsSummary />
        </div>
      </div>
      {canSeeActivity && (
        <div className={`${baseClass}__section one-column`}>
          <div className={`${baseClass}__info-card`}>
            <div className={`${baseClass}__section-title`}>
              <h2>Activity</h2>
            </div>
            <ActivityFeed />
          </div>
        </div>
      )}
      {/* TODO: Re-add this commented out section once the /software API is running */}
      {/* <div className={`
        ${baseClass}__section 
        ${currentTeam ? 'one' : 'two'}-column
      `}>
        {!currentTeam && (
          <div className={`${baseClass}__info-card`}>
            <div className={`${baseClass}__section-title`}>
              <h2>Software</h2>
              <Button
                className={`${baseClass}__action-button`}
                variant="text-link"
                onClick={() => setIsSoftwareModalOpen(true)}
              >
                <>
                  <span>View all software</span>
                  <img src={LinkArrow} alt="link arrow" id="link-arrow" />
                </>
              </Button>
            </div>
            <Software
              isModalOpen={isSoftwareModalOpen}
              setIsSoftwareModalOpen={setIsSoftwareModalOpen}
            />
          </div>
        )}
        <div className={`${baseClass}__info-card`}>
          <div className={`${baseClass}__section-title`}>
            <h2>Activity</h2>
          </div>
          <ActivityFeed />
        </div>
      </div> */}
    </div>
  );
};

export default Homepage;
