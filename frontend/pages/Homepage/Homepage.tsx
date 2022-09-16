import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { AppContext } from "context/app";
import { find } from "lodash";

import {
  IEnrollSecret,
  IEnrollSecretsResponse,
} from "interfaces/enroll_secret";
import { IHostSummary, IHostSummaryPlatforms } from "interfaces/host_summary";
import { ILabelSummary } from "interfaces/label";
import {
  IDataTableMdmFormat,
  IMdmSolution,
  IMacadminAggregate,
  IMunkiIssuesAggregate,
  IMunkiVersionsAggregate,
} from "interfaces/macadmins";
import { IOsqueryPlatform } from "interfaces/platform";
import { ITeam } from "interfaces/team";
import enrollSecretsAPI from "services/entities/enroll_secret";
import hostSummaryAPI from "services/entities/host_summary";
import macadminsAPI from "services/entities/macadmins";
import teamsAPI, { ILoadTeamsResponse } from "services/entities/teams";
import sortUtils from "utilities/sort";
import { PLATFORM_DROPDOWN_OPTIONS } from "utilities/constants";

import TeamsDropdown from "components/TeamsDropdown";
import Spinner from "components/Spinner";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import MainContent from "components/MainContent";
import LastUpdatedText from "components/LastUpdatedText";
import useInfoCard from "./components/InfoCard";
import HostsStatus from "./cards/HostsStatus";
import HostsSummary from "./cards/HostsSummary";
import ActivityFeed from "./cards/ActivityFeed";
import Software from "./cards/Software";
import LearnFleet from "./cards/LearnFleet";
import WelcomeHost from "./cards/WelcomeHost";
import Mdm from "./cards/MDM";
import Munki from "./cards/Munki";
import OperatingSystems from "./cards/OperatingSystems";
import AddHostsModal from "../../components/AddHostsModal";
import ExternalLinkIcon from "../../../assets/images/icon-external-link-12x12@2x.png";

const baseClass = "homepage";

