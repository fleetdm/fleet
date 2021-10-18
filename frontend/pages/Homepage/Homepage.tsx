import React, { useContext } from "react";
import { useQuery } from "react-query";
import paths from "router/paths";
import { Link } from "react-router";
import { AppContext } from "context/app";
import { find } from "lodash";

import teamsAPI from "services/entities/teams";
import { ITeam } from "interfaces/team";

import TeamsDropdown from "components/TeamsDropdown";
import HostsSummary from "./HostsSummary";
import ActivityFeed from "./ActivityFeed";
import LinkArrow from "../../../assets/images/icon-arrow-right-vibrant-blue-10x18@2x.png";

interface ITeamsResponse {
  teams: ITeam[];
}

const baseClass = "homepage";

const Homepage = (): JSX.Element => {
  const { MANAGE_HOSTS } = paths;
  const {
    currentTeam,
    isPremiumTier,
    setCurrentTeam,
  } = useContext(AppContext);

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

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__header-wrap`}>
        <div className={`${baseClass}__header`}>
          <TeamsDropdown
            currentTeamId={currentTeam?.id || 0}
            isLoading={isLoadingTeams}
            teams={teams || []}
            onChange={(newSelectedValue: number) =>
              handleTeamSelect(newSelectedValue)
            }
          />
        </div>
      </div>
      <div className={`${baseClass}__section one-column`}>
        <div className={`${baseClass}__info-card`}>
          <div className={`${baseClass}__section-title`}>
            <h2>Hosts</h2>
            <Link to={MANAGE_HOSTS} className={`${baseClass}__host-link`}>
              <span>View all hosts</span>
              <img src={LinkArrow} alt="link arrow" id="link-arrow" />
            </Link>
          </div>
          <HostsSummary />
        </div>
      </div>
      <div className={`
        ${baseClass}__section 
        ${currentTeam ? 'two' : 'one'}-column
      `}>
        <div className={`${baseClass}__info-card`}>
          <div className={`${baseClass}__section-title`}>
            <h2>Activity</h2>
          </div>
          <ActivityFeed />
        </div>
      </div>
    </div>
  );
};

export default Homepage;
