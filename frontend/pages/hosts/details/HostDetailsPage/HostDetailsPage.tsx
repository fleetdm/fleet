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
import commandAPI from "services/entities/command";

import { IHost, IMacadminsResponse, IHostResponse } from "interfaces/host";
import { ILabel } from "interfaces/label";
import { IListSort } from "interfaces/list_options";
import { IHostPolicy } from "interfaces/policy";
import {
  IHostSoftware,
  resolveUninstallStatus,
  SCRIPT_PACKAGE_SOURCES,
  SoftwareInstallUninstallStatus,
} from "interfaces/software";
import { ITeam } from "interfaces/team";
import { ActivityType, IHostUpcomingActivity } from "interfaces/activity";
import {
  IHostCertificate,
  CERTIFICATES_DEFAULT_SORT,
} from "interfaces/certificates";
import {
  isBYODAccountDrivenUserEnrollment,
  FLEET_FILEVAULT_PROFILE_DISPLAY_NAME,
} from "interfaces/mdm";
import { ICommand } from "interfaces/command";

import { normalizeEmptyValues, wrapFleetHelper } from "utilities/helpers";
import permissions from "utilities/permissions";
import {
  DOCUMENT_TITLE_SUFFIX,
  HOST_SUMMARY_DATA,
  HOST_VITALS_DATA,
  HOST_OSQUERY_DATA,
  DEFAULT_USE_QUERY_OPTIONS,
} from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";

import {
  isAppleDevice,
  isMacOS,
  isAndroid,
  isIPadOrIPhone,
  isLinuxLike,
  isWindows,
} from "interfaces/platform";

import Spinner from "components/Spinner";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import MainContent, { IMainContentConfig } from "components/MainContent";
import BackButton from "components/BackButton";
import CustomLink from "components/CustomLink/CustomLink";
import EmptyState from "components/EmptyState";

import RunScriptDetailsModal from "pages/DashboardPage/cards/ActivityFeed/components/RunScriptDetailsModal";
import {
  VppInstallDetailsModal,
  IVppInstallDetails,
} from "components/ActivityDetails/InstallDetails/VppInstallDetailsModal/VppInstallDetailsModal";
import {
  SoftwareInstallDetailsModal,
  IPackageInstallDetails,
} from "components/ActivityDetails/InstallDetails/SoftwareInstallDetailsModal/SoftwareInstallDetailsModal";
import { SoftwareScriptDetailsModal } from "components/ActivityDetails/InstallDetails/SoftwareScriptDetailsModal/SoftwareScriptDetailsModal";
import {
  SoftwareIpaInstallDetailsModal,
  ISoftwareIpaInstallDetails,
} from "components/ActivityDetails/InstallDetails/SoftwareIpaInstallDetailsModal/SoftwareIpaInstallDetailsModal";
import SoftwareUninstallDetailsModal, {
  ISWUninstallDetailsParentState,
} from "components/ActivityDetails/InstallDetails/SoftwareUninstallDetailsModal/SoftwareUninstallDetailsModal";
import { IShowActivityDetailsData } from "components/ActivityItem/ActivityItem";
import CertificateInstallDetailsModal, {
  ICertificateInstallDetails,
} from "components/ActivityDetails/InstallDetails/CertificateInstallDetailsModal";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";

import CommandResultsModal from "pages/hosts/components/CommandDetailsModal";
import FailedEnrollmentProfileModal, {
  IFailedEnrollmentProfileModalProps,
} from "components/modals/FailedEnrollmentProfileModal";

import HostSummaryCard from "../cards/HostSummary";
import VitalsCard from "../cards/Vitals";
import UserCard from "../cards/User";
import ActivityCard from "../cards/Activity";
import AgentOptionsCard from "../cards/AgentOptions";
import LabelsCard from "../cards/Labels";
import MunkiIssuesCard from "../cards/MunkiIssues";
import SoftwareInventoryCard from "../cards/Software";
import SoftwareLibraryCard from "../cards/HostSoftwareLibrary";
import LocalUserAccountsCard from "../cards/LocalUserAccounts";
import PoliciesCard from "../cards/Policies";
import PolicyDetailsModal from "../cards/Policies/HostPoliciesTable/PolicyDetailsModal";
import HostReportsTab from "../HostReportsTab";
import CertificatesCard from "../cards/Certificates";

import TransferHostModal from "../../components/TransferHostModal";
import DeleteHostModal from "../../components/DeleteHostModal";

import UnenrollMdmModal from "./modals/UnenrollMdmModal";
import DiskEncryptionKeyModal from "./modals/DiskEncryptionKeyModal";
import RecoveryLockPasswordModal from "./modals/RecoveryLockPasswordModal";
import ManagedAccountModal from "./modals/ManagedAccountModal";
import HostActionsDropdown from "./HostActionsDropdown/HostActionsDropdown";
import OSSettingsModal from "../OSSettingsModal";
import BootstrapPackageModal from "./modals/BootstrapPackageModal";
import ScriptModalGroup from "./modals/ScriptModalGroup";
import SelectReportModal from "./modals/SelectReportModal";
import HostDetailsBanners from "./components/HostDetailsBanners";
import LockModal from "./modals/LockModal";
import UnlockModal from "./modals/UnlockModal";
import {
  HostMdmDeviceStatusUIState,
  getHostDeviceStatusUIState,
} from "../helpers";
import WipeModal from "./modals/WipeModal";
import { parseHostSoftwareQueryParams } from "../cards/Software/HostSoftware";
import { getErrorMessage } from "./helpers";
import CancelActivityModal from "./modals/CancelActivityModal";
import CertificateDetailsModal from "../modals/CertificateDetailsModal";
import HostHeader from "../cards/HostHeader";
import InventoryVersionsModal from "../modals/InventoryVersionsModal";
import UpdateEndUserModal from "../cards/User/components/UpdateEndUserModal";
import LocationModal from "../modals/LocationModal";
import MDMStatusModal from "../modals/MDMStatusModal";
import ClearPasscodeModal from "./modals/ClearPasscodeModal";

