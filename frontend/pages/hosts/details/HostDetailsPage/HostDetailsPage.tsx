import React, { useContext, useState, useCallback, useEffect } from "react";
import { Params, InjectedRouter } from "react-router/lib/Router";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import { pick } from "lodash";

import PATHS from "router/paths";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import activitiesAPI, {
  IHostPastActivitiesResponse,
  IHostUpcomingActivitiesResponse,
} from "services/entities/activities";
import hostAPI, {
  IGetHostCertificatesResponse,
  IGetHostCertsRequestParams,
} from "services/entities/hosts";
import teamAPI, { ILoadTeamsResponse } from "services/entities/teams";

import {
  IHost,
  IMacadminsResponse,
  IHostResponse,
  IHostMdmData,
  IPackStats,
} from "interfaces/host";
import { ILabel } from "interfaces/label";
import { IListSort } from "interfaces/list_options";
import { IHostPolicy } from "interfaces/policy";
import { IQueryStats } from "interfaces/query_stats";
import { IHostSoftware } from "interfaces/software";
import { ITeam } from "interfaces/team";
import { ActivityType, IHostUpcomingActivity } from "interfaces/activity";
import {
  IHostCertificate,
  CERTIFICATES_DEFAULT_SORT,
} from "interfaces/certificates";

import { normalizeEmptyValues, wrapFleetHelper } from "utilities/helpers";
import permissions from "utilities/permissions";
import {
  DOCUMENT_TITLE_SUFFIX,
  HOST_SUMMARY_DATA,
  HOST_ABOUT_DATA,
  HOST_OSQUERY_DATA,
  DEFAULT_USE_QUERY_OPTIONS,
} from "utilities/constants";

import { isAndroid, isIPadOrIPhone, isLinuxLike } from "interfaces/platform";

import Spinner from "components/Spinner";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import MainContent from "components/MainContent";
import BackLink from "components/BackLink";
import Card from "components/Card";
import RunScriptDetailsModal from "pages/DashboardPage/cards/ActivityFeed/components/RunScriptDetailsModal";
import {
  AppInstallDetailsModal,
  IAppInstallDetails,
} from "components/ActivityDetails/InstallDetails/AppInstallDetails/AppInstallDetails";
import {
  SoftwareInstallDetailsModal,
  IPackageInstallDetails,
} from "components/ActivityDetails/InstallDetails/SoftwareInstallDetails/SoftwareInstallDetails";
import SoftwareUninstallDetailsModal from "components/ActivityDetails/InstallDetails/SoftwareUninstallDetailsModal";
import { IShowActivityDetailsData } from "components/ActivityItem/ActivityItem";

import HostSummaryCard from "../cards/HostSummary";
import AboutCard from "../cards/About";
import UserCard from "../cards/User";
import ActivityCard from "../cards/Activity";
import AgentOptionsCard from "../cards/AgentOptions";
import LabelsCard from "../cards/Labels";
import MunkiIssuesCard from "../cards/MunkiIssues";
import SoftwareInventoryCard from "../cards/Software";
import SoftwareLibraryCard from "../cards/HostSoftwareLibrary";
import LocalUserAccountsCard from "../cards/LocalUserAccounts";
import PoliciesCard from "../cards/Policies";
import QueriesCard from "../cards/Queries";
import PacksCard from "../cards/Packs";
import PolicyDetailsModal from "../cards/Policies/HostPoliciesTable/PolicyDetailsModal";
import CertificatesCard from "../cards/Certificates";

import TransferHostModal from "../../components/TransferHostModal";
import DeleteHostModal from "../../components/DeleteHostModal";

import UnenrollMdmModal from "./modals/UnenrollMdmModal";
import DiskEncryptionKeyModal from "./modals/DiskEncryptionKeyModal";
import HostActionsDropdown from "./HostActionsDropdown/HostActionsDropdown";
import OSSettingsModal from "../OSSettingsModal";
import BootstrapPackageModal from "./modals/BootstrapPackageModal";
import ScriptModalGroup from "./modals/ScriptModalGroup";
import SelectQueryModal from "./modals/SelectQueryModal";
import HostDetailsBanners from "./components/HostDetailsBanners";
import LockModal from "./modals/LockModal";
import UnlockModal from "./modals/UnlockModal";
import {
  HostMdmDeviceStatusUIState,
  getHostDeviceStatusUIState,
} from "../helpers";
import WipeModal from "./modals/WipeModal";
import SoftwareDetailsModal from "../cards/Software/SoftwareDetailsModal";
import { parseHostSoftwareQueryParams } from "../cards/Software/HostSoftware";
import { getErrorMessage } from "./helpers";
import CancelActivityModal from "./modals/CancelActivityModal";
import CertificateDetailsModal from "../modals/CertificateDetailsModal";
import AddEndUserModal from "../cards/User/components/AddEndUserModal";
import {
  generateChromeProfilesValues,
  generateOtherEmailsValues,
  generateUsernameValues,
} from "../cards/User/helpers";
import HostHeader from "../cards/HostHeader";

