import React, { useContext, useState, useCallback, useEffect } from "react";
import { Params, InjectedRouter } from "react-router/lib/Router";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import { RouteProps } from "react-router";

import { pick } from "lodash";

import PATHS from "router/paths";
import hostAPI from "services/entities/hosts";
import queryAPI from "services/entities/queries";
import teamAPI, { ILoadTeamsResponse } from "services/entities/teams";
import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import { NotificationContext } from "context/notification";
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
import { ITeam } from "interfaces/team";
import {
  IListQueriesResponse,
  IQueryKeyQueriesLoadAll,
  ISchedulableQuery,
} from "interfaces/schedulable_query";

import Spinner from "components/Spinner";
import TabsWrapper from "components/TabsWrapper";
import MainContent from "components/MainContent";
import InfoBanner from "components/InfoBanner";
import BackLink from "components/BackLink";

import { normalizeEmptyValues, wrapFleetHelper } from "utilities/helpers";
import permissions from "utilities/permissions";

import HostSummaryCard from "../cards/HostSummary";
import AboutCard from "../cards/About";
import AgentOptionsCard from "../cards/AgentOptions";
import LabelsCard from "../cards/Labels";
import MunkiIssuesCard from "../cards/MunkiIssues";
import SoftwareCard from "../cards/Software";
import UsersCard from "../cards/Users";
import PoliciesCard from "../cards/Policies";
import ScheduleCard from "../cards/Schedule";
import PacksCard from "../cards/Packs";
import PolicyDetailsModal from "../cards/Policies/HostPoliciesTable/PolicyDetailsModal";
import OSPolicyModal from "./modals/OSPolicyModal";
import UnenrollMdmModal from "./modals/UnenrollMdmModal";
import TransferHostModal from "../../components/TransferHostModal";
import DeleteHostModal from "../../components/DeleteHostModal";

import parseOsVersion from "./modals/OSPolicyModal/helpers";

import DiskEncryptionKeyModal from "./modals/DiskEncryptionKeyModal";
import HostActionDropdown from "./HostActionsDropdown/HostActionsDropdown";
import MacSettingsModal from "../MacSettingsModal";
import BootstrapPackageModal from "./modals/BootstrapPackageModal";
import SelectQueryModal from "./modals/SelectQueryModal";

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

