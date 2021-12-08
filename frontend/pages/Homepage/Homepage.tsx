import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import paths from "router/paths";
import { AppContext } from "context/app";
import { find } from "lodash";

import hostSummaryAPI from "services/entities/host_summary";
import teamsAPI from "services/entities/teams";
import { IHostSummary, IHostSummaryPlatforms } from "interfaces/host_summary";
import { ITeam } from "interfaces/team";
import sortUtils from "utilities/sort";

import TeamsDropdown from "components/TeamsDropdown";
import InfoCard from "./components/InfoCard";
import HostsStatus from "./cards/HostsStatus";
import HostsSummary from "./cards/HostsSummary";
import ActivityFeed from "./cards/ActivityFeed";
import Software from "./cards/Software";
import LearnFleet from "./cards/LearnFleet";
import WelcomeHost from "./cards/WelcomeHost";

interface ITeamsResponse {
  teams: ITeam[];
}

const baseClass = "homepage";

const TAGGED_TEMPLATES = {
  hostsByTeamRoute: (teamId: number | undefined | null) => {
    return `${teamId ? `/?team_id=${teamId}` : ""}`;
  },
};

const Homepage = (): JSX.Element => {
  const { MANAGE_HOSTS } = paths;
  const {
    config,
    currentTeam,
    isPremiumTier,
    isFreeTier,
    isPreviewMode,
    isOnGlobalTeam,
    setCurrentTeam,
  } = useContext(AppContext);

  const [isSoftwareModalOpen, setIsSoftwareModalOpen] = useState<boolean>(
    false
  );
  const [totalCount, setTotalCount] = useState<string | undefined>();
  const [macCount, setMacCount] = useState<string>("0");
  const [windowsCount, setWindowsCount] = useState<string>("0");
  const [onlineCount, setOnlineCount] = useState<string | undefined>();
  const [offlineCount, setOfflineCount] = useState<string | undefined>();
  const [newCount, setNewCount] = useState<string | undefined>();

  const { data: teams } = useQuery<ITeamsResponse, Error, ITeam[]>(
    ["teams"],
    () => teamsAPI.loadAll(),
    {
      enabled: !!isPremiumTier,
      select: (data: ITeamsResponse) =>
        data.teams.sort((a, b) => sortUtils.caseInsensitiveAsc(a.name, b.name)),
      onSuccess: (responseTeams) => {
        if (!currentTeam && !isOnGlobalTeam && responseTeams.length) {
          setCurrentTeam(responseTeams[0]);
        }
      },
    }
  );

  const handleTeamSelect = (teamId: number) => {
    const selectedTeam = find(teams, ["id", teamId]);
    setCurrentTeam(selectedTeam);
  };

  useQuery<IHostSummary, Error, IHostSummary>(
    ["host summary", currentTeam],
    () => {
      return hostSummaryAPI.getSummary(currentTeam?.id);
    },
    {
      select: (data: IHostSummary) => data,
      onSuccess: (data: any) => {
        setTotalCount(data.totals_hosts_count.toLocaleString("en-US"));
        setOnlineCount(data.online_count.toLocaleString("en-US"));
        setOfflineCount(data.offline_count.toLocaleString("en-US"));
        setNewCount(data.new_count.toLocaleString("en-US"));
        const macHosts = data.platforms?.find(
          (platform: IHostSummaryPlatforms) => platform.platform === "darwin"
        ) || { platform: "darwin", hosts_count: 0 };
        setMacCount(macHosts.hosts_count.toLocaleString("en-US"));
        const windowsHosts = data.platforms?.find(
          (platform: IHostSummaryPlatforms) => platform.platform === "windows"
        ) || { platform: "windows", hosts_count: 0 };
        setWindowsCount(windowsHosts.hosts_count.toLocaleString("en-US"));
      },
    }
  );

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__wrapper body-wrap`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <div className={`${baseClass}__title`}>
                {isFreeTier && <h1>{config?.org_name}</h1>}
                {isPremiumTier &&
                  teams &&
                  (teams.length > 1 || isOnGlobalTeam) && (
                    <TeamsDropdown
                      selectedTeamId={currentTeam?.id || 0}
                      currentUserTeams={teams || []}
                      onChange={(newSelectedValue: number) =>
                        handleTeamSelect(newSelectedValue)
                      }
                    />
                  )}
                {isPremiumTier &&
                  !isOnGlobalTeam &&
                  teams &&
                  teams.length === 1 && <h1>{teams[0].name}</h1>}
              </div>
            </div>
          </div>
        </div>
        <div className={`${baseClass}__section one-column`}>
          <InfoCard
            title="Hosts"
            action={{
              type: "link",
              to:
                MANAGE_HOSTS +
                TAGGED_TEMPLATES.hostsByTeamRoute(currentTeam?.id),
              text: "View all hosts",
            }}
            total_host_count={totalCount}
          >
            <HostsSummary
              currentTeamId={currentTeam?.id}
              macCount={macCount}
              windowsCount={windowsCount}
            />
          </InfoCard>
        </div>
        <div className={`${baseClass}__section one-column`}>
          <InfoCard title="">
            <HostsStatus
              onlineCount={onlineCount}
              offlineCount={offlineCount}
              newCount={newCount}
            />
          </InfoCard>
        </div>
        {isPreviewMode && (
          <div className={`${baseClass}__section two-column`}>
            <InfoCard title="Welcome to Fleet">
              <WelcomeHost />
            </InfoCard>
            <InfoCard title="Learn how to use Fleet">
              <LearnFleet />
            </InfoCard>
          </div>
        )}
        <div
          className={`
          ${baseClass}__section 
          ${currentTeam ? "one" : "two"}-column
        `}
        >
          <InfoCard
            title="Software"
            action={{
              type: "button",
              text: "View all software",
              onClick: () => setIsSoftwareModalOpen(true),
            }}
          >
            <Software
              currentTeamId={currentTeam?.id}
              isModalOpen={isSoftwareModalOpen}
              setIsSoftwareModalOpen={setIsSoftwareModalOpen}
            />
          </InfoCard>
          {!isPreviewMode && !currentTeam && isOnGlobalTeam && (
            <InfoCard title="Activity">
              <ActivityFeed />
            </InfoCard>
          )}
        </div>
      </div>
    </div>
  );
};

export default Homepage;