const baseClass = "host-details";

const defaultCardClass = `${baseClass}__card`;
const fullWidthCardClass = `${baseClass}__card--full-width`;
const tripleHeightCardClass = `${baseClass}__card--triple-height`;

export const REFETCH_HOST_DETAILS_POLLING_INTERVAL = 2000; // 2 seconds
const BYOD_SW_INSTALL_LEARN_MORE_LINK =
  "https://fleetdm.com/learn-more-about/byod-hosts-vpp-install";
const ANDROID_SW_INSTALL_LEARN_MORE_LINK =
  "https://fleetdm.com/learn-more-about/install-google-play-apps";

const ACTIVITY_CARD_DATA_STALE_TIME = 5000; // 5 seconds

interface IHostDetailsProps {
  router: InjectedRouter; // v3
  location: {
    pathname: string;
    query: {
      page?: string;
      query?: string;
      order_key?: string;
      order_direction?: "asc" | "desc";
      fleet_id?: string;
      show_mdm_status?: string;
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
    isGlobalTechnician,
    isTeamMaintainerOrTeamAdmin,
    isPremiumTier = false,
    isOnlyObserver,
    filteredHostsPath,
    currentTeam,
    isMacMdmEnabledAndConfigured,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const handlePageError = useErrorHandler();

  const [showDeleteHostModal, setShowDeleteHostModal] = useState(false);
  const [showTransferHostModal, setShowTransferHostModal] = useState(false);
  const [showSelectReportModal, setShowSelectReportModal] = useState(false);
  const [showScriptModalGroup, setShowScriptModalGroup] = useState(false);
  const [showPolicyDetailsModal, setPolicyDetailsModal] = useState(false);
  const [showOSSettingsModal, setShowOSSettingsModal] = useState(false);
  const [showUnenrollMdmModal, setShowUnenrollMdmModal] = useState(false);
  const [showDiskEncryptionModal, setShowDiskEncryptionModal] = useState(false);
  const [
    showRecoveryLockPasswordModal,
    setShowRecoveryLockPasswordModal,
  ] = useState(false);
  const [showManagedAccountModal, setShowManagedAccountModal] = useState(false);
  const [showBootstrapPackageModal, setShowBootstrapPackageModal] = useState(
    false
  );
  const [showLockHostModal, setShowLockHostModal] = useState(false);
  const [showUnlockHostModal, setShowUnlockHostModal] = useState(false);
  const [showWipeModal, setShowWipeModal] = useState(false);
  const [showUpdateEndUserModal, setShowUpdateEndUserModal] = useState(false);
  // Undefined used to return to true after closing the lock modal
  const [showLocationModal, setShowLocationModal] = useState<
    boolean | undefined
  >(false);
  const [showMDMStatusModal, setShowMDMStatusModal] = useState(
    location.query.show_mdm_status === "true"
  );
  // Sync MDM status modal state when the query param changes while mounted
  // (e.g., browser back/forward navigation).
  useEffect(() => {
    setShowMDMStatusModal(location.query.show_mdm_status === "true");
  }, [location.query.show_mdm_status]);

  const [showClearPasscodeModal, setShowClearPasscodeModal] = useState(false);

  // General-use updating state
  const [isUpdating, setIsUpdating] = useState(false);

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
    scriptPackageDetails,
    setScriptPackageDetails,
  ] = useState<IPackageInstallDetails | null>(null);
  const [
    ipaPackageInstallDetails,
    setIpaPackageInstallDetails,
  ] = useState<ISoftwareIpaInstallDetails | null>(null);
  const [
    packageUninstallDetails,
    setPackageUninstallDetails,
  ] = useState<ISWUninstallDetailsParentState | null>(null);
  const [
    activityVPPInstallDetails,
    setActivityVPPInstallDetails,
  ] = useState<IVppInstallDetails | null>(null);
  const [
    certificateInstallDetails,
    setCertificateInstallDetails,
  ] = useState<ICertificateInstallDetails | null>(null);
  const [mdmCommandDetails, setMdmCommandDetails] = useState<ICommand | null>(
    null
  );
  const [
    enrollmentProfileFailedDetails,
    setEnrollmentProfileFailedDetails,
  ] = useState<Omit<IFailedEnrollmentProfileModalProps, "onDone"> | null>(null);

  const [refetchStartTime, setRefetchStartTime] = useState<number | null>(null);
  const [showRefetchSpinner, setShowRefetchSpinner] = useState(false);
  const [usersState, setUsersState] = useState<{ username: string }[]>([]);
  const [usersSearchString, setUsersSearchString] = useState("");
  const [
    hostMdmDeviceStatus,
    setHostMdmDeviceState,
  ] = useState<HostMdmDeviceStatusUIState>("unlocked");
  const [
    selectedHostSWForInventoryVersions,
    setSelectedHostSWForInventoryVersions,
  ] = useState<IHostSoftware | null>(null);
  const [
    selectedCancelActivity,
    setSelectedCancelActivity,
  ] = useState<IHostUpcomingActivity | null>(null);

  // activity states
  const [activeActivityTab, setActiveActivityTab] = useState<
    "past" | "upcoming"
  >("past");
  const [activityPage, setActivityPage] = useState(0);
  const [showMDMCommands, setShowMDMCommands] = useState(false);

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

  const { data: teams } = useQuery<ILoadTeamsResponse, Error, ITeam[]>(
    "teams",
    () => teamAPI.loadAll(),
    {
      enabled: !!hostIdFromURL && !!isPremiumTier,
      retry: false,
      select: (data: ILoadTeamsResponse) => data.teams,
    }
  );

  const { data: macadmins, refetch: refetchMacadmins } = useQuery(
    ["macadmins", hostIdFromURL],
    () => hostAPI.loadHostDetailsExtension(hostIdFromURL, "macadmins"),
    {
      enabled: !!hostIdFromURL, // TODO(android): disable for unsupported platforms?
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
    hostCertificates && refetchHostCertificates();
  };

  /**
   * Hides refetch spinner and resets refetch timer,
   * ensuring no stale timeout triggers on new requests.
   */
  const resetHostRefetchStates = () => {
    setShowRefetchSpinner(false);
    setRefetchStartTime(null);
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
      retry: false,
      select: (data: IHostResponse) => data.host,
      onSuccess: (returnedHost) => {
        // If API returns refetch_requested: true,
        // only set timer if *not* already set!
        if (returnedHost.refetch_requested) {
          if (!refetchStartTime) {
            setRefetchStartTime(Date.now());
          }
          setShowRefetchSpinner(true);

          // If Android, don't run timers/polling logic
          if (!isAndroid(returnedHost.platform)) {
            // Compute how long since timer started (if set)
            const totalElapsedTime = refetchStartTime
              ? Date.now() - refetchStartTime
              : 0;
            if (!refetchStartTime) {
              // Timer just started - poll again after interval!
              if (
                returnedHost.status === "online" ||
                isIPadOrIPhone(returnedHost.platform)
              ) {
                setTimeout(() => {
                  refetchHostDetails();
                  refetchExtensions();
                }, REFETCH_HOST_DETAILS_POLLING_INTERVAL);
              } else {
                resetHostRefetchStates();
              }
            } else if (totalElapsedTime < 60000) {
              // Timer running, still inside poll window
              if (
                returnedHost.status === "online" ||
                isIPadOrIPhone(returnedHost.platform)
              ) {
                setTimeout(() => {
                  refetchHostDetails();
                  refetchExtensions();
                }, REFETCH_HOST_DETAILS_POLLING_INTERVAL);
              } else {
                renderFlash(
                  "error",
                  `This host is offline. Please try refetching host vitals later.`
                );
                resetHostRefetchStates();
              }
            } else {
              // Total elapsed poll window exceeded (60s), stop and alert
              renderFlash(
                "error",
                `We're having trouble fetching fresh vitals for this host. Please try again later.`
              );
              resetHostRefetchStates();
            }
          }
        } else {
          // Not refetching: reset spinner and timer
          resetHostRefetchStates();
        }

        setHostMdmDeviceState(
          getHostDeviceStatusUIState(
            returnedHost.mdm.device_status,
            returnedHost.mdm.pending_action
          )
        );
        setUsersState(returnedHost.users || []);
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
      hostId: number;
      pageIndex: number;
      perPage: number;
      activeTab: "past" | "upcoming";
    }>
  >(
    [
      {
        scope: "past-activities",
        hostId: hostIdFromURL,
        pageIndex: activityPage,
        perPage: DEFAULT_ACTIVITY_PAGE_SIZE,
        activeTab: activeActivityTab,
      },
    ],
    ({ queryKey: [{ hostId, pageIndex, perPage }] }) => {
      return activitiesAPI.getHostPastActivities(hostId, pageIndex, perPage);
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      keepPreviousData: true,
      staleTime: ACTIVITY_CARD_DATA_STALE_TIME,
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
      hostId: number;
      pageIndex: number;
      perPage: number;
      activeTab: "past" | "upcoming";
    }>
  >(
    [
      {
        scope: "upcoming-activities",
        hostId: hostIdFromURL,
        pageIndex: activityPage,
        perPage: DEFAULT_ACTIVITY_PAGE_SIZE,
        activeTab: activeActivityTab,
      },
    ],
    ({ queryKey: [{ hostId, pageIndex, perPage }] }) => {
      return activitiesAPI.getHostUpcomingActivities(
        hostId,
        pageIndex,
        perPage
      );
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      keepPreviousData: true,
      staleTime: ACTIVITY_CARD_DATA_STALE_TIME,
    }
  );

  const mdmConfig = host?.team_id
    ? teams?.find((t) => t.id === host.team_id)?.mdm
    : config?.mdm;

  // We must check if the host has a UUID. Not-yet-enrolled hosts synced over from ABM will have
  // a pending MDM status but no UUID, so there are no commands and no way to fetch them
  const canGetMDMCommands =
    !!isMacMdmEnabledAndConfigured &&
    isAppleDevice(host?.platform) &&
    !!host?.uuid;

  const {
    data: pastMDMCommands,
    isError: pastMDMCommandsIsError,
    isFetching: pastMDMCommandsIsFetching,
    isLoading: pastMDMCommandsIsLoading,
  } = useQuery(
    [
      {
        scope: "host-past-mdm-commands",
        pageIndex: activityPage,
        perPage: DEFAULT_ACTIVITY_PAGE_SIZE,
        hostUUID: host?.uuid,
        activeTab: activeActivityTab,
        commandStatus: "ran,failed",
      },
    ],
    ({ queryKey: [{ pageIndex, perPage, hostUUID, commandStatus }] }) => {
      return commandAPI.getCommands({
        page: pageIndex,
        per_page: perPage,
        host_identifier: hostUUID,
        command_status: commandStatus,
      });
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: canGetMDMCommands,
      keepPreviousData: true,
      staleTime: ACTIVITY_CARD_DATA_STALE_TIME,
    }
  );

  const {
    data: upcomingMDMCommands,
    isError: upcomingMDMCommandsIsError,
    isFetching: upcomingMDMCommandsIsFetching,
    isLoading: upcomingMDMCommandsIsLoading,
  } = useQuery(
    [
      {
        scope: "host-upcoming-mdm-commands",
        pageIndex: activityPage,
        perPage: DEFAULT_ACTIVITY_PAGE_SIZE,
        hostUUID: host?.uuid,
        activeTab: activeActivityTab,
        commandStatus: "pending",
      },
    ],
    ({ queryKey: [{ pageIndex, perPage, hostUUID, commandStatus }] }) => {
      return commandAPI.getCommands({
        page: pageIndex,
        per_page: perPage,
        host_identifier: hostUUID,
        command_status: commandStatus,
      });
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: canGetMDMCommands,
      keepPreviousData: true,
      staleTime: ACTIVITY_CARD_DATA_STALE_TIME,
    }
  );

  const featuresConfig = host?.team_id
    ? teams?.find((t) => t.id === host.team_id)?.features
    : config?.features;

  const getOSVersionRequirementFromMDMConfig = (hostPlatform: string) => {
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
        host?.users?.filter((user) => {
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

  const vitalsData = normalizeEmptyValues(pick(host, HOST_VITALS_DATA));

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

  const toggleLocationModal = useCallback(() => {
    setShowLocationModal(!showLocationModal);
  }, [showLocationModal, setShowLocationModal]);

  const toggleMDMStatusModal = useCallback(() => {
    setShowMDMStatusModal((prev) => {
      const closing = prev;
      // When closing, strip ?show_mdm_status=true so that refreshing or
      // sharing the URL won't reopen the modal. Opening never adds the param
      // — it's only set by external deep-links.
      if (closing && location.query.show_mdm_status === "true") {
        const { show_mdm_status: _, ...rest } = location.query;
        router.replace({ pathname: location.pathname, query: rest });
      }
      return !prev;
    });
  }, [location, router]);

  const toggleClearPasscodeModal = useCallback(() => {
    setShowClearPasscodeModal(!showClearPasscodeModal);
  }, [showClearPasscodeModal, setShowClearPasscodeModal]);

  const onCancelPolicyDetailsModal = useCallback(() => {
    setPolicyDetailsModal(!showPolicyDetailsModal);
    setSelectedPolicy(null);
  }, [showPolicyDetailsModal, setPolicyDetailsModal, setSelectedPolicy]);

  const toggleUnenrollMdmModal = useCallback(() => {
    setShowUnenrollMdmModal(!showUnenrollMdmModal);
  }, [showUnenrollMdmModal, setShowUnenrollMdmModal]);

  const onDestroyHost = async () => {
    if (host) {
      setIsUpdating(true);
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
        setIsUpdating(false);
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
          }, REFETCH_HOST_DETAILS_POLLING_INTERVAL);
        });
      } catch (error) {
        renderFlash("error", getErrorMessage(error, host.display_name));
        resetHostRefetchStates();
      }
    }
  };

  const resendProfile = useCallback(
    (profileUUID: string): Promise<void> => {
      if (!host?.id) {
        return Promise.resolve();
      }
      return hostAPI.resendProfile(host.id, profileUUID);
    },
    [host?.id]
  );

  const resendCertificate = useCallback(
    (certificateTemplateId: number): Promise<void> => {
      if (!host?.id) {
        return Promise.resolve();
      }
      return hostAPI.resendCertificate(host.id, certificateTemplateId);
    },
    [host?.id]
  );

  const rotateRecoveryLockPassword = useCallback((): Promise<void> => {
    if (!host?.id) {
      return new Promise(() => undefined);
    }
    return hostAPI.rotateRecoveryLockPassword(host.id);
  }, [host?.id]);

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
          if (details?.command_uuid) {
            setIpaPackageInstallDetails({
              fleetInstallStatus: details?.status as SoftwareInstallUninstallStatus,
              hostDisplayName:
                host?.display_name || details?.host_display_name || "",
              appName: getDisplayedSoftwareName(
                details.software_title,
                details.software_display_name
              ),
              commandUuid: details?.command_uuid,
            });
          } else if (SCRIPT_PACKAGE_SOURCES.includes(details?.source || "")) {
            setScriptPackageDetails({
              ...details,
              // FIXME: It seems like the backend is not using the correct display name when it returns
              // upcoming install activities. As a workaround, we'll prefer the display name from
              // the host object if it's available.
              host_display_name:
                host?.display_name || details?.host_display_name || "",
            });
          } else {
            setPackageInstallDetails({
              ...details,
              // FIXME: It seems like the backend is not using the correct display name when it returns
              // upcoming install activities. As a workaround, we'll prefer the display name from
              // the host object if it's available.
              host_display_name:
                host?.display_name || details?.host_display_name || "",
            });
          }
          break;
        case "uninstalled_software":
          setPackageUninstallDetails({
            ...details,
            softwareName: getDisplayedSoftwareName(
              details?.software_title,
              details?.software_display_name
            ),
            uninstallStatus: resolveUninstallStatus(details?.status),
            scriptExecutionId: details?.script_execution_id || "",
            hostDisplayName: host?.display_name || details?.host_display_name,
          });
          break;
        case "installed_app_store_app":
          setActivityVPPInstallDetails({
            appName: getDisplayedSoftwareName(
              details?.software_title,
              details?.software_display_name
            ),
            fleetInstallStatus: (details?.status ||
              "pending_install") as SoftwareInstallUninstallStatus,
            commandUuid: details?.command_uuid || "",
            // FIXME: It seems like the backend is not using the correct display name when it returns
            // upcoming install activities. As a workaround, we'll prefer the display name from
            // the host object if it's available.
            hostDisplayName:
              host?.display_name || details?.host_display_name || "",
            platform: details?.host_platform || host?.platform,
          });
          break;
        case "installed_certificate":
          setCertificateInstallDetails({
            certificateName: details?.certificate_name || "",
            hostDisplayName:
              host?.display_name || details?.host_display_name || "",
            status: details?.status || "",
            detail: details?.detail || "",
          });
          break;
        case ActivityType.FailedEnrollmentProfileRenewal:
          setEnrollmentProfileFailedDetails({
            command: {
              command_uuid: details?.command_uuid || "",
            },
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

  const onSetSelectedHostSWForInventoryVersions = useCallback(
    (hostSW?: IHostSoftware) => {
      if (hostSW) {
        setSelectedHostSWForInventoryVersions(hostSW);
      }
    },
    [setSelectedHostSWForInventoryVersions]
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

  const onCancelIpaSoftwareInstallDetailsModal = useCallback(() => {
    setIpaPackageInstallDetails(null);
  }, []);

  const onCancelVppInstallDetailsModal = useCallback(() => {
    setActivityVPPInstallDetails(null);
  }, []);

  const onCancelMdmCommandDetailsModal = useCallback(() => {
    setMdmCommandDetails(null);
  }, []);

  const onTransferHostSubmit = async (team: ITeam) => {
    setIsUpdating(true);

    const teamId = typeof team.id === "number" ? team.id : null;

    try {
      await hostAPI.transferToTeam(teamId, [hostIdFromURL]);

      const successMessage =
        teamId === null
          ? `Host successfully removed from fleets.`
          : `Host successfully transferred to  ${team.name}.`;

      renderFlash("success", successMessage);
      refetchHostDetails(); // Note: it is not necessary to `refetchExtensions` here because only team has changed
      setShowTransferHostModal(false);
    } catch (error) {
      renderFlash("error", "Could not transfer host. Please try again.");
    } finally {
      setIsUpdating(false);
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
        setShowSelectReportModal(true);
        break;
      case "diskEncryption":
        setShowDiskEncryptionModal(true);
        break;
      case "recoveryLockPassword":
        setShowRecoveryLockPasswordModal(true);
        break;
      case "managedAccount":
        setShowManagedAccountModal(true);
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
      case "clearPasscode":
        setShowClearPasscodeModal(true);
        break;
      default: // do nothing
    }
  };

  const onSelectCertificate = (certificate: IHostCertificate) => {
    setSelectedCertificate(certificate);
  };

  const renderActionsDropdown = () => {
    if (!host) {
      return null;
    }

    const diskEncryptionProfile = host.mdm.profiles?.find(
      (p) => p.name === FLEET_FILEVAULT_PROFILE_DISPLAY_NAME
    );

    return (
      <HostActionsDropdown
        hostTeamId={host.team_id}
        onSelect={onSelectHostAction}
        hostPlatform={host.platform}
        hostCpuType={host.cpu_type}
        hostStatus={host.status}
        hostMdmDeviceStatus={hostMdmDeviceStatus}
        hostMdmEnrollmentStatus={host.mdm.enrollment_status}
        doesStoreEncryptionKey={
          host.mdm.encryption_key_available ||
          !!host.mdm.encryption_key_archived
        }
        isConnectedToFleetMdm={host.mdm?.connected_to_fleet}
        hostScriptsEnabled={host.scripts_enabled}
        isRecoveryLockPasswordEnabled={
          mdmConfig?.enable_recovery_lock_password ?? false
        }
        diskEncryptionProfileStatus={diskEncryptionProfile?.status}
        recoveryLockPasswordAvailable={
          host.mdm.os_settings?.recovery_lock_password?.password_available ??
          false
        }
        isManagedLocalAccountEnabled={
          mdmConfig?.macos_setup?.enable_managed_local_account ?? false
        }
        managedAccountStatus={
          host.mdm.os_settings?.managed_local_account?.status
        }
        managedAccountPasswordAvailable={
          host.mdm.os_settings?.managed_local_account?.password_available ??
          false
        }
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

  const onUpdateEndUser = async (username: string) => {
    setIsUpdating(true);
    try {
      if (username === "") {
        await hostAPI.deleteHostIdp(hostIdFromURL);
        renderFlash("success", "Removed end user.");
      } else {
        await hostAPI.updateHostIdp(hostIdFromURL, username);
        renderFlash("success", "Updated end user.");
      }
      setShowUpdateEndUserModal(false);
      refetchHostDetails();
    } catch (e) {
      renderFlash("error", "Could not update end user. Please try again.");
    } finally {
      setIsUpdating(false);
    }
  };

  if (
    !host ||
    isLoadingHost ||
    pastActivitiesIsLoading ||
    upcomingActivitiesIsLoading ||
    pastMDMCommandsIsLoading ||
    upcomingMDMCommandsIsLoading
  ) {
    return <Spinner />;
  }
  const failingPoliciesCount = host?.issues.failing_policies_count || 0;

  const isMacOSHost = isMacOS(host.platform);
  const isIosOrIpadosHost = isIPadOrIPhone(host.platform);
  const isAndroidHost = isAndroid(host.platform);
  const isWindowsHost = isWindows(host.platform);
  const isAppleDeviceHost = isAppleDevice(host.platform);
  const isChromeOsHost = host?.platform === "chrome";

  const showReportsTab =
    !isIosOrIpadosHost && !isAndroidHost && !isChromeOsHost;

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
    // Only include Reports for supported platforms
    ...(showReportsTab
      ? [
          {
            name: "Reports",
            title: "reports",
            pathname: PATHS.HOST_REPORTS(hostIdFromURL),
          },
        ]
      : []),
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
    const selected = hostDetailsSubNav.findIndex((navItem) => {
      // tab stays highlighted for paths that ends with same pathname
      return path.startsWith(navItem.pathname);
    });
    // If our URL doesn't match anything, return the first (Details) tab by default
    return selected === -1 ? 0 : selected;
  };

  const getSoftwareTabIndex = (path: string): number => {
    return hostSoftwareSubNav.findIndex((navItem) => {
      // tab stays highlighted for paths that ends with same pathname
      return path.endsWith(navItem.pathname);
    });
  };

  const navigateToNav = (i: number): void => {
    const navPath = hostDetailsSubNav[i].pathname;
    router.push(
      getPathWithQueryParams(navPath, {
        fleet_id: currentTeam?.id || location.query.fleet_id,
      })
    );
  };
  const navigateToSoftwareTab = (i: number): void => {
    const navPath = hostSoftwareSubNav[i].pathname;
    router.push(
      getPathWithQueryParams(navPath, {
        fleet_id: currentTeam?.id || location.query.fleet_id,
      })
    );
  };

  const isHostTeamAdmin = permissions.isTeamAdmin(currentUser, host?.team_id);
  const isHostTeamMaintainer = permissions.isTeamMaintainer(
    currentUser,
    host?.team_id
  );
  const isHostTeamTechnician = permissions.isTeamTechnician(
    currentUser,
    host?.team_id
  );

  const bootstrapPackageData = {
    status: host?.mdm.setup_experience?.bootstrap_package_status,
    details: host?.mdm.setup_experience?.details,
    name: host?.mdm.setup_experience?.bootstrap_package_name,
  };

  const canResendProfiles =
    (isAppleDeviceHost || isWindowsHost || isAndroidHost) &&
    (isGlobalAdmin ||
      isGlobalMaintainer ||
      isGlobalTechnician ||
      isHostTeamAdmin ||
      isHostTeamMaintainer ||
      isHostTeamTechnician);

  const showSoftwareLibraryTab = isPremiumTier;
  const showReportsEmptyState = host.mdm?.enrollment_status === "Pending";
  const showAgentOptionsCard = !isIosOrIpadosHost && !isAndroidHost;
  const showLocalUserAccountsCard = !isIosOrIpadosHost && !isAndroidHost;
  const showCertificatesCard =
    isAppleDeviceHost && !!hostCertificates?.certificates.length;

  const renderSoftwareCard = () => {
    return (
      <div className={`${baseClass}__software-card`}>
        {showSoftwareLibraryTab ? (
          <>
            <TabList>
              <Tab>
                <TabText>Inventory</TabText>
              </Tab>
              <Tab>
                <TabText>Library</TabText>
              </Tab>
            </TabList>
            <TabPanel>
              <SoftwareInventoryCard
                id={host.id}
                platform={host.platform}
                softwareUpdatedAt={host.software_updated_at}
                isSoftwareEnabled={featuresConfig?.enable_software_inventory}
                router={router}
                queryParams={{
                  ...parseHostSoftwareQueryParams(location.query),
                  include_available_for_install: false,
                }}
                pathname={location.pathname}
                onShowInventoryVersions={
                  onSetSelectedHostSWForInventoryVersions
                }
                hostTeamId={host.team_id || 0}
                hostMdmEnrollmentStatus={host.mdm.enrollment_status}
              />
              {isMacOSHost && macadmins?.munki?.version && (
                <MunkiIssuesCard
                  isLoading={isLoadingHost}
                  munkiIssues={macadmins.munki_issues}
                  deviceType={host?.platform === "darwin" ? "macos" : ""}
                />
              )}
            </TabPanel>
            <TabPanel>
              {/* There is a special case for BYOD account driven enrolled mdm hosts where we are not
               currently supporting software installs. This check should be removed
               when we add that feature. Note: Android is currently a subset of BYODAccountDrivenUserEnrollment */}
              {isBYODAccountDrivenUserEnrollment(host.mdm.enrollment_status) ||
              isAndroidHost ? (
                <EmptyState
                  info={
                    <>
                      Software install is coming soon.{" "}
                      <CustomLink
                        text="Learn more"
                        url={
                          isAndroidHost
                            ? ANDROID_SW_INSTALL_LEARN_MORE_LINK
                            : BYOD_SW_INSTALL_LEARN_MORE_LINK
                        }
                        newTab
                      />
                    </>
                  }
                  header="Software library is currently not supported on this host"
                />
              ) : (
                <SoftwareLibraryCard
                  id={host.id}
                  platform={host.platform}
                  hostDisplayName={host?.display_name || ""}
                  softwareUpdatedAt={host.software_updated_at}
                  hostScriptsEnabled={host.scripts_enabled || false}
                  isSoftwareEnabled={featuresConfig?.enable_software_inventory}
                  router={router}
                  queryParams={{
                    ...parseHostSoftwareQueryParams(location.query),
                    available_for_install: true,
                  }}
                  pathname={location.pathname}
                  onShowInventoryVersions={
                    onSetSelectedHostSWForInventoryVersions
                  }
                  hostTeamId={host.team_id || 0}
                  hostName={host.display_name}
                  hostMDMEnrolled={host.mdm.connected_to_fleet}
                  isHostOnline={host.status === "online"}
                  refetchHostDetails={refetchHostDetails}
                  isHostDetailsPolling={showRefetchSpinner}
                />
              )}
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
              queryParams={{
                ...parseHostSoftwareQueryParams(location.query),
                include_available_for_install: false,
              }}
              pathname={location.pathname}
              onShowInventoryVersions={onSetSelectedHostSWForInventoryVersions}
              hostTeamId={host.team_id || 0}
            />
            {isMacOSHost && macadmins?.munki?.version && (
              <MunkiIssuesCard
                isLoading={isLoadingHost}
                munkiIssues={macadmins.munki_issues}
                deviceType={host?.platform === "darwin" ? "macos" : ""}
              />
            )}
          </>
        )}
      </div>
    );
  };

  const renderContent = (mainContentConfig: IMainContentConfig) => {
    return (
      <>
        <>
          {!mainContentConfig.renderedBanner && (
            <HostDetailsBanners
              mdmEnrollmentStatus={host?.mdm.enrollment_status}
              hostPlatform={host?.platform}
              macDiskEncryptionStatus={
                host?.mdm.apple_settings?.disk_encryption
              }
              connectedToFleetMdm={host?.mdm.connected_to_fleet}
              diskEncryptionOSSetting={host?.mdm.os_settings?.disk_encryption}
              diskIsEncrypted={host?.disk_encryption_enabled}
              diskEncryptionKeyAvailable={host?.mdm.encryption_key_available}
              lastMdmEnrolledAt={host?.last_mdm_enrolled_at}
            />
          )}
          <div className={`${baseClass}__header-links`}>
            <BackButton
              text="Back to all hosts"
              path={
                filteredHostsPath ||
                getPathWithQueryParams(PATHS.MANAGE_HOSTS, {
                  fleet_id: location.query.fleet_id,
                })
              }
            />
          </div>
          <div className={`${baseClass}__header-summary`}>
            <HostHeader
              summaryData={summaryData}
              showRefetchSpinner={showRefetchSpinner}
              onRefetchHost={onRefetchHost}
              renderActionsDropdown={renderActionsDropdown}
              hostMdmDeviceStatus={hostMdmDeviceStatus}
              hostMdmEnrollmentStatus={host.mdm?.enrollment_status || undefined}
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
                      <TabText count={navItem.count} countVariant="alert">
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
                  className={fullWidthCardClass}
                />
                <VitalsCard
                  className={fullWidthCardClass}
                  vitalsData={vitalsData}
                  munki={macadmins?.munki}
                  mdm={host?.mdm}
                  osVersionRequirement={getOSVersionRequirementFromMDMConfig(
                    host.platform
                  )}
                  toggleLocationModal={toggleLocationModal}
                  toggleMDMStatusModal={toggleMDMStatusModal}
                />
                <ActivityCard
                  className={
                    showAgentOptionsCard
                      ? tripleHeightCardClass
                      : defaultCardClass
                  }
                  activeTab={activeActivityTab}
                  activities={
                    activeActivityTab === "past"
                      ? pastActivities
                      : upcomingActivities
                  }
                  commands={
                    activeActivityTab === "past"
                      ? pastMDMCommands
                      : upcomingMDMCommands
                  }
                  isLoading={
                    activeActivityTab === "past"
                      ? pastActivitiesIsFetching || pastMDMCommandsIsFetching
                      : upcomingActivitiesIsFetching ||
                        upcomingMDMCommandsIsFetching
                  }
                  isError={
                    activeActivityTab === "past"
                      ? pastActivitiesIsError || pastMDMCommandsIsError
                      : upcomingActivitiesIsError || upcomingMDMCommandsIsError
                  }
                  canCancelActivities={
                    isGlobalAdmin ||
                    isGlobalMaintainer ||
                    isHostTeamAdmin ||
                    isHostTeamMaintainer
                  }
                  isUpcomingDisabled={isAndroidHost}
                  showMDMCommandsToggle={canGetMDMCommands}
                  showMDMCommands={showMDMCommands}
                  onShowMDMCommands={() => {
                    setActivityPage(0);
                    setShowMDMCommands(true);
                  }}
                  onHideMDMCommands={() => {
                    setActivityPage(0);
                    setShowMDMCommands(false);
                  }}
                  upcomingCount={
                    (upcomingActivities?.count || 0) +
                    (upcomingMDMCommands?.count || 0)
                  }
                  onChangeTab={onChangeActivityTab}
                  onNextPage={() => setActivityPage(activityPage + 1)}
                  onPreviousPage={() => setActivityPage(activityPage - 1)}
                  onShowDetails={onShowActivityDetails}
                  onShowCommandDetails={setMdmCommandDetails}
                  onCancel={onCancelActivity}
                />
                <UserCard
                  className={defaultCardClass}
                  endUsers={host.end_users ?? []}
                  canWriteEndUser={
                    isTeamMaintainerOrTeamAdmin ||
                    isGlobalAdmin ||
                    isGlobalMaintainer
                  }
                  onClickUpdateUser={(
                    e:
                      | React.MouseEvent<HTMLButtonElement>
                      | React.KeyboardEvent<HTMLButtonElement>
                  ) => {
                    e.preventDefault();
                    setShowUpdateEndUserModal(true);
                  }}
                />
                <LabelsCard
                  className={defaultCardClass}
                  labels={host?.labels || []}
                  onLabelClick={onLabelClick}
                />
                {showAgentOptionsCard && (
                  <AgentOptionsCard
                    className={defaultCardClass}
                    osqueryData={osqueryData}
                    wrapFleetHelper={wrapFleetHelper}
                    isChromeOS={host?.platform === "chrome"}
                  />
                )}
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
                    onPreviousPage={() =>
                      setCertificatePage(certificatePage - 1)
                    }
                    sortDirection={sortCerts.order_direction}
                    sortHeader={sortCerts.order_key}
                    onSortChange={setSortCerts}
                  />
                )}
              </TabPanel>
              <TabPanel>
                <TabNav className={`${baseClass}__software-tab-nav`} secondary>
                  <Tabs
                    selectedIndex={getSoftwareTabIndex(location.pathname)}
                    onSelect={(i) => navigateToSoftwareTab(i)}
                  >
                    {renderSoftwareCard()}
                  </Tabs>
                </TabNav>
              </TabPanel>
              {showReportsTab && (
                <TabPanel>
                  <HostReportsTab
                    hostId={host.id}
                    hostName={host.display_name}
                    router={router}
                    location={location}
                    saveReportsDisabledInConfig={
                      config?.server_settings?.query_reports_disabled
                    }
                    showReportsEmptyState={showReportsEmptyState}
                  />
                </TabPanel>
              )}
              <TabPanel>
                <PoliciesCard
                  policies={host?.policies || []}
                  isLoading={isLoadingHost}
                  togglePolicyDetailsModal={togglePolicyDetailsModal}
                  hostPlatform={host.platform}
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
              isUpdating={isUpdating}
            />
          )}
          {showSelectReportModal && host && (
            <SelectReportModal
              onCancel={() => setShowSelectReportModal(false)}
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
              isUpdating={isUpdating}
              hostsTeamId={host.team_id}
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
              canResendProfiles={canResendProfiles}
              canRotateRecoveryLockPassword={
                isGlobalAdmin ||
                isGlobalMaintainer ||
                isHostTeamAdmin ||
                isHostTeamMaintainer
              }
              platform={host.platform}
              hostMDMData={host.mdm}
              onClose={toggleOSSettingsModal}
              resendRequest={resendProfile}
              resendCertificateRequest={resendCertificate}
              rotateRecoveryLockPassword={rotateRecoveryLockPassword}
              onProfileResent={refetchHostDetails}
            />
          )}
          {showUnenrollMdmModal && !!host && host.mdm.enrollment_status && (
            <UnenrollMdmModal
              hostId={host.id}
              hostPlatform={host.platform}
              hostName={host.display_name}
              enrollmentStatus={host.mdm.enrollment_status}
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
          {showRecoveryLockPasswordModal && host && (
            <RecoveryLockPasswordModal
              hostId={host.id}
              canRotatePassword={
                isGlobalAdmin ||
                isGlobalMaintainer ||
                isHostTeamAdmin ||
                isHostTeamMaintainer
              }
              onCancel={() => setShowRecoveryLockPasswordModal(false)}
            />
          )}
          {showManagedAccountModal && host && (
            <ManagedAccountModal
              hostId={host.id}
              canRotatePassword={
                isGlobalAdmin ||
                isGlobalMaintainer ||
                isHostTeamAdmin ||
                isHostTeamMaintainer
              }
              onCancel={() => {
                setShowManagedAccountModal(false);
                // Opening the modal triggers a "viewed managed account"
                // activity server-side and may set auto_rotate_at; refetch
                // host details + activities so they reflect the new state.
                refetchHostDetails();
                refetchPastActivities();
              }}
              onRotate={() => {
                // The rotation activity and cleared auto_rotate_at land on
                // host details / activities — refresh both so the banner and
                // feed are in sync with the newly-rotated state.
                refetchHostDetails();
                refetchPastActivities();
              }}
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
          {scriptPackageDetails && (
            <SoftwareScriptDetailsModal
              details={scriptPackageDetails}
              onCancel={() => setScriptPackageDetails(null)}
            />
          )}
          {ipaPackageInstallDetails && (
            <SoftwareIpaInstallDetailsModal
              details={{
                appName: ipaPackageInstallDetails.appName || "",
                fleetInstallStatus: (ipaPackageInstallDetails.fleetInstallStatus ||
                  "pending_install") as SoftwareInstallUninstallStatus,
                hostDisplayName: ipaPackageInstallDetails.hostDisplayName || "",
                commandUuid: ipaPackageInstallDetails.commandUuid || "",
              }}
              onCancel={onCancelIpaSoftwareInstallDetailsModal}
            />
          )}
          {packageUninstallDetails && (
            <SoftwareUninstallDetailsModal
              {...packageUninstallDetails}
              hostDisplayName={packageUninstallDetails.hostDisplayName || ""}
              onCancel={() => setPackageUninstallDetails(null)}
            />
          )}
          {!!activityVPPInstallDetails && (
            <VppInstallDetailsModal
              details={activityVPPInstallDetails}
              onCancel={onCancelVppInstallDetailsModal}
            />
          )}
          {!!certificateInstallDetails && (
            <CertificateInstallDetailsModal
              details={certificateInstallDetails}
              onCancel={() => setCertificateInstallDetails(null)}
            />
          )}
          {!!mdmCommandDetails && (
            <CommandResultsModal
              command={mdmCommandDetails}
              onDone={onCancelMdmCommandDetailsModal}
            />
          )}
          {enrollmentProfileFailedDetails && (
            <FailedEnrollmentProfileModal
              command={enrollmentProfileFailedDetails.command}
              onDone={() => setEnrollmentProfileFailedDetails(null)}
            />
          )}
          {showLockHostModal && (
            <LockModal
              id={host.id}
              platform={host.platform}
              hostName={host.display_name}
              onSuccess={() => {
                setHostMdmDeviceState("locking");
                setShowLocationModal(false);
                setShowLockHostModal(false);
              }}
              onClose={() => {
                setShowLockHostModal(false);
                showLocationModal === undefined && setShowLocationModal(true);
              }}
            />
          )}
          {showUnlockHostModal && (
            <UnlockModal
              id={host.id}
              platform={host.platform}
              hostName={host.display_name}
              onSuccess={() => {
                host.platform !== "darwin" &&
                  setHostMdmDeviceState("unlocking");
              }}
              onClose={() => setShowUnlockHostModal(false)}
            />
          )}
          {showWipeModal && (
            <WipeModal
              id={host.id}
              hostName={host.display_name}
              isWindowsHost={isWindowsHost}
              onSuccess={() => setHostMdmDeviceState("wiping")}
              onClose={() => setShowWipeModal(false)}
            />
          )}
          {selectedHostSWForInventoryVersions && (
            <InventoryVersionsModal
              hostSoftware={selectedHostSWForInventoryVersions}
              onExit={() => setSelectedHostSWForInventoryVersions(null)}
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
        {showUpdateEndUserModal && (
          <UpdateEndUserModal
            isPremiumTier={isPremiumTier}
            endUsers={host.end_users ?? []}
            onUpdate={onUpdateEndUser}
            isUpdating={isUpdating}
            onExit={() => setShowUpdateEndUserModal(false)}
          />
        )}
        {showLocationModal && (
          <LocationModal
            hostGeolocation={host.geolocation}
            onExit={toggleLocationModal}
            iosOrIpadosDetails={{
              isIosOrIpadosHost,
              hostMdmDeviceStatus,
            }}
            onClickLock={() => {
              setShowLockHostModal(true);
              setShowLocationModal(undefined);
            }}
            detailsUpdatedAt={host.detail_updated_at}
          />
        )}
        {showMDMStatusModal && host.mdm.enrollment_status && (
          <MDMStatusModal
            fleetId={currentTeam?.id}
            hostId={host.id}
            depProfileError={host.mdm.dep_profile_error}
            enrollmentStatus={host.mdm.enrollment_status}
            isPremiumTier={isPremiumTier}
            isAppleDevice={isAppleDeviceHost}
            router={router}
            onExit={toggleMDMStatusModal}
          />
        )}
        {showClearPasscodeModal && (
          <ClearPasscodeModal id={host.id} onExit={toggleClearPasscodeModal} />
        )}
      </>
    );
  };

  return <MainContent className={baseClass}>{renderContent}</MainContent>;
};

export default HostDetailsPage;
