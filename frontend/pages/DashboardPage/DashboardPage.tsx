import React, { useContext, useState, useEffect } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";

import { AppContext } from "context/app";
import paths from "router/paths";

import {
  IEnrollSecret,
  IEnrollSecretsResponse,
} from "interfaces/enroll_secret";
import { IHostSummary, IHostSummaryPlatforms } from "interfaces/host_summary";
import { ILabelSummary } from "interfaces/label";
import {
  IMacadminAggregate,
  IMunkiIssuesAggregate,
  IMunkiVersionsAggregate,
} from "interfaces/macadmins";
import {
  IMdmStatusCardData,
  IMdmSolution,
  IMdmSummaryResponse,
} from "interfaces/mdm";
import { SelectedPlatform } from "interfaces/platform";
import { ISoftwareResponse, ISoftwareCountResponse } from "interfaces/software";
import { ITeam } from "interfaces/team";
import { useTeamIdParam } from "hooks/useTeamIdParam";
import enrollSecretsAPI from "services/entities/enroll_secret";
import hostSummaryAPI from "services/entities/host_summary";
import macadminsAPI from "services/entities/macadmins";
import softwareAPI, {
  ISoftwareQueryKey,
  ISoftwareCountQueryKey,
} from "services/entities/software";
import teamsAPI, { ILoadTeamsResponse } from "services/entities/teams";
import hosts from "services/entities/hosts";
import sortUtils from "utilities/sort";
import {
  PLATFORM_DROPDOWN_OPTIONS,
  PLATFORM_NAME_TO_LABEL_NAME,
} from "utilities/constants";
import { ITableQueryData } from "components/TableContainer/TableContainer";

import TeamsDropdown from "components/TeamsDropdown";
import Spinner from "components/Spinner";
import CustomLink from "components/CustomLink";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import MainContent from "components/MainContent";
import LastUpdatedText from "components/LastUpdatedText";
import SandboxGate from "components/Sandbox/SandboxGate";
import useInfoCard from "./components/InfoCard";
import MissingHosts from "./cards/MissingHosts";
import LowDiskSpaceHosts from "./cards/LowDiskSpaceHosts";
import HostsSummary from "./cards/HostsSummary";
import ActivityFeed from "./cards/ActivityFeed";
import Software from "./cards/Software";
import LearnFleet from "./cards/LearnFleet";
import WelcomeHost from "./cards/WelcomeHost";
import Mdm from "./cards/MDM";
import Munki from "./cards/Munki";
import OperatingSystems from "./cards/OperatingSystems";
import AddHostsModal from "../../components/AddHostsModal";

const baseClass = "dashboard-page";

// Premium feature, Gb must be set between 1-100
const LOW_DISK_SPACE_GB = 32;

interface IDashboardProps {
  router: InjectedRouter; // v3
  location: {
    pathname: string;
    search: string;
    hash?: string;
    query: {
      team_id?: string;
    };
  };
}