const Homepage = (): JSX.Element => {
  const {
    config,
    currentTeam,
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
    isPremiumTier,
    isFreeTier,
    isSandboxMode,
    isOnGlobalTeam,
    setCurrentTeam,
  } = useContext(AppContext);

  const [selectedPlatform, setSelectedPlatform] = useState("");
  const [labels, setLabels] = useState<ILabelSummary[]>();
  const [macCount, setMacCount] = useState(0);
  const [windowsCount, setWindowsCount] = useState(0);
  const [linuxCount, setLinuxCount] = useState(0);
  const [onlineCount, setOnlineCount] = useState(0);
  const [offlineCount, setOfflineCount] = useState(0);
  const [showActivityFeedTitle, setShowActivityFeedTitle] = useState(false);
  const [showSoftwareUI, setShowSoftwareUI] = useState(false);
  const [showMunkiCard, setShowMunkiCard] = useState(true);
  const [showAddHostsModal, setShowAddHostsModal] = useState(false);
  const [showOperatingSystemsUI, setShowOperatingSystemsUI] = useState(false);
  const [showHostsUI, setShowHostsUI] = useState(false); // Hides UI on first load only
  const [formattedMdmData, setFormattedMdmData] = useState<
    IDataTableMdmFormat[]
  >([]);
  const [mdmSolutions, setMdmSolutions] = useState<IMdmSolution[] | null>([]);

  const [munkiIssuesData, setMunkiIssuesData] = useState<
    IMunkiIssuesAggregate[]
  >([]);
  const [munkiVersionsData, setMunkiVersionsData] = useState<
    IMunkiVersionsAggregate[]
  >([]);
  const [mdmTitleDetail, setMdmTitleDetail] = useState<
    JSX.Element | string | null
  >();
  const [munkiTitleDetail, setMunkiTitleDetail] = useState<
    JSX.Element | string | null
  >();

  const canEnrollHosts =
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer;
  const canEnrollGlobalHosts = isGlobalAdmin || isGlobalMaintainer;

  const { data: teams, isLoading: isLoadingTeams } = useQuery<
    ILoadTeamsResponse,
    Error,
    ITeam[]
  >(["teams"], () => teamsAPI.loadAll(), {
    enabled: !!isPremiumTier,
    select: (data: ILoadTeamsResponse) =>
      data.teams.sort((a, b) => sortUtils.caseInsensitiveAsc(a.name, b.name)),
    onSuccess: (responseTeams) => {
      if (!currentTeam && !isOnGlobalTeam && responseTeams.length) {
        setCurrentTeam(responseTeams[0]);
      }
    },
  });

  const {
    data: hostSummaryData,
    isFetching: isHostSummaryFetching,
    error: errorHosts,
  } = useQuery<IHostSummary, Error, IHostSummary>(
    ["host summary", currentTeam, selectedPlatform],
    () =>
      hostSummaryAPI.getSummary({
        teamId: currentTeam?.id,
        platform: selectedPlatform,
      }),
    {
      select: (data: IHostSummary) => data,
      onSuccess: (data: IHostSummary) => {
        setLabels(data.builtin_labels);
        setOnlineCount(data.online_count);
        setOfflineCount(data.offline_count);

        const macHosts = data.platforms?.find(
          (platform: IHostSummaryPlatforms) => platform.platform === "darwin"
        ) || { platform: "darwin", hosts_count: 0 };

        const windowsHosts = data.platforms?.find(
          (platform: IHostSummaryPlatforms) => platform.platform === "windows"
        ) || { platform: "windows", hosts_count: 0 };

        setMacCount(macHosts.hosts_count);
        setWindowsCount(windowsHosts.hosts_count);
        setLinuxCount(data.all_linux_count);
        setShowHostsUI(true);
      },
    }
  );

  const { isLoading: isGlobalSecretsLoading, data: globalSecrets } = useQuery<
    IEnrollSecretsResponse,
    Error,
    IEnrollSecret[]
  >(["global secrets"], () => enrollSecretsAPI.getGlobalEnrollSecrets(), {
    enabled: !!canEnrollGlobalHosts,
    select: (data: IEnrollSecretsResponse) => data.secrets,
  });

  const { data: teamSecrets } = useQuery<
    IEnrollSecretsResponse,
    Error,
    IEnrollSecret[]
  >(
    ["team secrets", currentTeam],
    () => {
      if (currentTeam) {
        return enrollSecretsAPI.getTeamEnrollSecrets(currentTeam.id);
      }
      return { secrets: [] };
    },
    {
      enabled: !!currentTeam?.id && !!canEnrollHosts,
      select: (data: IEnrollSecretsResponse) => data.secrets,
    }
  );

  const { isFetching: isMacAdminsFetching, error: errorMacAdmins } = useQuery<
    IMacadminAggregate,
    Error
  >(
    ["macAdmins", currentTeam?.id],
    () => macadminsAPI.loadAll(currentTeam?.id),
    {
      keepPreviousData: true,
      enabled: selectedPlatform === "darwin",
      onSuccess: (data) => {
        const {
          counts_updated_at: macadmins_counts_updated_at,
          mobile_device_management_enrollment_status,
          mobile_device_management_solution,
        } = data.macadmins;
        const {
          enrolled_manual_hosts_count,
          enrolled_automated_hosts_count,
          unenrolled_hosts_count,
        } = mobile_device_management_enrollment_status;

        const {
          counts_updated_at: munki_counts_updated_at,
          munki_versions,
          munki_issues,
        } = data.macadmins;

        setMdmTitleDetail(
          <LastUpdatedText
            lastUpdatedAt={macadmins_counts_updated_at}
            whatToRetrieve={"MDM enrollment"}
          />
        );
        setFormattedMdmData([
          {
            status: "Enrolled (manual)",
            hosts: enrolled_manual_hosts_count,
          },
          {
            status: "Enrolled (automatic)",
            hosts: enrolled_automated_hosts_count,
          },
          { status: "Unenrolled", hosts: unenrolled_hosts_count },
        ]);
        setMdmSolutions(mobile_device_management_solution);
        setMunkiVersionsData(munki_versions);
        setMunkiIssuesData(munki_issues);
        setShowMunkiCard(!!munki_versions);
        setMunkiTitleDetail(
          <LastUpdatedText
            lastUpdatedAt={munki_counts_updated_at}
            whatToRetrieve={"Munki"}
          />
        );
      },
    }
  );

  const handleTeamSelect = (teamId: number) => {
    const selectedTeam = find(teams, ["id", teamId]);
    setCurrentTeam(selectedTeam);
  };

  const toggleAddHostsModal = () => {
    setShowAddHostsModal(!showAddHostsModal);
  };

  const HostsSummaryCard = useInfoCard({
    title: "Hosts",
    action: {
      type: "link",
      text: "View all hosts",
    },
    total_host_count: (() => {
      if (!isHostSummaryFetching && !errorHosts) {
        return `${hostSummaryData?.totals_hosts_count}` || undefined;
      }

      return undefined;
    })(),
    showTitle: true,
    children: (
      <HostsSummary
        currentTeamId={currentTeam?.id}
        macCount={macCount}
        windowsCount={windowsCount}
        linuxCount={linuxCount}
        isLoadingHostsSummary={isHostSummaryFetching}
        showHostsUI={showHostsUI}
        selectedPlatform={selectedPlatform}
        labels={labels}
        errorHosts={!!errorHosts}
      />
    ),
  });

  const HostsStatusCard = useInfoCard({
    title: "",
    children: (
      <HostsStatus
        onlineCount={onlineCount}
        offlineCount={offlineCount}
        isLoadingHosts={isHostSummaryFetching}
        showHostsUI={showHostsUI}
      />
    ),
  });

  const WelcomeHostCard = useInfoCard({
    title: "Welcome to Fleet",
    showTitle: true,
    children: (
      <WelcomeHost
        totalsHostsCount={
          (hostSummaryData && hostSummaryData.totals_hosts_count) || 0
        }
        toggleAddHostsModal={toggleAddHostsModal}
      />
    ),
  });

  const LearnFleetCard = useInfoCard({
    title: "Learn how to use Fleet",
    showTitle: true,
    children: <LearnFleet />,
  });

  const ActivityFeedCard = useInfoCard({
    title: "Activity",
    showTitle: showActivityFeedTitle,
    children: (
      <ActivityFeed setShowActivityFeedTitle={setShowActivityFeedTitle} />
    ),
  });

  const SoftwareCard = useInfoCard({
    title: "Software",
    action: {
      type: "link",
      text: "View all software",
      to: "software",
    },
    showTitle: showSoftwareUI,
    children: (
      <Software
        currentTeamId={currentTeam?.id}
        setShowSoftwareUI={setShowSoftwareUI}
        showSoftwareUI={showSoftwareUI}
      />
    ),
  });

  const MunkiCard = useInfoCard({
    title: "Munki",
    titleDetail: munkiTitleDetail,
    showTitle: !isMacAdminsFetching,
    description: (
      <p>
        Munki is a tool for managing software on macOS devices.{" "}
        <a
          href="https://www.munki.org/munki/"
          target="_blank"
          rel="noopener noreferrer"
        >
          Learn about Munki
          <img src={ExternalLinkIcon} alt="Open external link" />
        </a>
      </p>
    ),
    children: (
      <Munki
        errorMacAdmins={errorMacAdmins}
        isMacAdminsFetching={isMacAdminsFetching}
        munkiIssuesData={munkiIssuesData}
        munkiVersionsData={munkiVersionsData}
      />
    ),
  });

  const MDMCard = useInfoCard({
    title: "Mobile device management (MDM)",
    titleDetail: mdmTitleDetail,
    showTitle: !isMacAdminsFetching,
    description: (
      <p>
        MDM is used to manage configuration on macOS devices.{" "}
        <a
          href="https://support.apple.com/guide/deployment/intro-to-mdm-depc0aadd3fe/web"
          target="_blank"
          rel="noopener noreferrer"
        >
          Learn about MDM
          <img src={ExternalLinkIcon} alt="Open external link" />
        </a>
      </p>
    ),
    children: (
      <Mdm
        isMacAdminsFetching={isMacAdminsFetching}
        errorMacAdmins={errorMacAdmins}
        formattedMdmData={formattedMdmData}
        mdmSolutions={mdmSolutions}
      />
    ),
  });

  const OperatingSystemsCard = useInfoCard({
    title: "Operating systems",
    showTitle: showOperatingSystemsUI,
    children: (
      <OperatingSystems
        currentTeamId={currentTeam?.id}
        selectedPlatform={selectedPlatform as IOsqueryPlatform}
        showTitle={showOperatingSystemsUI}
        setShowTitle={setShowOperatingSystemsUI}
      />
    ),
  });

  const allLayout = () => {
    return (
      <div className={`${baseClass}__section`}>
        {!currentTeam &&
          canEnrollGlobalHosts &&
          hostSummaryData &&
          hostSummaryData?.totals_hosts_count < 2 && (
            <>
              {WelcomeHostCard}
              {LearnFleetCard}
            </>
          )}
        {SoftwareCard}
        {!currentTeam && isOnGlobalTeam && <>{ActivityFeedCard}</>}
      </div>
    );
  };

  const macOSLayout = () => (
    <>
      <div className={`${baseClass}__section`}>{OperatingSystemsCard}</div>
      <div className={`${baseClass}__section`}>{MDMCard}</div>
      {showMunkiCard && (
        <div className={`${baseClass}__section`}>{MunkiCard}</div>
      )}
    </>
  );

  const windowsLayout = () => (
    <div className={`${baseClass}__section`}>{OperatingSystemsCard}</div>
  );
  const linuxLayout = () => null;

  const renderCards = () => {
    switch (selectedPlatform) {
      case "darwin":
        return macOSLayout();
      case "windows":
        return windowsLayout();
      case "linux":
        return linuxLayout();
      default:
        return allLayout();
    }
  };

  const renderAddHostsModal = () => {
    const enrollSecret =
      // TODO: Currently, prepacked installers in Fleet Sandbox use the global enroll secret,
      // and Fleet Sandbox runs Fleet Free so the isSandboxMode check here is an
      // additional precaution/reminder to revisit this in connection with future changes.
      // See https://github.com/fleetdm/fleet/issues/4970#issuecomment-1187679407.
      currentTeam && !isSandboxMode
        ? teamSecrets?.[0].secret
        : globalSecrets?.[0].secret;

    return (
      <AddHostsModal
        currentTeam={currentTeam}
        enrollSecret={enrollSecret}
        isLoading={isLoadingTeams || isGlobalSecretsLoading}
        isSandboxMode={!!isSandboxMode}
        onCancel={toggleAddHostsModal}
      />
    );
  };

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__header`}>
          <div className={`${baseClass}__text`}>
            <div className={`${baseClass}__title`}>
              {isFreeTier && <h1>{config?.org_info.org_name}</h1>}
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
        <div className={`${baseClass}__platforms`}>
          <span>Platform:&nbsp;</span>
          <Dropdown
            value={selectedPlatform}
            className={`${baseClass}__platform_dropdown`}
            options={PLATFORM_DROPDOWN_OPTIONS}
            searchable={false}
            onChange={(value: string) => setSelectedPlatform(value)}
          />
        </div>
        <div className="host-sections">
          <>
            {isHostSummaryFetching && (
              <div className="spinner">
                <Spinner />
              </div>
            )}
            <div className={`${baseClass}__section`}>{HostsSummaryCard}</div>
            <div className={`${baseClass}__section`}>{HostsStatusCard}</div>
          </>
        </div>
        {renderCards()}
        {showAddHostsModal && renderAddHostsModal()}
      </div>
    </MainContent>
  );
};

export default Homepage;
