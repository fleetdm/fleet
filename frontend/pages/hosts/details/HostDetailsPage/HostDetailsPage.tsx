import React, { useContext, useState, useCallback, useEffect } from "react";
import { Params, InjectedRouter } from "react-router/lib/Router";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import { RouteProps } from "react-router";
import { pick } from "lodash";

import PATHS from "router/paths";

import { AppContext } from "context/app";
import { QueryContext } from "context/query";
import { NotificationContext } from "context/notification";

import activitiesAPI, {
  IPastActivitiesResponse,
  IUpcomingActivitiesResponse,
} from "services/entities/activities";
import hostAPI from "services/entities/hosts";
import queryAPI from "services/entities/queries";
import teamAPI, { ILoadTeamsResponse } from "services/entities/teams";

import {
  IHost,
  IDeviceMappingResponse,
  IMacadminsResponse,
  IHostResponse,
  IHostMdmData,
  IPackStats,
} from "interfaces/host";
import { ILabel } from "interfaces/label";
import { IHostPolicy } from "interfaces/policy";
import { IQueryStats } from "interfaces/query_stats";
import { ISoftware } from "interfaces/software";
import { DEFAULT_TARGETS_BY_TYPE } from "interfaces/target";
import { ITeam } from "interfaces/team";
import {
  IListQueriesResponse,
  IQueryKeyQueriesLoadAll,
  ISchedulableQuery,
} from "interfaces/schedulable_query";

import {
  normalizeEmptyValues,
  wrapFleetHelper,
  TAGGED_TEMPLATES,
} from "utilities/helpers";
import permissions from "utilities/permissions";
import {
  DOCUMENT_TITLE_SUFFIX,
  HOST_SUMMARY_DATA,
  HOST_ABOUT_DATA,
  HOST_OSQUERY_DATA,
} from "utilities/constants";

import Spinner from "components/Spinner";
import TabsWrapper from "components/TabsWrapper";
import MainContent from "components/MainContent";
import BackLink from "components/BackLink";
import ScriptDetailsModal from "pages/DashboardPage/cards/ActivityFeed/components/ScriptDetailsModal";

import HostSummaryCard from "../cards/HostSummary";
import AboutCard from "../cards/About";
import ActivityCard from "../cards/Activity";
import AgentOptionsCard from "../cards/AgentOptions";
import LabelsCard from "../cards/Labels";
import MunkiIssuesCard from "../cards/MunkiIssues";
import SoftwareCard from "../cards/Software";
import UsersCard from "../cards/Users";
import PoliciesCard from "../cards/Policies";
import QueriesCard from "../cards/Queries";
import PacksCard from "../cards/Packs";
import PolicyDetailsModal from "../cards/Policies/HostPoliciesTable/PolicyDetailsModal";
import UnenrollMdmModal from "./modals/UnenrollMdmModal";
import TransferHostModal from "../../components/TransferHostModal";
import DeleteHostModal from "../../components/DeleteHostModal";

import DiskEncryptionKeyModal from "./modals/DiskEncryptionKeyModal";
import HostActionDropdown from "./HostActionsDropdown/HostActionsDropdown";
import OSSettingsModal from "../OSSettingsModal";
import BootstrapPackageModal from "./modals/BootstrapPackageModal";
import RunScriptModal from "./modals/RunScriptModal";
import SelectQueryModal from "./modals/SelectQueryModal";
import { isSupportedPlatform } from "./modals/DiskEncryptionKeyModal/DiskEncryptionKeyModal";
import HostDetailsBanners from "./components/HostDetailsBanners";
import { IShowActivityDetailsData } from "../cards/Activity/Activity";
import LockModal from "./modals/LockModal";
import UnlockModal from "./modals/UnlockModal";
import {
  HostMdmDeviceStatusUIState,
  getHostDeviceStatusUIState,
} from "../helpers";
import WipeModal from "./modals/WipeModal";

const baseClass = "host-details";