const baseClass = "host-details";

const defaultCardClass = `${baseClass}__card`;
const fullWidthCardClass = `${baseClass}__card--full-width`;
const doubleHeightCardClass = `${baseClass}__card--double-height`;

interface IHostDetailsProps {
  router: InjectedRouter; // v3
  location: {
    pathname: string;
    query: {
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
  name: React.ReactNode;
  title: string;
  pathname: string;
  count?: number;
}

const DEFAULT_ACTIVITY_PAGE_SIZE = 8;
const DEFAULT_CERTIFICATES_PAGE_SIZE = 10;
const DEFAULT_CERTIFICATES_PAGE = 0;

const HostDetailsPage = ({
  router,
  location,
  params: { host_id },
}: IHostDetailsProps): JSX.Element => {
  const hostIdFromURL = parseInt(host_id, 10);

  const {
    config,
    currentUser,
    isGlobalAdmin = false,
    isGlobalMaintainer,
    isGlobalObserver,
    isPremiumTier = false,
    isOnlyObserver,
    filteredHostsPath,
    currentTeam,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const handlePageError = useErrorHandler();

  const [showDeleteHostModal, setShowDeleteHostModal] = useState(false);
  const [showTransferHostModal, setShowTransferHostModal] = useState(false);
  const [showSelectQueryModal, setShowSelectQueryModal] = useState(false);
  const [showScriptModalGroup, setShowScriptModalGroup] = useState(false);
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

  // Used in activities to show run script details modal
  const [scriptExecutionId, setScriptExecutiontId] = useState("");
  const [selectedPolicy, setSelectedPolicy] = useState<IHostPolicy | null>(
    null
  );
  const [
    packageInstallDetails,
    setPackageInstallDetails,
  ] = useState<IPackageInstallDetails | null>(null);
  const [
    packageUninstallDetails,
    setPackageUninstallDetails,
  ] = useState<IPackageInstallDetails | null>(null);
  const [
    appInstallDetails,
    setAppInstallDetails,
  ] = useState<IAppInstallDetails | null>(null);

  const [isUpdatingHost, setIsUpdatingHost] = useState(false);
  const [refetchStartTime, setRefetchStartTime] = useState<number | null>(null);
  const [showRefetchSpinner, setShowRefetchSpinner] = useState(false);
  const [schedule, setSchedule] = useState<IQueryStats[]>();
  const [packsState, setPackState] = useState<IPackStats[]>();
  const [usersState, setUsersState] = useState<{ username: string }[]>([]);
  const [usersSearchString, setUsersSearchString] = useState("");
  const [
    hostMdmDeviceStatus,
    setHostMdmDeviceState,
  ] = useState<HostMdmDeviceStatusUIState>("unlocked");
  const [
    selectedSoftwareDetails,
    setSelectedSoftwareDetails,
  ] = useState<IHostSoftware | null>(null);
  const [selectedScriptExecutionId, setSelectedScriptExecutionId] = useState<
    string | undefined
  >(undefined);
  const [
    selectedCancelActivity,
    setSelectedCancelActivity,
  ] = useState<IHostUpcomingActivity | null>(null);

  // activity states
  const [activeActivityTab, setActiveActivityTab] = useState<
    "past" | "upcoming"
  >("past");
  const [activityPage, setActivityPage] = useState(0);

  // certificates states
  const [
    selectedCertificate,
    setSelectedCertificate,
  ] = useState<IHostCertificate | null>(null);
  const [certificatePage, setCertificatePage] = useState(
    DEFAULT_CERTIFICATES_PAGE
  );
  const [sortCerts, setSortCerts] = useState<IListSort>({
    ...CERTIFICATES_DEFAULT_SORT,
  });
  const [showAddEndUserModal, setShowAddEndUserModal] = useState(false);

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
      enabled: !!hostIdFromURL, // TODO(android): disable for unsupported platforms?
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: IMacadminsResponse) => data.macadmins,
    }
  );