const TAGGED_TEMPLATES = {
  queryByHostRoute: (hostId: number | undefined | null) => {
    return `${hostId ? `?host_ids=${hostId}` : ""}`;
  },
};

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
  } = useContext(AppContext);
  const {
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryResolution,
    setLastEditedQueryCritical,
    setPolicyTeamId,
  } = useContext(PolicyContext);
  const { renderFlash } = useContext(NotificationContext);

  const handlePageError = useErrorHandler();

  const [showDeleteHostModal, setShowDeleteHostModal] = useState(false);
  const [showTransferHostModal, setShowTransferHostModal] = useState(false);
  const [showSelectQueryModal, setShowSelectQueryModal] = useState(false);
  const [showPolicyDetailsModal, setPolicyDetailsModal] = useState(false);
  const [showOSPolicyModal, setShowOSPolicyModal] = useState(false);
  const [showMacSettingsModal, setShowMacSettingsModal] = useState(false);
  const [showUnenrollMdmModal, setShowUnenrollMdmModal] = useState(false);
  const [showDiskEncryptionModal, setShowDiskEncryptionModal] = useState(false);
  const [showBootstrapPackageModal, setShowBootstrapPackageModal] = useState(
    false
  );
  const [selectedPolicy, setSelectedPolicy] = useState<IHostPolicy | null>(
    null
  );
  const [isUpdatingHost, setIsUpdatingHost] = useState(false);

  const [refetchStartTime, setRefetchStartTime] = useState<number | null>(null);
  const [showRefetchSpinner, setShowRefetchSpinner] = useState(false);
  const [schedule, setSchedule] = useState<IQueryStats[]>();
  const [packsState, setPacksState] = useState<IPackStats[]>();
  const [hostSoftware, setHostSoftware] = useState<ISoftware[]>([]);
  const [usersState, setUsersState] = useState<{ username: string }[]>([]);
  const [usersSearchString, setUsersSearchString] = useState("");
  const [pathname, setPathname] = useState("");

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
          setPacksState(packStatsByType.packs);
        }
      },
      onError: (error) => handlePageError(error),
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

  // Used for back to software pathname
  useEffect(() => {
    setPathname(location.pathname + location.search);
  }, [location]);

  const titleData = normalizeEmptyValues(
    pick(host, [
      "id",
      "status",
      "issues",
      "memory",
      "cpu_type",
      "platform",
      "os_version",
      "osquery_version",
      "enroll_secret_name",
      "detail_updated_at",
      "percent_disk_space_available",
      "gigs_disk_space_available",
      "team_name",
      "display_name",
    ])
  );

  const [osPolicyLabel, osPolicyQuery] = parseOsVersion(host?.os_version);

  const aboutData = normalizeEmptyValues(
    pick(host, [
      "seen_time",
      "uptime",
      "last_enrolled_at",
      "hardware_model",
      "hardware_serial",
      "primary_ip",
      "public_ip",
      "geolocation",
      "batteries",
      "detail_updated_at",
    ])
  );

  const osqueryData = normalizeEmptyValues(
    pick(host, [
      "config_tls_refresh",
      "logger_tls_period",
      "distributed_interval",
    ])
  );

  const togglePolicyDetailsModal = useCallback(
    (policy: IHostPolicy) => {
      setPolicyDetailsModal(!showPolicyDetailsModal);
      setSelectedPolicy(policy);
    },
    [showPolicyDetailsModal, setPolicyDetailsModal, setSelectedPolicy]
  );

  const toggleOSPolicyModal = useCallback(() => {
    setShowOSPolicyModal(!showOSPolicyModal);
  }, [showOSPolicyModal, setShowOSPolicyModal]);

  const toggleMacSettingsModal = useCallback(() => {
    setShowMacSettingsModal(!showMacSettingsModal);
  }, [showMacSettingsModal, setShowMacSettingsModal]);

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

  const onCreateNewPolicy = () => {
    const { NEW_POLICY } = PATHS;
    host?.team_name
      ? setLastEditedQueryName(`${osPolicyLabel} (${host.team_name})`)
      : setLastEditedQueryName(osPolicyLabel);
    setPolicyTeamId(host?.team_id ? host?.team_id : 0);
    setLastEditedQueryDescription(
      "Checks to see if the required minimum operating system version is installed."
    );
    setLastEditedQueryBody(osPolicyQuery);
    setLastEditedQueryResolution("");
    setLastEditedQueryCritical(false);
    router.replace(NEW_POLICY);
  };

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

  const onLabelClick = (label: ILabel) => {
    return label.name === "All Hosts"
      ? router.push(PATHS.MANAGE_HOSTS)
      : router.push(PATHS.MANAGE_HOSTS_LABEL(label.id));
  };

  const onQueryHostCustom = () => {
    router.push(PATHS.NEW_QUERY + TAGGED_TEMPLATES.queryByHostRoute(host?.id));
  };

  const onQueryHostSaved = (selectedQuery: ISchedulableQuery) => {
    router.push(
      PATHS.EDIT_QUERY(selectedQuery.id) +
        TAGGED_TEMPLATES.queryByHostRoute(host?.id)
    );
  };

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
      default:
    }
  };

  const renderActionButtons = () => {
    if (!host) {
      return null;
    }

    return (
      <HostActionDropdown
        onSelect={onSelectHostAction}
        hostStatus={host.status}
        hostMdmEnrollemntStatus={host.mdm.enrollment_status}
        doesStoreEncryptionKey={host.mdm.encryption_key_available}
        mdmName={mdm?.name}
      />
    );
  };

  if (isLoadingHost) {
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
      name: "Schedule",
      title: "schedule",
      pathname: PATHS.HOST_SCHEDULE(hostIdFromURL),
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

  const isMdmUnenrolled =
    host?.mdm.enrollment_status === "Off" || !host?.mdm.enrollment_status;

  const showDiskEncryptionUserActionRequired =
    config?.mdm.enabled_and_configured &&
    host?.mdm.name === "Fleet" &&
    host?.mdm.macos_settings?.disk_encryption === "action_required";

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
      <div className={`${baseClass}__wrapper`}>
        {host?.platform === "darwin" &&
          isMdmUnenrolled &&
          config?.mdm.enabled_and_configured && (
            <InfoBanner color="yellow">
              To change settings and install software, ask the end user to
              follow the <strong>Turn on MDM</strong> instructions on their{" "}
              <strong>My device</strong> page.
            </InfoBanner>
          )}
        {showDiskEncryptionUserActionRequired && (
          <InfoBanner color="yellow">
            Disk encryption: Requires action from the end user. Ask the end user
            to follow <b>Disk encryption</b> instructions on their{" "}
            <b>My device</b> page.
          </InfoBanner>
        )}
        <div className={`${baseClass}__header-links`}>
          <BackLink
            text="Back to all hosts"
            path={filteredHostsPath || PATHS.MANAGE_HOSTS}
          />
        </div>
        <HostSummaryCard
          titleData={titleData}
          diskEncryptionEnabled={host?.disk_encryption_enabled}
          bootstrapPackageData={bootstrapPackageData}
          isPremiumTier={isPremiumTier}
          isSandboxMode={isSandboxMode}
          isOnlyObserver={isOnlyObserver}
          toggleOSPolicyModal={toggleOSPolicyModal}
          toggleMacSettingsModal={toggleMacSettingsModal}
          toggleBootstrapPackageModal={toggleBootstrapPackageModal}
          hostMdmProfiles={host?.mdm.profiles ?? []}
          mdmName={mdm?.name}
          showRefetchSpinner={showRefetchSpinner}
          onRefetchHost={onRefetchHost}
          renderActionButtons={renderActionButtons}
        />
        <TabsWrapper>
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
            <TabPanel>
              <AboutCard
                aboutData={aboutData}
                deviceMapping={deviceMapping}
                munki={macadmins?.munki}
                mdm={mdm}
                wrapFleetHelper={wrapFleetHelper}
              />
              <div className="col-2">
                <AgentOptionsCard
                  osqueryData={osqueryData}
                  wrapFleetHelper={wrapFleetHelper}
                  isChromeOS={host?.platform === "chrome"}
                />
                <LabelsCard
                  labels={host?.labels || []}
                  onLabelClick={onLabelClick}
                />
              </div>
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
              {host?.platform === "darwin" && macadmins && (
                <MunkiIssuesCard
                  isLoading={isLoadingHost}
                  munkiIssues={macadmins.munki_issues}
                  deviceType={host?.platform === "darwin" ? "macos" : ""}
                />
              )}
            </TabPanel>
            <TabPanel>
              <ScheduleCard
                isChromeOSHost={host?.platform === "chrome"}
                schedule={schedule}
                isLoading={isLoadingHost}
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
        {showOSPolicyModal && (
          <OSPolicyModal
            onCancel={() => setShowOSPolicyModal(false)}
            onCreateNewPolicy={onCreateNewPolicy}
            osVersion={host?.os_version}
            detailsUpdatedAt={host?.detail_updated_at}
            osPolicy={osPolicyQuery}
            osPolicyLabel={osPolicyLabel}
          />
        )}
        {showMacSettingsModal && (
          <MacSettingsModal
            hostMDMData={host?.mdm}
            onClose={toggleMacSettingsModal}
          />
        )}
        {showUnenrollMdmModal && !!host && (
          <UnenrollMdmModal hostId={host.id} onClose={toggleUnenrollMdmModal} />
        )}
        {showDiskEncryptionModal && host && (
          <DiskEncryptionKeyModal
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
      </div>
    </MainContent>
  );
};

export default HostDetailsPage;