const DashboardPage = ({ router, location }: IDashboardProps): JSX.Element => {
  const { pathname } = location;
  const {
    config,
    isGlobalAdmin,
    isGlobalMaintainer,
    isPremiumTier,
    isSandboxMode,
    isOnGlobalTeam,
  } = useContext(AppContext);

  const {
    currentTeamId,
    currentTeamName,
    isAnyTeamSelected,
    isRouteOk,
    isTeamAdmin,
    isTeamMaintainer,
    teamIdForApi,
    userTeams,
    handleTeamChange,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: true,
    includeNoTeam: false,
  });

  const [selectedPlatform, setSelectedPlatform] = useState<SelectedPlatform>(
    "all"
  );
  const [
    selectedPlatformLabelId,
    setSelectedPlatformLabelId,
  ] = useState<number>();
  const [labels, setLabels] = useState<ILabelSummary[]>();
  const [macCount, setMacCount] = useState(0);
  const [windowsCount, setWindowsCount] = useState(0);
  const [linuxCount, setLinuxCount] = useState(0);
  const [chromeCount, setChromeCount] = useState(0);
  const [missingCount, setMissingCount] = useState(0);
  const [lowDiskSpaceCount, setLowDiskSpaceCount] = useState(0);
  const [showActivityFeedTitle, setShowActivityFeedTitle] = useState(false);
  const [softwareTitleDetail, setSoftwareTitleDetail] = useState<
    JSX.Element | string | null
  >("");
  const [softwareNavTabIndex, setSoftwareNavTabIndex] = useState(0);
  const [softwarePageIndex, setSoftwarePageIndex] = useState(0);
  const [softwareActionUrl, setSoftwareActionUrl] = useState<string>();
  const [showMunkiCard, setShowMunkiCard] = useState(true);
  const [showMdmCard, setShowMdmCard] = useState(true);
  const [showSoftwareCard, setShowSoftwareCard] = useState(false);
  const [showAddHostsModal, setShowAddHostsModal] = useState(false);
  const [showOperatingSystemsUI, setShowOperatingSystemsUI] = useState(false);
  const [showHostsUI, setShowHostsUI] = useState(false); // Hides UI on first load only
  const [mdmStatusData, setMdmStatusData] = useState<IMdmStatusCardData[]>([]);
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

  useEffect(() => {
    const platformByPathname =
      PLATFORM_DROPDOWN_OPTIONS?.find((platform) => platform.path === pathname)
        ?.value || "all";

    setSelectedPlatform(platformByPathname);
  }, [pathname]);

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
  });

  const {
    data: hostSummaryData,
    isFetching: isHostSummaryFetching,
    error: errorHosts,
  } = useQuery<IHostSummary, Error, IHostSummary>(
    ["host summary", teamIdForApi, isPremiumTier, selectedPlatform],
    () =>
      hostSummaryAPI.getSummary({
        teamId: teamIdForApi,
        platform: selectedPlatform !== "all" ? selectedPlatform : undefined,
        lowDiskSpace: isPremiumTier ? LOW_DISK_SPACE_GB : undefined,
      }),
    {
      enabled: isRouteOk,
      select: (data: IHostSummary) => data,
      onSuccess: (data: IHostSummary) => {
        setLabels(data.builtin_labels);
        if (isPremiumTier) {
          setMissingCount(data.missing_30_days_count || 0);
          setLowDiskSpaceCount(data.low_disk_space_count || 0);
        }
        const macHosts = data.platforms?.find(
          (platform: IHostSummaryPlatforms) => platform.platform === "darwin"
        ) || { platform: "darwin", hosts_count: 0 };

        const windowsHosts = data.platforms?.find(
          (platform: IHostSummaryPlatforms) => platform.platform === "windows"
        ) || { platform: "windows", hosts_count: 0 };

        const chromebooks = data.platforms?.find(
          (platform: IHostSummaryPlatforms) => platform.platform === "chrome"
        ) || { platform: "chrome", hosts_count: 0 };

        setMacCount(macHosts.hosts_count);
        setWindowsCount(windowsHosts.hosts_count);
        setLinuxCount(data.all_linux_count);
        setChromeCount(chromebooks.hosts_count);
        setShowHostsUI(true);
      },
    }
  );

  const { isLoading: isGlobalSecretsLoading, data: globalSecrets } = useQuery<
    IEnrollSecretsResponse,
    Error,
    IEnrollSecret[]
  >(["global secrets"], () => enrollSecretsAPI.getGlobalEnrollSecrets(), {
    enabled: isRouteOk && canEnrollGlobalHosts,
    select: (data: IEnrollSecretsResponse) => data.secrets,
  });

  const { data: teamSecrets } = useQuery<
    IEnrollSecretsResponse,
    Error,
    IEnrollSecret[]
  >(
    ["team secrets", teamIdForApi],
    () => {
      if (isAnyTeamSelected) {
        return enrollSecretsAPI.getTeamEnrollSecrets(teamIdForApi);
      }
      return { secrets: [] };
    },
    {
      enabled: isRouteOk && isAnyTeamSelected && canEnrollHosts,
      select: (data: IEnrollSecretsResponse) => data.secrets,
    }
  );

  const featuresConfig = isAnyTeamSelected
    ? teams?.find((t) => t.id === currentTeamId)?.features
    : config?.features;
  const isSoftwareEnabled = !!featuresConfig?.enable_software_inventory;

  const SOFTWARE_DEFAULT_SORT_DIRECTION = "desc";
  const SOFTWARE_DEFAULT_SORT_HEADER = "hosts_count";
  const SOFTWARE_DEFAULT_PAGE_SIZE = 8;

  const {
    data: software,
    isFetching: isSoftwareFetching,
    error: errorSoftware,
  } = useQuery<
    ISoftwareResponse,
    Error,
    ISoftwareResponse,
    ISoftwareQueryKey[]
  >(
    [
      {
        scope: "software",
        page: softwarePageIndex,
        perPage: SOFTWARE_DEFAULT_PAGE_SIZE,
        orderDirection: SOFTWARE_DEFAULT_SORT_DIRECTION,
        orderKey: SOFTWARE_DEFAULT_SORT_HEADER,
        teamId: teamIdForApi,
        vulnerable: !!softwareNavTabIndex, // we can take the tab index as a boolean to represent the vulnerable flag :)
      },
    ],
    ({ queryKey }) => softwareAPI.load(queryKey[0]),
    {
      enabled: isRouteOk && isSoftwareEnabled,
      keepPreviousData: true,
      staleTime: 30000, // stale time can be adjusted if fresher data is desired based on software inventory interval
      onSuccess: (data) => {
        if (data.software?.length > 0) {
          setSoftwareTitleDetail &&
            setSoftwareTitleDetail(
              <LastUpdatedText
                lastUpdatedAt={data.counts_updated_at}
                whatToRetrieve="software"
              />
            );
          setShowSoftwareCard(true);
        } else {
          setShowSoftwareCard(false);
        }
      },
    }
  );

  // If no vulnerable software, !software?.software can return undefined
  // Must check non-vuln software count > 0 to show software card iff API returning undefined
  const { data: softwareCount } = useQuery<
    ISoftwareCountResponse,
    Error,
    number,
    ISoftwareCountQueryKey[]
  >(
    [
      {
        scope: "softwareCount",
        teamId: teamIdForApi,
      },
    ],
    ({ queryKey }) => softwareAPI.getCount(queryKey[0]),
    {
      enabled: isRouteOk && !software?.software,
      keepPreviousData: true,
      refetchOnWindowFocus: false,
      retry: 1,
      select: (data) => data.count,
    }
  );

  const { isFetching: isMdmFetching, error: errorMdm } = useQuery<
    IMdmSummaryResponse,
    Error
  >(
    [`mdm-${selectedPlatform}`, teamIdForApi],
    () => hosts.getMdmSummary(selectedPlatform, teamIdForApi),
    {
      enabled: isRouteOk && !["linux", "chrome"].includes(selectedPlatform),
      onSuccess: ({
        counts_updated_at,
        mobile_device_management_solution,
        mobile_device_management_enrollment_status: {
          enrolled_automated_hosts_count,
          enrolled_manual_hosts_count,
          unenrolled_hosts_count,
          pending_hosts_count,
          hosts_count,
        },
      }) => {
        if (hosts_count === 0 && mobile_device_management_solution === null) {
          setShowMdmCard(false);
          return;
        }

        setMdmTitleDetail(
          <LastUpdatedText
            lastUpdatedAt={counts_updated_at}
            whatToRetrieve="MDM information"
          />
        );
        const statusData: IMdmStatusCardData[] = [
          {
            status: "On (manual)",
            hosts: enrolled_manual_hosts_count,
          },
          {
            status: "On (automatic)",
            hosts: enrolled_automated_hosts_count,
          },
          { status: "Off", hosts: unenrolled_hosts_count },
        ];
        isPremiumTier &&
          statusData.push({
            status: "Pending",
            hosts: pending_hosts_count || 0,
          });
        setMdmStatusData(statusData);
        setMdmSolutions(mobile_device_management_solution);
        setShowMdmCard(true);
      },
    }
  );

  const { isFetching: isMacAdminsFetching, error: errorMacAdmins } = useQuery<
    IMacadminAggregate,
    Error
  >(["macAdmins", teamIdForApi], () => macadminsAPI.loadAll(teamIdForApi), {
    keepPreviousData: true,
    enabled: isRouteOk && selectedPlatform === "darwin",
    onSuccess: ({
      macadmins: { munki_issues, munki_versions, counts_updated_at },
    }) => {
      setMunkiVersionsData(munki_versions);
      setMunkiIssuesData(munki_issues);
      setShowMunkiCard(!!munki_versions);
      setMunkiTitleDetail(
        <LastUpdatedText
          lastUpdatedAt={counts_updated_at}
          whatToRetrieve="Munki"
        />
      );
    },
  });

  useEffect(() => {
    softwareCount && softwareCount > 0
      ? setShowSoftwareCard(true)
      : setShowSoftwareCard(false);
  }, [softwareCount]);

  // Sets selected platform label id for links to filtered manage host page
  useEffect(() => {
    if (labels) {
      const getLabel = (
        labelString: string,
        summaryLabels: ILabelSummary[]
      ): ILabelSummary | undefined => {
        return Object.values(summaryLabels).find((label: ILabelSummary) => {
          return label.label_type === "builtin" && label.name === labelString;
        });
      };

      if (selectedPlatform !== "all") {
        const labelValue = PLATFORM_NAME_TO_LABEL_NAME[selectedPlatform];
        setSelectedPlatformLabelId(getLabel(labelValue, labels)?.id);
      } else {
        setSelectedPlatformLabelId(undefined);
      }
    }
  }, [labels, selectedPlatform]);

  const toggleAddHostsModal = () => {
    setShowAddHostsModal(!showAddHostsModal);
  };

  const { MANAGE_HOSTS } = paths;

  const HostsSummaryCard = useInfoCard({
    title: "Hosts",
    action:
      selectedPlatform === "all"
        ? {
            type: "link",
            text: "View all hosts",
          }
        : undefined,
    actionUrl: selectedPlatform === "all" ? MANAGE_HOSTS : undefined,
    total_host_count:
      !isHostSummaryFetching && !errorHosts
        ? hostSummaryData?.totals_hosts_count
        : undefined,
    showTitle: true,
    children: (
      <HostsSummary
        currentTeamId={teamIdForApi}
        macCount={macCount}
        windowsCount={windowsCount}
        linuxCount={linuxCount}
        chromeCount={chromeCount}
        isLoadingHostsSummary={isHostSummaryFetching}
        builtInLabels={labels}
        showHostsUI={showHostsUI}
        selectedPlatform={selectedPlatform}
        errorHosts={!!errorHosts}
      />
    ),
  });

  // NOTE: this is called once on the initial rendering. The initial render of
  // the TableContainer child component will call this handler.
  const onSoftwareQueryChange = async ({
    pageIndex: newPageIndex,
  }: ITableQueryData) => {
    if (softwarePageIndex !== newPageIndex) {
      setSoftwarePageIndex(newPageIndex);
    }
  };

  const onSoftwareTabChange = (index: number) => {
    const { SOFTWARE_TITLES } = paths;
    setSoftwareNavTabIndex(index);
    setSoftwareActionUrl &&
      setSoftwareActionUrl(
        index === 1 ? `${SOFTWARE_TITLES}?vulnerable=true` : SOFTWARE_TITLES
      );
  };

  // TODO: Rework after backend is adjusted to differentiate empty search/filter results from
  // collecting inventory
  const isCollectingInventory =
    !isAnyTeamSelected &&
    !softwarePageIndex &&
    !software?.software &&
    software?.counts_updated_at === null;

  const MissingHostsCard = useInfoCard({
    title: "",
    children: (
      <MissingHosts
        missingCount={missingCount}
        isLoadingHosts={isHostSummaryFetching}
        showHostsUI={showHostsUI}
        selectedPlatformLabelId={selectedPlatformLabelId}
        currentTeamId={teamIdForApi}
        isSandboxMode={isSandboxMode}
      />
    ),
  });

  const LowDiskSpaceHostsCard = useInfoCard({
    title: "",
    children: (
      <LowDiskSpaceHosts
        lowDiskSpaceGb={LOW_DISK_SPACE_GB}
        lowDiskSpaceCount={lowDiskSpaceCount}
        isLoadingHosts={isHostSummaryFetching}
        showHostsUI={showHostsUI}
        selectedPlatformLabelId={selectedPlatformLabelId}
        currentTeamId={teamIdForApi}
        isSandboxMode={isSandboxMode}
        notSupported={selectedPlatform === "chrome"}
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
      <ActivityFeed
        setShowActivityFeedTitle={setShowActivityFeedTitle}
        isPremiumTier={isPremiumTier || false}
        isSandboxMode={isSandboxMode}
      />
    ),
  });

  const SoftwareCard = useInfoCard({
    title: "Software",
    action: {
      type: "link",
      text: "View all software",
      to: "software",
    },
    actionUrl: softwareActionUrl,
    titleDetail: softwareTitleDetail,
    showTitle: !isSoftwareFetching,
    children: (
      <Software
        errorSoftware={errorSoftware}
        isCollectingInventory={isCollectingInventory}
        isSoftwareFetching={isSoftwareFetching}
        isSoftwareEnabled={isSoftwareEnabled}
        software={software}
        teamId={currentTeamId}
        pageIndex={softwarePageIndex}
        navTabIndex={softwareNavTabIndex}
        onTabChange={onSoftwareTabChange}
        onQueryChange={onSoftwareQueryChange}
        router={router}
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
        <CustomLink
          url="https://www.munki.org/munki/"
          text="Learn about Munki"
          newTab
        />
      </p>
    ),
    children: (
      <Munki
        errorMacAdmins={errorMacAdmins}
        isMacAdminsFetching={isMacAdminsFetching}
        munkiIssuesData={munkiIssuesData}
        munkiVersionsData={munkiVersionsData}
        selectedTeamId={currentTeamId}
      />
    ),
  });

  const MDMCard = (
    <SandboxGate>
      {useInfoCard({
        title: "Mobile device management (MDM)",
        titleDetail: mdmTitleDetail,
        showTitle: !isMdmFetching,
        description: (
          <p>
            MDM is used to change settings and install software on your hosts.
          </p>
        ),
        children: (
          <Mdm
            isFetching={isMdmFetching}
            error={errorMdm}
            mdmStatusData={mdmStatusData}
            mdmSolutions={mdmSolutions}
            selectedPlatformLabelId={selectedPlatformLabelId}
            selectedTeamId={currentTeamId}
          />
        ),
      })}
    </SandboxGate>
  );

  const OperatingSystemsCard = useInfoCard({
    title: "Operating systems",
    showTitle: showOperatingSystemsUI,
    children: (
      <OperatingSystems
        currentTeamId={teamIdForApi}
        selectedPlatform={selectedPlatform}
        showTitle={showOperatingSystemsUI}
        setShowTitle={setShowOperatingSystemsUI}
      />
    ),
  });

  const allLayout = () => {
    return (
      <div className={`${baseClass}__section`}>
        {!isAnyTeamSelected &&
          canEnrollGlobalHosts &&
          hostSummaryData &&
          hostSummaryData?.totals_hosts_count < 2 && (
            <>
              {WelcomeHostCard}
              {LearnFleetCard}
            </>
          )}
        {showSoftwareCard && SoftwareCard}
        {!isAnyTeamSelected && isOnGlobalTeam && <>{ActivityFeedCard}</>}
        {showMdmCard && <>{MDMCard}</>}
      </div>
    );
  };

  const macOSLayout = () => (
    <>
      <div className={`${baseClass}__section`}>{OperatingSystemsCard}</div>
      {showMdmCard && <div className={`${baseClass}__section`}>{MDMCard}</div>}
      {showMunkiCard && (
        <div className={`${baseClass}__section`}>{MunkiCard}</div>
      )}
    </>
  );

  const windowsLayout = () => (
    <>
      <div className={`${baseClass}__section`}>{OperatingSystemsCard}</div>
      {showMdmCard && <div className={`${baseClass}__section`}>{MDMCard}</div>}
    </>
  );
  const linuxLayout = () => null;

  const chromeLayout = () => (
    <>
      <div className={`${baseClass}__section`}>{OperatingSystemsCard}</div>
    </>
  );

  const renderCards = () => {
    switch (selectedPlatform) {
      case "darwin":
        return macOSLayout();
      case "windows":
        return windowsLayout();
      case "linux":
        return linuxLayout();
      case "chrome":
        return chromeLayout();
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
      isAnyTeamSelected && !isSandboxMode
        ? teamSecrets?.[0].secret
        : globalSecrets?.[0].secret;

    return (
      <AddHostsModal
        currentTeamName={currentTeamName}
        enrollSecret={enrollSecret}
        isAnyTeamSelected={isAnyTeamSelected}
        isLoading={isLoadingTeams || isGlobalSecretsLoading}
        isSandboxMode={!!isSandboxMode}
        onCancel={toggleAddHostsModal}
      />
    );
  };

  const renderDashboardHeader = () => {
    if (isPremiumTier) {
      if (userTeams) {
        if (userTeams.length > 1 || isOnGlobalTeam) {
          return (
            <TeamsDropdown
              selectedTeamId={currentTeamId}
              currentUserTeams={userTeams}
              onChange={handleTeamChange}
              isSandboxMode={isSandboxMode}
            />
          );
        }
        if (userTeams.length === 1) {
          return <h1>{userTeams[0].name}</h1>;
        }
      }
      // userTeams.length should have at least 1 element
      return null;
    }
    // Free tier
    return <h1>{config?.org_info.org_name}</h1>;
  };
  return !isRouteOk ? (
    <Spinner />
  ) : (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__header`}>
          <div className={`${baseClass}__text`}>
            <div className={`${baseClass}__title`}>
              {renderDashboardHeader()}
            </div>
          </div>
        </div>
        <div className={`${baseClass}__platforms`}>
          <span>Platform:&nbsp;</span>
          <Dropdown
            value={selectedPlatform || ""}
            className={`${baseClass}__platform_dropdown`}
            options={PLATFORM_DROPDOWN_OPTIONS}
            searchable={false}
            onChange={(value: SelectedPlatform) => {
              const selectedPlatformOption = PLATFORM_DROPDOWN_OPTIONS.find(
                (platform) => platform.value === value
              );
              router.push(
                (selectedPlatformOption?.path || paths.DASHBOARD)
                  .concat(location.search)
                  .concat(location.hash || "")
              );
            }}
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
            {isPremiumTier && (
              <div className={`${baseClass}__section`}>
                {MissingHostsCard}
                {LowDiskSpaceHostsCard}
              </div>
            )}
          </>
        </div>
        {renderCards()}
        {showAddHostsModal && renderAddHostsModal()}
      </div>
    </MainContent>
  );
};

export default DashboardPage;