interface IHostDetailsProps {
  route: RouteProps;
  router: InjectedRouter; // v3
  location: {
    pathname: string;
    query: {
      vulnerable?: string;
      page?: string;
      query?: string;
      order_key?: string;
      order_direction?: "asc" | "desc";
    };
    search?: string;
  };
  params: Params;
}

interface ISearchQueryData {
  searchQuery: string;
  sortHeader: string;
  sortDirection: string;
  pageSize: number;
  pageIndex: number;
}

interface IHostDetailsSubNavItem {
  name: string | JSX.Element;
  title: string;
  pathname: string;
}

const DEFAULT_ACTIVITY_PAGE_SIZE = 8;

const HostDetailsPage = ({
  route,
  router,
  location,
  params: { host_id },
}: IHostDetailsProps): JSX.Element => {
  const hostIdFromURL = parseInt(host_id, 10);
  const routeTemplate = route?.path ?? "";
  const queryParams = location.query;

  const {
    config,
    currentUser,
    isGlobalAdmin = false,
    isGlobalObserver,
    isPremiumTier = false,
    isSandboxMode,
    isOnlyObserver,
    filteredHostsPath,
    currentTeam,
  } = useContext(AppContext);
  const { setSelectedQueryTargetsByType } = useContext(QueryContext);
  const { renderFlash } = useContext(NotificationContext);

  const handlePageError = useErrorHandler();

  const [showDeleteHostModal, setShowDeleteHostModal] = useState(false);
  const [showTransferHostModal, setShowTransferHostModal] = useState(false);
  const [showSelectQueryModal, setShowSelectQueryModal] = useState(false);
  const [showRunScriptModal, setShowRunScriptModal] = useState(false);
  const [showPolicyDetailsModal, setPolicyDetailsModal] = useState(false);
  const [showOSSettingsModal, setShowOSSettingsModal] = useState(false);
  const [showUnenrollMdmModal, setShowUnenrollMdmModal] = useState(false);
  const [showDiskEncryptionModal, setShowDiskEncryptionModal] = useState(false);
  const [showBootstrapPackageModal, setShowBootstrapPackageModal] = useState(
    false
  );
  const [showLockHostModal, setShowLockHostModal] = useState(false);
  const [showUnlockHostModal, setShowUnlockHostModal] = useState(false);
  const [showWipeModal, setShowWipeModal] = useState(false);
  const [scriptDetailsId, setScriptDetailsId] = useState("");
  const [selectedPolicy, setSelectedPolicy] = useState<IHostPolicy | null>(
    null
  );
  const [isUpdatingHost, setIsUpdatingHost] = useState(false);
  const [refetchStartTime, setRefetchStartTime] = useState<number | null>(null);
  const [showRefetchSpinner, setShowRefetchSpinner] = useState(false);
  const [schedule, setSchedule] = useState<IQueryStats[]>();
  const [packsState, setPackState] = useState<IPackStats[]>();
  const [hostSoftware, setHostSoftware] = useState<ISoftware[]>([]);
  const [usersState, setUsersState] = useState<{ username: string }[]>([]);
  const [usersSearchString, setUsersSearchString] = useState("");
  const [pathname, setPathname] = useState("");
  const [
    hostMdmDeviceStatus,
    setHostMdmDeviceState,
  ] = useState<HostMdmDeviceStatusUIState>("unlocked");

  // activity states
  const [activeActivityTab, setActiveActivityTab] = useState<
    "past" | "upcoming"
  >("past");
  const [activityPage, setActivityPage] = useState(0);

  const { data: fleetQueries, error: fleetQueriesError } = useQuery<
    IListQueriesResponse,
    Error,
    ISchedulableQuery[],
    IQueryKeyQueriesLoadAll[]
  >([{ scope: "queries", teamId: undefined }], () => queryAPI.loadAll(), {
    enabled: !!hostIdFromURL,
    refetchOnMount: false,
    refetchOnReconnect: false,
    refetchOnWindowFocus: false,
    retry: false,
    select: (data: IListQueriesResponse) => data.queries,
  });

  const { data: teams } = useQuery<ILoadTeamsResponse, Error, ITeam[]>(
    "teams",
    () => teamAPI.loadAll(),
    {
      enabled: !!hostIdFromURL && !!isPremiumTier,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: ILoadTeamsResponse) => data.teams,
    }
  );

  const { data: deviceMapping, refetch: refetchDeviceMapping } = useQuery(
    ["deviceMapping", hostIdFromURL],
    () => hostAPI.loadHostDetailsExtension(hostIdFromURL, "device_mapping"),
    {
      enabled: !!hostIdFromURL,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: IDeviceMappingResponse) => data.device_mapping,
    }
  );

  const { data: mdm, refetch: refetchMdm } = useQuery<IHostMdmData>(
    ["mdm", hostIdFromURL],
    () => hostAPI.getMdm(hostIdFromURL),
    {
      enabled: !!hostIdFromURL,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      onError: (err) => {
        // no handling needed atm. data is simply not shown.
        console.error(err);
      },
    }
  );

  const { data: macadmins, refetch: refetchMacadmins } = useQuery(
    ["macadmins", hostIdFromURL],
    () => hostAPI.loadHostDetailsExtension(hostIdFromURL, "macadmins"),
    {
      enabled: !!hostIdFromURL,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: IMacadminsResponse) => data.macadmins,
    }
  );

  const refetchExtensions = () => {
    deviceMapping !== null && refetchDeviceMapping();
    macadmins !== null && refetchMacadmins();
    mdm?.enrollment_status !== null && refetchMdm();
  };

  const {
    isLoading: isLoadingHost,
    data: host,
    refetch: refetchHostDetails,
  } = useQuery<IHostResponse, Error, IHost>(
    ["host", hostIdFromURL],
    () => hostAPI.loadHostDetails(hostIdFromURL),
    {
      enabled: !!hostIdFromURL,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: IHostResponse) => data.host,
      onSuccess: (returnedHost) => {
        setShowRefetchSpinner(returnedHost.refetch_requested);
        setHostMdmDeviceState(
          getHostDeviceStatusUIState(
            returnedHost.mdm.device_status,
            returnedHost.mdm.pending_action
          )
        );
        if (returnedHost.refetch_requested) {
          // If the API reports that a Fleet refetch request is pending, we want to check back for fresh
          // host details. Here we set a one second timeout and poll the API again using
          // fullyReloadHost. We will repeat this process with each onSuccess cycle for a total of
          // 60 seconds or until the API reports that the Fleet refetch request has been resolved
          // or that the host has gone offline.
          if (!refetchStartTime) {
            // If our 60 second timer wasn't already started (e.g., if a refetch was pending when
            // the first page loads), we start it now if the host is online. If the host is offline,
            // we skip the refetch on page load.
            if (returnedHost.status === "online") {
              setRefetchStartTime(Date.now());
              setTimeout(() => {
                refetchHostDetails();
                refetchExtensions();
              }, 1000);
            } else {
              setShowRefetchSpinner(false);
            }
          } else {
            const totalElapsedTime = Date.now() - refetchStartTime;
            if (totalElapsedTime < 60000) {
              if (returnedHost.status === "online") {
                setTimeout(() => {
                  refetchHostDetails();
                  refetchExtensions();
                }, 1000);
              } else {
                renderFlash(
                  "error",
                  `This host is offline. Please try refetching host vitals later.`
                );
                setShowRefetchSpinner(false);
              }
            } else {
              renderFlash(
                "error",
                `We're having trouble fetching fresh vitals for this host. Please try again later.`
              );
              setShowRefetchSpinner(false);
            }
          }
          return; // exit early because refectch is pending so we can avoid unecessary steps below
        }
        setHostSoftware(returnedHost.software || []);
        setUsersState(returnedHost.users || []);
        setSchedule(schedule);
        if (returnedHost.pack_stats) {
          const packStatsByType = returnedHost.pack_stats.reduce(
            (
              dictionary: {
                packs: IPackStats[];
                schedule: IQueryStats[];
              },
              pack: IPackStats
            ) => {
              if (pack.type === "pack") {
                dictionary.packs.push(pack);
              } else {
                dictionary.schedule.push(...pack.query_stats);
              }
              return dictionary;
            },
            { packs: [], schedule: [] }
          );
          setSchedule(packStatsByType.schedule);
        }
      },
      onError: (error) => handlePageError(error),
    }
  );

  // get activities data. This is at the host details level because we want to
  // wait to show the host details page until we have the activities data.
  const {
    data: pastActivities,
    isFetching: pastActivitiesIsFetching,
    isLoading: pastActivitiesIsLoading,
    isError: pastActivitiesIsError,
    refetch: refetchPastActivities,
  } = useQuery<
    IPastActivitiesResponse,
    Error,
    IPastActivitiesResponse,
    Array<{
      scope: string;
      pageIndex: number;
      perPage: number;
      activeTab: "past" | "upcoming";
    }>
  >(
    [
      {
        scope: "past-activities",
        pageIndex: activityPage,
        perPage: DEFAULT_ACTIVITY_PAGE_SIZE,
        activeTab: activeActivityTab,
      },
    ],
    ({ queryKey: [{ pageIndex: page, perPage }] }) => {
      return activitiesAPI.getHostPastActivities(hostIdFromURL, page, perPage);
    },
    {
      keepPreviousData: true,
      staleTime: 2000,
    }
  );

  const {
    data: upcomingActivities,
    isFetching: upcomingActivitiesIsFetching,
    isLoading: upcomingActivitiesIsLoading,
    isError: upcomingActivitiesIsError,
    refetch: refetchUpcomingActivities,
  } = useQuery<
    IUpcomingActivitiesResponse,
    Error,
    IUpcomingActivitiesResponse,
    Array<{
      scope: string;
      pageIndex: number;
      perPage: number;
      activeTab: "past" | "upcoming";
    }>
  >(
    [
      {
        scope: "upcoming-activities",
        pageIndex: activityPage,
        perPage: DEFAULT_ACTIVITY_PAGE_SIZE,
        activeTab: activeActivityTab,
      },
    ],
    ({ queryKey: [{ pageIndex: page, perPage }] }) => {
      return activitiesAPI.getHostUpcomingActivities(
        hostIdFromURL,
        page,
        perPage
      );
    },
    {
      keepPreviousData: true,
      staleTime: 2000,
    }
  );

  const featuresConfig = host?.team_id
    ? teams?.find((t) => t.id === host.team_id)?.features
    : config?.features;

  useEffect(() => {
    setUsersState(() => {
      return (
        host?.users.filter((user) => {
          return user.username
            .toLowerCase()
            .includes(usersSearchString.toLowerCase());
        }) || []
      );
    });
  }, [usersSearchString, host?.users]);

  // Updates title that shows up on browser tabs
  useEffect(() => {
    if (host?.display_name) {
      // e.g., Rachel's Macbook Pro | Hosts | Fleet
      document.title = `${host?.display_name} | Hosts | ${DOCUMENT_TITLE_SUFFIX}`;
    } else {
      document.title = `Hosts | ${DOCUMENT_TITLE_SUFFIX}`;
    }
  }, [location.pathname, host]);

  // Used for back to software pathname
  useEffect(() => {
    setPathname(location.pathname + location.search);
  }, [location]);

  const summaryData = normalizeEmptyValues(pick(host, HOST_SUMMARY_DATA));

  const aboutData = normalizeEmptyValues(pick(host, HOST_ABOUT_DATA));

  const osqueryData = normalizeEmptyValues(pick(host, HOST_OSQUERY_DATA));

  const togglePolicyDetailsModal = useCallback(
    (policy: IHostPolicy) => {
      setPolicyDetailsModal(!showPolicyDetailsModal);
      setSelectedPolicy(policy);
    },
    [showPolicyDetailsModal, setPolicyDetailsModal, setSelectedPolicy]
  );

  const toggleOSSettingsModal = useCallback(() => {
    setShowOSSettingsModal(!showOSSettingsModal);
  }, [showOSSettingsModal, setShowOSSettingsModal]);

  const toggleBootstrapPackageModal = useCallback(() => {
    setShowBootstrapPackageModal(!showBootstrapPackageModal);
  }, [showBootstrapPackageModal, setShowBootstrapPackageModal]);

  const onCancelPolicyDetailsModal = useCallback(() => {
    setPolicyDetailsModal(!showPolicyDetailsModal);
    setSelectedPolicy(null);
  }, [showPolicyDetailsModal, setPolicyDetailsModal, setSelectedPolicy]);

  const toggleUnenrollMdmModal = useCallback(() => {
    setShowUnenrollMdmModal(!showUnenrollMdmModal);
  }, [showUnenrollMdmModal, setShowUnenrollMdmModal]);

  const onDestroyHost = async () => {
    if (host) {
      setIsUpdatingHost(true);
      try {
        await hostAPI.destroy(host);
        renderFlash(
          "success",
          `Host "${host.display_name}" was successfully deleted.`
        );
        router.push(PATHS.MANAGE_HOSTS);
      } catch (error) {
        console.log(error);
        renderFlash(
          "error",
          `Host "${host.display_name}" could not be deleted.`
        );
      } finally {
        setShowDeleteHostModal(false);
        setIsUpdatingHost(false);
      }
    }
  };

  const onRefetchHost = async () => {
    if (host) {
      // Once the user clicks to refetch, the refetch loading spinner should continue spinning
      // unless there is an error. The spinner state is also controlled in the fullyReloadHost
      // method.
      setShowRefetchSpinner(true);
      try {
        await hostAPI.refetch(host).then(() => {
          setRefetchStartTime(Date.now());
          setTimeout(() => {
            refetchHostDetails();
            refetchExtensions();
          }, 1000);
        });
      } catch (error) {
        console.log(error);
        renderFlash("error", `Host "${host.display_name}" refetch error`);
        setShowRefetchSpinner(false);
      }
    }
  };

  const onChangeActivityTab = (tabIndex: number) => {
    setActiveActivityTab(tabIndex === 0 ? "past" : "upcoming");
    setActivityPage(0);
  };

  const onShowActivityDetails = useCallback(
    ({ type, details }: IShowActivityDetailsData) => {
      switch (type) {
        case "ran_script":
          setScriptDetailsId(details?.script_execution_id || "");
          break;
        default: // do nothing
      }
    },
    []
  );

  const onLabelClick = (label: ILabel) => {
    return label.name === "All Hosts"
      ? router.push(PATHS.MANAGE_HOSTS)
      : router.push(PATHS.MANAGE_HOSTS_LABEL(label.id));
  };

  const onQueryHostCustom = () => {
    setSelectedQueryTargetsByType(DEFAULT_TARGETS_BY_TYPE);
    router.push(
      PATHS.NEW_QUERY() +
        TAGGED_TEMPLATES.queryByHostRoute(host?.id, currentTeam?.id)
    );
  };

  const onQueryHostSaved = (selectedQuery: ISchedulableQuery) => {
    setSelectedQueryTargetsByType(DEFAULT_TARGETS_BY_TYPE);
    router.push(
      PATHS.EDIT_QUERY(selectedQuery.id) +
        TAGGED_TEMPLATES.queryByHostRoute(host?.id, currentTeam?.id)
    );
  };

  const onCancelScriptDetailsModal = useCallback(() => {
    setScriptDetailsId("");
  }, [setScriptDetailsId]);

  const onTransferHostSubmit = async (team: ITeam) => {
    setIsUpdatingHost(true);

    const teamId = typeof team.id === "number" ? team.id : null;

    try {
      await hostAPI.transferToTeam(teamId, [hostIdFromURL]);

      const successMessage =
        teamId === null
          ? `Host successfully removed from teams.`
          : `Host successfully transferred to  ${team.name}.`;

      renderFlash("success", successMessage);
      refetchHostDetails(); // Note: it is not necessary to `refetchExtensions` here because only team has changed
      setShowTransferHostModal(false);
    } catch (error) {
      console.log(error);
      renderFlash("error", "Could not transfer host. Please try again.");
    } finally {
      setIsUpdatingHost(false);
    }
  };

  const onUsersTableSearchChange = useCallback(
    (queryData: ISearchQueryData) => {
      const { searchQuery } = queryData;
      setUsersSearchString(searchQuery);
    },
    []
  );

  const onCloseRunScriptModal = useCallback(() => {
    setShowRunScriptModal(false);
    refetchPastActivities();
    refetchUpcomingActivities();
  }, [refetchPastActivities, refetchUpcomingActivities]);

  const onSelectHostAction = (action: string) => {
    switch (action) {
      case "transfer":
        setShowTransferHostModal(true);
        break;
      case "query":
        setShowSelectQueryModal(true);
        break;
      case "diskEncryption":
        setShowDiskEncryptionModal(true);
        break;
      case "mdmOff":
        toggleUnenrollMdmModal();
        break;
      case "delete":
        setShowDeleteHostModal(true);
        break;
      case "runScript":
        setShowRunScriptModal(true);
        break;
      case "lock":
        setShowLockHostModal(true);
        break;
      case "unlock":
        setShowUnlockHostModal(true);
        break;
      case "wipe":
        setShowWipeModal(true);
        break;
      default: // do nothing
    }
  };

  // const hostDeviceStatusUIState = getHostDeviceStatusUIState(
  //   host.mdm.device_status,
  //   host.mdm.pending_action
  // );

  const renderActionButtons = () => {
    if (!host) {
      return null;
    }

    return (
      <HostActionDropdown
        hostTeamId={host.team_id}
        onSelect={onSelectHostAction}
        hostPlatform={host.platform}
        hostStatus={host.status}
        hostMdmDeviceStatus={hostMdmDeviceStatus}
        hostMdmEnrollmentStatus={host.mdm.enrollment_status}
        doesStoreEncryptionKey={host.mdm.encryption_key_available}
        mdmName={mdm?.name}
        hostScriptsEnabled={host.scripts_enabled}
      />
    );
  };

  if (
    !host ||
    isLoadingHost ||
    pastActivitiesIsLoading ||
    upcomingActivitiesIsLoading
  ) {
    return <Spinner />;
  }
  const failingPoliciesCount = host?.issues.failing_policies_count || 0;

  const hostDetailsSubNav: IHostDetailsSubNavItem[] = [
    {
      name: "Details",
      title: "details",
      pathname: PATHS.HOST_DETAILS(hostIdFromURL),
    },
    {
      name: "Software",
      title: "software",
      pathname: PATHS.HOST_SOFTWARE(hostIdFromURL),
    },
    {
      name: "Queries",
      title: "queries",
      pathname: PATHS.HOST_QUERIES(hostIdFromURL),
    },
    {
      name: (
        <>
          {failingPoliciesCount > 0 && (
            <span className="count">{failingPoliciesCount}</span>
          )}
          Policies
        </>
      ),
      title: "policies",
      pathname: PATHS.HOST_POLICIES(hostIdFromURL),
    },
  ];

  const getTabIndex = (path: string): number => {
    return hostDetailsSubNav.findIndex((navItem) => {
      // tab stays highlighted for paths that ends with same pathname
      return path.endsWith(navItem.pathname);
    });
  };

  const navigateToNav = (i: number): void => {
    const navPath = hostDetailsSubNav[i].pathname;
    router.push(navPath);
  };

  /*  Context team id might be different that host's team id
  Observer plus must be checked against host's team id  */
  const isGlobalOrHostsTeamObserverPlus =
    currentUser && host?.team_id
      ? permissions.isObserverPlus(currentUser, host.team_id)
      : false;

  const isHostsTeamObserver =
    currentUser && host?.team_id
      ? permissions.isTeamObserver(currentUser, host.team_id)
      : false;

  const canViewPacks =
    !isGlobalObserver &&
    !isGlobalOrHostsTeamObserverPlus &&
    !isHostsTeamObserver;

  const bootstrapPackageData = {
    status: host?.mdm.macos_setup?.bootstrap_package_status,
    details: host?.mdm.macos_setup?.details,
    name: host?.mdm.macos_setup?.bootstrap_package_name,
  };

  return (
    <MainContent className={baseClass}>
      <>
        <HostDetailsBanners
          hostMdmEnrollmentStatus={host?.mdm.enrollment_status}
          hostPlatform={host?.platform}
          mdmName={host?.mdm.name}
          diskEncryptionStatus={host?.mdm.macos_settings?.disk_encryption}
        />
        <div className={`${baseClass}__header-links`}>
          <BackLink
            text="Back to all hosts"
            path={filteredHostsPath || PATHS.MANAGE_HOSTS}
          />
        </div>
        <HostSummaryCard
          summaryData={summaryData}
          bootstrapPackageData={bootstrapPackageData}
          isPremiumTier={isPremiumTier}
          isSandboxMode={isSandboxMode}
          toggleOSSettingsModal={toggleOSSettingsModal}
          toggleBootstrapPackageModal={toggleBootstrapPackageModal}
          hostMdmProfiles={host?.mdm.profiles ?? []}
          mdmName={mdm?.name}
          showRefetchSpinner={showRefetchSpinner}
          onRefetchHost={onRefetchHost}
          renderActionButtons={renderActionButtons}
          osSettings={host?.mdm.os_settings}
          hostMdmDeviceStatus={hostMdmDeviceStatus}
        />
        <TabsWrapper className={`${baseClass}__tabs-wrapper`}>
          <Tabs
            selectedIndex={getTabIndex(location.pathname)}
            onSelect={(i) => navigateToNav(i)}
          >
            <TabList>
              {hostDetailsSubNav.map((navItem) => {
                // Bolding text when the tab is active causes a layout shift
                // so we add a hidden pseudo element with the same text string
                return <Tab key={navItem.title}>{navItem.name}</Tab>;
              })}
            </TabList>
            <TabPanel className={`${baseClass}__details-panel`}>
              <AboutCard
                aboutData={aboutData}
                deviceMapping={deviceMapping}
                munki={macadmins?.munki}
                mdm={mdm}
              />
              <ActivityCard
                activeTab={activeActivityTab}
                activities={
                  activeActivityTab === "past"
                    ? pastActivities
                    : upcomingActivities
                }
                isLoading={
                  activeActivityTab === "past"
                    ? pastActivitiesIsFetching
                    : upcomingActivitiesIsFetching
                }
                isError={
                  activeActivityTab === "past"
                    ? pastActivitiesIsError
                    : upcomingActivitiesIsError
                }
                upcomingCount={upcomingActivities?.count || 0}
                onChangeTab={onChangeActivityTab}
                onNextPage={() => setActivityPage(activityPage + 1)}
                onPreviousPage={() => setActivityPage(activityPage - 1)}
                onShowDetails={onShowActivityDetails}
              />
              <AgentOptionsCard
                osqueryData={osqueryData}
                wrapFleetHelper={wrapFleetHelper}
                isChromeOS={host?.platform === "chrome"}
              />
              <LabelsCard
                labels={host?.labels || []}
                onLabelClick={onLabelClick}
              />
              <UsersCard
                users={host?.users || []}
                usersState={usersState}
                isLoading={isLoadingHost}
                onUsersTableSearchChange={onUsersTableSearchChange}
                hostUsersEnabled={featuresConfig?.enable_host_users}
              />
            </TabPanel>
            <TabPanel>
              <SoftwareCard
                isLoading={isLoadingHost}
                software={hostSoftware}
                isSoftwareEnabled={featuresConfig?.enable_software_inventory}
                deviceType={host?.platform === "darwin" ? "macos" : ""}
                router={router}
                queryParams={queryParams}
                routeTemplate={routeTemplate}
                pathname={pathname}
                pathPrefix={PATHS.HOST_SOFTWARE(host?.id || 0)}
              />
              {host?.platform === "darwin" && macadmins?.munki?.version && (
                <MunkiIssuesCard
                  isLoading={isLoadingHost}
                  munkiIssues={macadmins.munki_issues}
                  deviceType={host?.platform === "darwin" ? "macos" : ""}
                />
              )}
            </TabPanel>
            <TabPanel>
              <QueriesCard
                hostId={host.id}
                router={router}
                isChromeOSHost={host.platform === "chrome"}
                schedule={schedule}
                queryReportsDisabled={
                  config?.server_settings?.query_reports_disabled
                }
              />
              {canViewPacks && (
                <PacksCard packsState={packsState} isLoading={isLoadingHost} />
              )}
            </TabPanel>
            <TabPanel>
              <PoliciesCard
                policies={host?.policies || []}
                isLoading={isLoadingHost}
                togglePolicyDetailsModal={togglePolicyDetailsModal}
              />
            </TabPanel>
          </Tabs>
        </TabsWrapper>
        {showDeleteHostModal && (
          <DeleteHostModal
            onCancel={() => setShowDeleteHostModal(false)}
            onSubmit={onDestroyHost}
            hostName={host?.display_name}
            isUpdating={isUpdatingHost}
          />
        )}
        {showSelectQueryModal && host && (
          <SelectQueryModal
            onCancel={() => setShowSelectQueryModal(false)}
            queries={fleetQueries || []}
            queryErrors={fleetQueriesError}
            isOnlyObserver={isOnlyObserver}
            onQueryHostCustom={onQueryHostCustom}
            onQueryHostSaved={onQueryHostSaved}
            hostsTeamId={host?.team_id}
          />
        )}
        {showRunScriptModal && (
          <RunScriptModal
            host={host}
            currentUser={currentUser}
            scriptDetailsId={scriptDetailsId}
            setScriptDetailsId={setScriptDetailsId}
            onClose={onCloseRunScriptModal}
          />
        )}
        {!!host && showTransferHostModal && (
          <TransferHostModal
            onCancel={() => setShowTransferHostModal(false)}
            onSubmit={onTransferHostSubmit}
            teams={teams || []}
            isGlobalAdmin={isGlobalAdmin as boolean}
            isUpdating={isUpdatingHost}
          />
        )}
        {!!host && showPolicyDetailsModal && (
          <PolicyDetailsModal
            onCancel={onCancelPolicyDetailsModal}
            policy={selectedPolicy}
          />
        )}
        {showOSSettingsModal && (
          <OSSettingsModal
            platform={host?.platform}
            hostMDMData={host?.mdm}
            onClose={toggleOSSettingsModal}
          />
        )}
        {showUnenrollMdmModal && !!host && (
          <UnenrollMdmModal hostId={host.id} onClose={toggleUnenrollMdmModal} />
        )}
        {showDiskEncryptionModal &&
          host &&
          isSupportedPlatform(host.platform) && (
            <DiskEncryptionKeyModal
              platform={host.platform}
              hostId={host.id}
              onCancel={() => setShowDiskEncryptionModal(false)}
            />
          )}
        {showBootstrapPackageModal &&
          bootstrapPackageData.details &&
          bootstrapPackageData.name && (
            <BootstrapPackageModal
              packageName={bootstrapPackageData.name}
              details={bootstrapPackageData.details}
              onClose={() => setShowBootstrapPackageModal(false)}
            />
          )}
        {!!scriptDetailsId && (
          <ScriptDetailsModal
            scriptExecutionId={scriptDetailsId}
            onCancel={onCancelScriptDetailsModal}
          />
        )}
        {showLockHostModal && (
          <LockModal
            id={host.id}
            platform={host.platform}
            hostName={host.display_name}
            onSuccess={() => setHostMdmDeviceState("locking")}
            onClose={() => setShowLockHostModal(false)}
          />
        )}
        {showUnlockHostModal && (
          <UnlockModal
            id={host.id}
            platform={host.platform}
            hostName={host.display_name}
            onSuccess={() => {
              host.platform !== "darwin" && setHostMdmDeviceState("unlocking");
            }}
            onClose={() => setShowUnlockHostModal(false)}
          />
        )}
        {showWipeModal && (
          <WipeModal
            id={host.id}
            hostName={host.display_name}
            onSuccess={() => setHostMdmDeviceState("wiping")}
            onClose={() => setShowWipeModal(false)}
          />
        )}
      </>
    </MainContent>
  );
};

export default HostDetailsPage;