  const {
    data: hostCertificates,
    isError: isErrorHostCertificates,
    refetch: refetchHostCertificates,
  } = useQuery<
    IGetHostCertificatesResponse,
    Error,
    IGetHostCertificatesResponse,
    Array<IGetHostCertsRequestParams & { scope: "host-certificates" }>
  >(
    [
      {
        scope: "host-certificates",
        host_id: hostIdFromURL,
        page: certificatePage,
        per_page: DEFAULT_CERTIFICATES_PAGE_SIZE,
        order_key: sortCerts.order_key,
        order_direction: sortCerts.order_direction,
      },
    ],
    ({ queryKey }) => hostAPI.getHostCertificates(queryKey[0]),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      // FIXME: is it worth disabling for unsupported platforms? we'd have to workaround the a
      // catch-22 where we need to know the platform to know if it's supported but we also need to
      // be able to include the cert refetch in the hosts query hook.
      enabled: !!hostIdFromURL,
      keepPreviousData: true,
      staleTime: 15000,
    }
  );

  const refetchExtensions = () => {
    macadmins !== null && refetchMacadmins();
    mdm?.enrollment_status !== null && refetchMdm();
    hostCertificates && refetchHostCertificates();
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
        if (
          returnedHost.refetch_requested &&
          !isAndroid(returnedHost.platform)
        ) {
          // If the API reports that a Fleet refetch request is pending, we want to check back for fresh
          // host details. Here we set a one second timeout and poll the API again using
          // fullyReloadHost. We will repeat this process with each onSuccess cycle for a total of
          // 60 seconds or until the API reports that the Fleet refetch request has been resolved
          // or that the host has gone offline.
          if (!refetchStartTime) {
            // If our 60 second timer wasn't already started (e.g., if a refetch was pending when
            // the first page loads), we start it now if the host is online. If the host is offline,
            // we skip the refetch on page load.
            if (
              returnedHost.status === "online" ||
              isIPadOrIPhone(returnedHost.platform)
            ) {
              setRefetchStartTime(Date.now());
              setTimeout(() => {
                refetchHostDetails();
                refetchExtensions();
              }, 1000);
            } else {
              setShowRefetchSpinner(false);
            }
          } else {
            // !!refetchStartTime
            const totalElapsedTime = Date.now() - refetchStartTime;
            if (totalElapsedTime < 60000) {
              if (
                returnedHost.status === "online" ||
                isIPadOrIPhone(returnedHost.platform)
              ) {
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
              // totalElapsedTime > 60000
              renderFlash(
                "error",
                `We're having trouble fetching fresh vitals for this host. Please try again later.`
              );
              setShowRefetchSpinner(false);
            }
          }
          return; // exit early because refectch is pending so we can avoid unecessary steps below
        }
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
    IHostPastActivitiesResponse,
    Error,
    IHostPastActivitiesResponse,
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
    ({ queryKey: [{ pageIndex, perPage }] }) => {
      return activitiesAPI.getHostPastActivities(
        hostIdFromURL,
        pageIndex,
        perPage
      );
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
    IHostUpcomingActivitiesResponse,
    Error,
    IHostUpcomingActivitiesResponse,
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
    ({ queryKey: [{ pageIndex, perPage }] }) => {
      return activitiesAPI.getHostUpcomingActivities(
        hostIdFromURL,
        pageIndex,
        perPage
      );
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      keepPreviousData: true,
      staleTime: 2000,
    }
  );

  const featuresConfig = host?.team_id
    ? teams?.find((t) => t.id === host.team_id)?.features
    : config?.features;

  const getOSVersionRequirementFromMDMConfig = (hostPlatform: string) => {
    const mdmConfig = host?.team_id
      ? teams?.find((t) => t.id === host.team_id)?.mdm
      : config?.mdm;

    switch (hostPlatform) {
      case "darwin":
        return mdmConfig?.macos_updates;
      case "ipados":
        return mdmConfig?.ipados_updates;
      case "ios":
        return mdmConfig?.ios_updates;
      default:
        return undefined;
    }
  };

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
        router.push(PATHS.MANAGE_HOSTS);
        renderFlash(
          "success",
          `Host "${host.display_name}" was successfully deleted.`
        );
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
        renderFlash("error", getErrorMessage(error, host.display_name));
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
          setScriptExecutiontId(details?.script_execution_id || "");
          break;
        case "installed_software":
          setPackageInstallDetails({
            ...details,
            // FIXME: It seems like the backend is not using the correct display name when it returns
            // upcoming install activities. As a workaround, we'll prefer the display name from
            // the host object if it's available.
            host_display_name:
              host?.display_name || details?.host_display_name || "",
          });
          break;
        case "uninstalled_software":
          setPackageUninstallDetails({
            ...details,
            host_display_name:
              host?.display_name || details?.host_display_name || "",
          });
          break;
        case "installed_app_store_app":
          setAppInstallDetails({
            ...details,
            // FIXME: It seems like the backend is not using the correct display name when it returns
            // upcoming install activities. As a workaround, we'll prefer the display name from
            // the host object if it's available.
            host_display_name:
              host?.display_name || details?.host_display_name || "",
          });
          break;
        default: // do nothing
      }
    },
    [host?.display_name]
  );

  const onCancelActivity = (activity: IHostUpcomingActivity) => {
    setSelectedCancelActivity(activity);
  };

  const onLabelClick = (label: ILabel) => {
    return label.name === "All Hosts"
      ? router.push(PATHS.MANAGE_HOSTS)
      : router.push(PATHS.MANAGE_HOSTS_LABEL(label.id));
  };

  const onShowSoftwareDetails = useCallback(
    (software?: IHostSoftware) => {
      if (software) {
        setSelectedSoftwareDetails(software);
      }
    },
    [setSelectedSoftwareDetails]
  );

  const onShowUninstallDetails = useCallback(
    (uninstallScriptExecutionId?: string) => {
      if (uninstallScriptExecutionId) {
        setSelectedScriptExecutionId(uninstallScriptExecutionId);
      }
    },
    [setSelectedScriptExecutionId]
  );

  const onCancelRunScriptDetailsModal = useCallback(() => {
    setScriptExecutiontId("");
    // refetch activities to make sure they up-to-date with what was displayed in the modal
    refetchPastActivities();
    refetchUpcomingActivities();
  }, [refetchPastActivities, refetchUpcomingActivities]);

  const onCancelSoftwareInstallDetailsModal = useCallback(() => {
    setPackageInstallDetails(null);
  }, []);

  const onCancelAppInstallDetailsModal = useCallback(() => {
    setAppInstallDetails(null);
  }, []);

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

  const onCloseScriptModalGroup = useCallback(() => {
    setShowScriptModalGroup(false);
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
        setShowScriptModalGroup(true);
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

  const onSelectCertificate = (certificate: IHostCertificate) => {
    setSelectedCertificate(certificate);
  };

  const renderActionDropdown = () => {
    if (!host) {
      return null;
    }

    return (
      <HostActionsDropdown
        hostTeamId={host.team_id}
        onSelect={onSelectHostAction}
        hostPlatform={host.platform}
        hostStatus={host.status}
        hostMdmDeviceStatus={hostMdmDeviceStatus}
        hostMdmEnrollmentStatus={host.mdm.enrollment_status}
        doesStoreEncryptionKey={host.mdm.encryption_key_available}
        isConnectedToFleetMdm={host.mdm?.connected_to_fleet}
        hostScriptsEnabled={host.scripts_enabled}
      />
    );
  };

  const onSuccessCancelActivity = (activity: IHostUpcomingActivity) => {
    if (!host) return;

    // only for windows and linux hosts we want to refetch host details
    // after cancelling ran script activity. This is because lock and wipe
    // activites are run as scripts on windows and linux hosts.
    if (
      activity.type === ActivityType.RanScript &&
      (host.platform === "windows" || isLinuxLike(host.platform))
    ) {
      refetchHostDetails();
    }
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
      name: "Policies",
      title: "policies",
      pathname: PATHS.HOST_POLICIES(hostIdFromURL),
      count: failingPoliciesCount,
    },
  ];

  const hostSoftwareSubNav: IHostDetailsSubNavItem[] = [
    {
      name: "Inventory",
      title: "inventory",
      pathname: PATHS.HOST_INVENTORY(hostIdFromURL),
    },
    {
      name: "Library",
      title: "library",
      pathname: PATHS.HOST_LIBRARY(hostIdFromURL),
    },
  ];

  const getTabIndex = (path: string): number => {
    return hostDetailsSubNav.findIndex((navItem) => {
      // tab stays highlighted for paths that ends with same pathname
      return path.startsWith(navItem.pathname);
    });
  };

  const getSoftwareTabIndex = (path: string): number => {
    return hostSoftwareSubNav.findIndex((navItem) => {
      // tab stays highlighted for paths that ends with same pathname
      return path.endsWith(navItem.pathname);
    });
  };

  const navigateToNav = (i: number): void => {
    const navPath = hostDetailsSubNav[i].pathname;
    router.push(navPath);
  };

  const navigateToSoftwareTab = (i: number): void => {
    const navPath = hostSoftwareSubNav[i].pathname;
    router.push(navPath);
  };

  const isHostTeamAdmin = permissions.isTeamAdmin(currentUser, host?.team_id);
  const isHostTeamMaintainer = permissions.isTeamMaintainer(
    currentUser,
    host?.team_id
  );

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

  // host.platform = "windows";

  const isDarwinHost = host.platform === "darwin";
  const isIosOrIpadosHost = isIPadOrIPhone(host.platform);
  const isAndroidHost = isAndroid(host.platform);

  const showUsersCard =
    isDarwinHost ||
    generateChromeProfilesValues(host.end_users ?? []).length > 0 ||
    generateOtherEmailsValues(host.end_users ?? []).length > 0;
  const showActivityCard = !isAndroidHost;
  const showAgentOptionsCard = !isIosOrIpadosHost && !isAndroidHost;
  const showLocalUserAccountsCard = !isIosOrIpadosHost && !isAndroidHost;
  const showCertificatesCard =
    (isIosOrIpadosHost || isDarwinHost) &&
    !!hostCertificates?.certificates.length;

  const renderSoftwareCard = () => {
    return (
      <Card
        className={`${baseClass}__software-card`}
        borderRadiusSize="xxlarge"
        paddingSize="xlarge"
        includeShadow
      >
        {isPremiumTier ? (
          <>
            <TabList>
              <Tab>Inventory</Tab>
              <Tab>Library</Tab>
            </TabList>
            <TabPanel>
              <SoftwareInventoryCard
                id={host.id}
                platform={host.platform}
                softwareUpdatedAt={host.software_updated_at}
                isSoftwareEnabled={featuresConfig?.enable_software_inventory}
                router={router}
                queryParams={parseHostSoftwareQueryParams(location.query)}
                pathname={location.pathname}
                onShowSoftwareDetails={setSelectedSoftwareDetails}
                hostTeamId={host.team_id || 0}
              />
              {isDarwinHost && macadmins?.munki?.version && (
                <MunkiIssuesCard
                  isLoading={isLoadingHost}
                  munkiIssues={macadmins.munki_issues}
                  deviceType={host?.platform === "darwin" ? "macos" : ""}
                />
              )}
            </TabPanel>
            <TabPanel>
              <SoftwareLibraryCard
                id={host.id}
                platform={host.platform}
                softwareUpdatedAt={host.software_updated_at}
                hostScriptsEnabled={host.scripts_enabled || false}
                isSoftwareEnabled={featuresConfig?.enable_software_inventory}
                router={router}
                queryParams={{
                  ...parseHostSoftwareQueryParams(location.query),
                  available_for_install: true,
                }}
                pathname={location.pathname}
                onShowSoftwareDetails={onShowSoftwareDetails}
                onShowUninstallDetails={onShowUninstallDetails}
                hostTeamId={host.team_id || 0}
                hostMDMEnrolled={host.mdm.connected_to_fleet}
                isHostOnline={host.status === "online"}
              />
            </TabPanel>
          </>
        ) : (
          <>
            <SoftwareInventoryCard
              id={host.id}
              platform={host.platform}
              softwareUpdatedAt={host.software_updated_at}
              isSoftwareEnabled={featuresConfig?.enable_software_inventory}
              router={router}
              queryParams={parseHostSoftwareQueryParams(location.query)}
              pathname={location.pathname}
              onShowSoftwareDetails={setSelectedSoftwareDetails}
              hostTeamId={host.team_id || 0}
            />
            {isDarwinHost && macadmins?.munki?.version && (
              <MunkiIssuesCard
                isLoading={isLoadingHost}
                munkiIssues={macadmins.munki_issues}
                deviceType={host?.platform === "darwin" ? "macos" : ""}
              />
            )}
          </>
        )}
      </Card>
    );
  };

  return (
    <MainContent className={baseClass}>
      <>
        <HostDetailsBanners
          mdmEnrollmentStatus={host?.mdm.enrollment_status}
          hostPlatform={host?.platform}
          macDiskEncryptionStatus={host?.mdm.macos_settings?.disk_encryption}
          connectedToFleetMdm={host?.mdm.connected_to_fleet}
          diskEncryptionOSSetting={host?.mdm.os_settings?.disk_encryption}
          diskIsEncrypted={host?.disk_encryption_enabled}
          diskEncryptionKeyAvailable={host?.mdm.encryption_key_available}
        />
        <div className={`${baseClass}__header-links`}>
          <BackLink
            text="Back to all hosts"
            path={filteredHostsPath || PATHS.MANAGE_HOSTS}
          />
        </div>
        <div className={`${baseClass}__header-summary`}>
          <HostHeader
            summaryData={summaryData}
            showRefetchSpinner={showRefetchSpinner}
            onRefetchHost={onRefetchHost}
            renderActionDropdown={renderActionDropdown}
            hostMdmDeviceStatus={hostMdmDeviceStatus}
          />
        </div>
        <TabNav className={`${baseClass}__tab-nav`}>
          <Tabs
            selectedIndex={getTabIndex(location.pathname)}
            onSelect={(i) => navigateToNav(i)}
          >
            <TabList>
              {hostDetailsSubNav.map((navItem) => {
                // Bolding text when the tab is active causes a layout shift
                // so we add a hidden pseudo element with the same text string
                return (
                  <Tab key={navItem.title}>
                    <TabText count={navItem.count} isErrorCount>
                      {navItem.name}
                    </TabText>
                  </Tab>
                );
              })}
            </TabList>
            <TabPanel className={`${baseClass}__details-panel`}>
              <HostSummaryCard
                summaryData={summaryData}
                bootstrapPackageData={bootstrapPackageData}
                isPremiumTier={isPremiumTier}
                toggleOSSettingsModal={toggleOSSettingsModal}
                toggleBootstrapPackageModal={toggleBootstrapPackageModal}
                hostSettings={host?.mdm.profiles ?? []}
                osSettings={host?.mdm.os_settings}
                osVersionRequirement={getOSVersionRequirementFromMDMConfig(
                  host.platform
                )}
                className={fullWidthCardClass}
              />
              <AboutCard
                className={
                  showUsersCard ? defaultCardClass : fullWidthCardClass
                }
                aboutData={aboutData}
                munki={macadmins?.munki}
                mdm={mdm}
              />
              {showUsersCard && (
                <UserCard
                  className={defaultCardClass}
                  platform={host.platform}
                  endUsers={host.end_users ?? []}
                  enableAddEndUser={
                    isDarwinHost &&
                    generateUsernameValues(host.end_users ?? []).length === 0
                  }
                  onAddEndUser={() => setShowAddEndUserModal(true)}
                />
              )}
              {showActivityCard && (
                <ActivityCard
                  className={
                    showAgentOptionsCard
                      ? doubleHeightCardClass
                      : defaultCardClass
                  }
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
                  canCancelActivities={
                    isGlobalAdmin ||
                    isGlobalMaintainer ||
                    isHostTeamAdmin ||
                    isHostTeamMaintainer
                  }
                  upcomingCount={upcomingActivities?.count || 0}
                  onChangeTab={onChangeActivityTab}
                  onNextPage={() => setActivityPage(activityPage + 1)}
                  onPreviousPage={() => setActivityPage(activityPage - 1)}
                  onShowDetails={onShowActivityDetails}
                  onCancel={onCancelActivity}
                />
              )}
              {showAgentOptionsCard && (
                <AgentOptionsCard
                  className={defaultCardClass}
                  osqueryData={osqueryData}
                  wrapFleetHelper={wrapFleetHelper}
                  isChromeOS={host?.platform === "chrome"}
                />
              )}
              <LabelsCard
                className={
                  !showActivityCard && !showAgentOptionsCard
                    ? fullWidthCardClass
                    : defaultCardClass
                }
                labels={host?.labels || []}
                onLabelClick={onLabelClick}
              />
              {showLocalUserAccountsCard && (
                <LocalUserAccountsCard
                  className={fullWidthCardClass}
                  users={host?.users || []}
                  usersState={usersState}
                  isLoading={isLoadingHost}
                  onUsersTableSearchChange={onUsersTableSearchChange}
                  hostUsersEnabled={featuresConfig?.enable_host_users}
                />
              )}
              {showCertificatesCard && (
                <CertificatesCard
                  className={fullWidthCardClass}
                  data={hostCertificates}
                  hostPlatform={host.platform}
                  onSelectCertificate={onSelectCertificate}
                  isError={isErrorHostCertificates}
                  page={certificatePage}
                  pageSize={DEFAULT_CERTIFICATES_PAGE_SIZE}
                  onNextPage={() => setCertificatePage(certificatePage + 1)}
                  onPreviousPage={() => setCertificatePage(certificatePage - 1)}
                  sortDirection={sortCerts.order_direction}
                  sortHeader={sortCerts.order_key}
                  onSortChange={setSortCerts}
                />
              )}
            </TabPanel>
            <TabPanel>
              <TabNav className={`${baseClass}__software-tab-nav`}>
                <Tabs
                  selectedIndex={getSoftwareTabIndex(location.pathname)}
                  onSelect={(i) => navigateToSoftwareTab(i)}
                >
                  {renderSoftwareCard()}
                </Tabs>
              </TabNav>
            </TabPanel>
            <TabPanel>
              <QueriesCard
                hostId={host.id}
                router={router}
                hostPlatform={host.platform}
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
                hostPlatform={host.platform}
                router={router}
                currentTeamId={currentTeam?.id}
              />
            </TabPanel>
          </Tabs>
        </TabNav>
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
            isOnlyObserver={isOnlyObserver}
            hostId={hostIdFromURL}
            hostTeamId={host?.team_id}
            router={router}
            currentTeamId={currentTeam?.id}
          />
        )}
        {showScriptModalGroup && (
          <ScriptModalGroup
            host={host}
            currentUser={currentUser}
            onCloseScriptModalGroup={onCloseScriptModalGroup}
            teamIdForApi={currentTeam?.id}
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
            canResendProfiles={host.platform === "darwin"}
            hostId={host.id}
            platform={host.platform}
            hostMDMData={host.mdm}
            onClose={toggleOSSettingsModal}
            onProfileResent={refetchHostDetails}
          />
        )}
        {showUnenrollMdmModal && !!host && (
          <UnenrollMdmModal
            hostId={host.id}
            hostPlatform={host.platform}
            hostName={host.display_name}
            onClose={toggleUnenrollMdmModal}
          />
        )}
        {showDiskEncryptionModal && host && (
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
        {scriptExecutionId && (
          <RunScriptDetailsModal
            scriptExecutionId={scriptExecutionId}
            onCancel={onCancelRunScriptDetailsModal}
          />
        )}
        {!!packageInstallDetails && (
          <SoftwareInstallDetailsModal
            details={packageInstallDetails}
            onCancel={onCancelSoftwareInstallDetailsModal}
          />
        )}
        {packageUninstallDetails && (
          <SoftwareUninstallDetailsModal
            details={packageUninstallDetails}
            onCancel={() => setPackageUninstallDetails(null)}
          />
        )}
        {!!appInstallDetails && (
          <AppInstallDetailsModal
            details={appInstallDetails}
            onCancel={onCancelAppInstallDetailsModal}
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
        {selectedSoftwareDetails && (
          <SoftwareDetailsModal
            hostDisplayName={host.display_name}
            software={selectedSoftwareDetails}
            onExit={() => setSelectedSoftwareDetails(null)}
          />
        )}
        {selectedScriptExecutionId && !!host && (
          <SoftwareUninstallDetailsModal
            details={{
              host_display_name: host.display_name,
              script_execution_id: selectedScriptExecutionId,
              status: "failed_uninstall",
            }}
            onCancel={() => setSelectedScriptExecutionId(undefined)}
          />
        )}
        {selectedCancelActivity && (
          <CancelActivityModal
            hostId={host.id}
            activity={selectedCancelActivity}
            onCancelActivity={() => refetchUpcomingActivities()}
            onSuccessCancel={onSuccessCancelActivity}
            onExit={() => setSelectedCancelActivity(null)}
          />
        )}
        {selectedCertificate && (
          <CertificateDetailsModal
            certificate={selectedCertificate}
            onExit={() => setSelectedCertificate(null)}
          />
        )}
      </>
      {showAddEndUserModal && (
        <AddEndUserModal onExit={() => setShowAddEndUserModal(false)} />
      )}
    </MainContent>
  );
};

export default HostDetailsPage;
