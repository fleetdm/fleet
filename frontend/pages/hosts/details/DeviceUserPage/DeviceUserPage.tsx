import React, { useState, useContext, useCallback, useEffect } from "react";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { useQuery } from "react-query";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import useIsMobileWidth from "hooks/useIsMobileWidth";
import { AxiosError } from "axios";

import { pick } from "lodash";

import { NotificationContext } from "context/notification";
import classNames from "classnames";

import deviceUserAPI, {
  IGetDeviceCertsRequestParams,
  IGetDeviceCertificatesResponse,
  IGetSetupExperienceStatusesResponse,
} from "services/entities/device_user";
import activitiesAPI, {
  IHostPastActivitiesResponse,
  IHostUpcomingActivitiesResponse,
} from "services/entities/activities";
import diskEncryptionAPI from "services/entities/disk_encryption";
import { IMacadminsResponse, IDUPDetails, IHostDevice } from "interfaces/host";
import { IListSort } from "interfaces/list_options";
import { IHostPolicy } from "interfaces/policy";
import { IDeviceGlobalConfig } from "interfaces/config";
import {
  IHostCertificate,
  CERTIFICATES_DEFAULT_SORT,
} from "interfaces/certificates";
import {
  isMacOS,
  isAppleDevice,
  isLinuxLike,
  isWindows,
} from "interfaces/platform";
import {
  IHostSoftware,
  resolveUninstallStatus,
  SCRIPT_PACKAGE_SOURCES,
  SoftwareInstallUninstallStatus,
} from "interfaces/software";
import { ISetupStep } from "interfaces/setup";

import shouldShowUnsupportedScreen from "layouts/UnsupportedScreenSize/helpers";

import DeviceUserError from "components/DeviceUserError";
// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";
import Spinner from "components/Spinner";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import FlashMessage from "components/FlashMessage";
import CustomLink from "components/CustomLink";

import { normalizeEmptyValues } from "utilities/helpers";
import PATHS from "router/paths";
import {
  DEFAULT_USE_QUERY_OPTIONS,
  DOCUMENT_TITLE_SUFFIX,
  HOST_VITALS_DATA,
  HOST_SUMMARY_DATA,
} from "utilities/constants";

import UnsupportedScreenSize from "layouts/UnsupportedScreenSize";

import { IShowActivityDetailsData } from "components/ActivityItem/ActivityItem";
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
import CertificateInstallDetailsModal, {
  ICertificateInstallDetails,
} from "components/ActivityDetails/InstallDetails/CertificateInstallDetailsModal";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";

import HostSummaryCard from "../cards/HostSummary";
import VitalsCard from "../cards/Vitals";
import SoftwareCard from "../cards/Software";
import PoliciesCard from "../cards/Policies";
import InfoModal from "./InfoModal";
import {
  getErrorMessage,
  hasRemainingSetupSteps,
  isSoftwareScriptSetup,
  isIPhone,
  isIPad,
} from "./helpers";

import PolicyDetailsModal from "../cards/Policies/HostPoliciesTable/PolicyDetailsModal";
import AutoEnrollMdmModal from "./AutoEnrollMdmModal";
import BitLockerPinModal from "./BitLockerPinModal";
import CreateLinuxKeyModal from "./CreateLinuxKeyModal";
import OSSettingsModal from "../OSSettingsModal";
import BootstrapPackageModal from "../HostDetailsPage/modals/BootstrapPackageModal";
import { parseHostSoftwareQueryParams } from "../cards/Software/HostSoftware";
import { parseSelfServiceQueryParams } from "../cards/Software/SelfService/SelfService";
import SelfService from "../cards/Software/SelfService";
import DeviceUserBanners from "./components/DeviceUserBanners";
import CertificateDetailsModal from "../modals/CertificateDetailsModal";
import CertificatesCard from "../cards/Certificates";
import ActivityCard from "../cards/Activity";
import UserCard from "../cards/User";
import HostHeader from "../cards/HostHeader/HostHeader";
import InventoryVersionsModal from "../modals/InventoryVersionsModal";
import { REFETCH_HOST_DETAILS_POLLING_INTERVAL } from "../HostDetailsPage/HostDetailsPage";

import SettingUpYourDevice from "./components/SettingUpYourDevice";
import InfoButton from "./components/InfoButton";
import BypassModal from "./BypassModal";

const baseClass = "device-user";

const fullWidthCardClass = `${baseClass}__card--full-width`;

const PREMIUM_TAB_PATHS = [
  PATHS.DEVICE_USER_DETAILS_SELF_SERVICE,
  PATHS.DEVICE_USER_DETAILS,
  PATHS.DEVICE_USER_DETAILS_SOFTWARE,
  PATHS.DEVICE_USER_DETAILS_POLICIES,
] as const;

const FREE_TAB_PATHS = [
  PATHS.DEVICE_USER_DETAILS,
  PATHS.DEVICE_USER_DETAILS_SOFTWARE,
] as const;

const DEFAULT_CERTIFICATES_PAGE_SIZE = 10;
const DEFAULT_CERTIFICATES_PAGE = 0;

interface IDeviceUserPageProps {
  location: {
    pathname: string;
    query: {
      vulnerable?: string;
      page?: string;
      query?: string;
      order_key?: string;
      order_direction?: "asc" | "desc";
      setup_only?: string;
    };
    search?: string;
  };
  router: InjectedRouter;
  params: Params;
}

const DeviceUserPage = ({
  location,
  router,
  params: { device_auth_token },
}: IDeviceUserPageProps): JSX.Element => {
  const deviceAuthToken = device_auth_token;
  const isMobileView = useIsMobileWidth();
  const isMobileDevice = isIPhone(navigator) || isIPad(navigator);

  const { renderFlash, notification, hideFlash } = useContext(
    NotificationContext
  );

  const [showBypassModal, setShowBypassModal] = useState(false);
  const [showBitLockerPINModal, setShowBitLockerPINModal] = useState(false);
  const [showInfoModal, setShowInfoModal] = useState(false);
  const [showEnrollMdmModal, setShowEnrollMdmModal] = useState(false);
  const [enrollUrlError, setEnrollUrlError] = useState<string | null>(null);
  const [selectedPolicy, setSelectedPolicy] = useState<IHostPolicy | null>(
    null
  );
  const [showPolicyDetailsModal, setShowPolicyDetailsModal] = useState(false);
  const [showOSSettingsModal, setShowOSSettingsModal] = useState(false);
  const [showBootstrapPackageModal, setShowBootstrapPackageModal] = useState(
    false
  );
  const [showCreateLinuxKeyModal, setShowCreateLinuxKeyModal] = useState(false);
  const [isTriggeringCreateLinuxKey, setIsTriggeringCreateLinuxKey] = useState(
    false
  );
  const [
    hostSWForInventoryVersions,
    setHostSWForInventoryVersions,
  ] = useState<IHostSoftware | null>(null);

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
  const [queuedSelfServiceRefetch, setQueuedSelfServiceRefetch] = useState(
    false
  );
  const [refetchStartTime, setRefetchStartTime] = useState<number | null>(null);
  const [showRefetchSpinner, setShowRefetchSpinner] = useState(false);

  // activity card states
  const [activeActivityTab, setActiveActivityTab] = useState<
    "past" | "upcoming"
  >("past");
  const [activityPage, setActivityPage] = useState(0);

  // activity details modal states (info icon on an activity item)
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
    activityCertificateInstallDetails,
    setActivityCertificateInstallDetails,
  ] = useState<ICertificateInstallDetails | null>(null);

  const { data: deviceMacAdminsData } = useQuery(
    ["macadmins", deviceAuthToken],
    () => deviceUserAPI.loadHostDetailsExtension(deviceAuthToken, "macadmins"),
    {
      enabled: !!deviceAuthToken,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: IMacadminsResponse) => data.macadmins,
    }
  );

  const {
    data: deviceCertificates,
    isLoading: isLoadingDeviceCertificates,
    isError: isErrorDeviceCertificates,
    refetch: refetchDeviceCertificates,
  } = useQuery<
    IGetDeviceCertificatesResponse,
    Error,
    IGetDeviceCertificatesResponse,
    Array<IGetDeviceCertsRequestParams & { scope: "device-certificates" }>
  >(
    [
      {
        scope: "device-certificates",
        token: deviceAuthToken,
        page: certificatePage,
        per_page: DEFAULT_CERTIFICATES_PAGE_SIZE,
        order_key: sortCerts.order_key,
        order_direction: sortCerts.order_direction,
      },
    ],
    ({ queryKey }) => deviceUserAPI.getDeviceCertificates(queryKey[0]),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      // FIXME: is it worth disabling for unsupported platforms? we'd have to workaround the a
      // catch-22 where we need to know the platform to know if it's supported but we also need to
      // be able to include the cert refetch in the hosts query hook.
      enabled: !!deviceUserAPI,
      keepPreviousData: true,
      staleTime: 15000,
    }
  );

  const DEVICE_ACTIVITY_PAGE_SIZE = 8;

  const {
    data: pastActivities,
    isFetching: pastActivitiesIsFetching,
    isError: pastActivitiesIsError,
    refetch: refetchPastActivities,
  } = useQuery<IHostPastActivitiesResponse, Error, IHostPastActivitiesResponse>(
    ["device-past-activities", deviceAuthToken, activityPage],
    () =>
      activitiesAPI.getDeviceHostPastActivities(
        deviceAuthToken,
        activityPage,
        DEVICE_ACTIVITY_PAGE_SIZE
      ),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: !!deviceAuthToken,
      keepPreviousData: true,
    }
  );

  const {
    data: upcomingActivities,
    isFetching: upcomingActivitiesIsFetching,
    isError: upcomingActivitiesIsError,
    refetch: refetchUpcomingActivities,
  } = useQuery<
    IHostUpcomingActivitiesResponse,
    Error,
    IHostUpcomingActivitiesResponse
  >(
    ["device-upcoming-activities", deviceAuthToken, activityPage],
    () =>
      activitiesAPI.getDeviceHostUpcomingActivities(
        deviceAuthToken,
        activityPage,
        DEVICE_ACTIVITY_PAGE_SIZE
      ),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: !!deviceAuthToken,
      keepPreviousData: true,
    }
  );

  const refetchExtensions = useCallback(() => {
    deviceCertificates && refetchDeviceCertificates();
    refetchPastActivities();
    refetchUpcomingActivities();
  }, [
    deviceCertificates,
    refetchDeviceCertificates,
    refetchPastActivities,
    refetchUpcomingActivities,
  ]);

  /**
   * Hides refetch spinner and resets refetch timer,
   * ensuring no stale timeout triggers on new requests.
   */
  const resetHostRefetchStates = () => {
    setShowRefetchSpinner(false);
    setRefetchStartTime(null);
  };

  const isRefetching = ({
    refetch_requested,
    refetch_critical_queries_until,
  }: IHostDevice) => {
    if (!refetch_critical_queries_until) {
      return refetch_requested;
    }

    const now = new Date();
    const refetchUntil = new Date(refetch_critical_queries_until);
    const isRefetchingCriticalQueries =
      !isNaN(refetchUntil.getTime()) && refetchUntil > now;
    return refetch_requested || isRefetchingCriticalQueries;
  };

  const {
    data: dupDetails,
    isLoading: isLoadingDupDetails,
    error: isDupDetailsError,
    refetch: refetchDupDetails,
  } = useQuery<IDUPDetails, AxiosError>(
    ["host", deviceAuthToken],
    () =>
      deviceUserAPI.loadHostDetails({
        token: deviceAuthToken,
        exclude_software: true,
      }),
    {
      enabled: !!deviceAuthToken,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      onSuccess: ({ host: responseHost }) => {
        // If we're just showing the setup screen,
        // we don't need to refetch or alert on offline hosts.
        if (location.query.setup_only) {
          return;
        }
        // Handle spinner and timer for refetch
        if (isRefetching(responseHost)) {
          setShowRefetchSpinner(true);

          // Only set timer if not already running
          if (!refetchStartTime) {
            // Here and below: iOS/iPadOS refetches use MDM commands which can be slower/less predictable
            // than osquery. Don't show an error, just reset and let the user try again.
            const isIOSOrIPadOS =
              responseHost.platform === "ios" ||
              responseHost.platform === "ipados";
            if (responseHost.status === "online" || isIOSOrIPadOS) {
              setRefetchStartTime(Date.now());
              setTimeout(() => {
                refetchDupDetails();
                refetchExtensions();
              }, REFETCH_HOST_DETAILS_POLLING_INTERVAL);
            } else {
              resetHostRefetchStates();
              renderFlash(
                "error",
                `This host is offline. Please try refetching host vitals later.`
              );
            }
          } else {
            const totalElapsedTime = Date.now() - refetchStartTime;
            if (totalElapsedTime < 180000) {
              const isIOSOrIPadOS =
                responseHost.platform === "ios" ||
                responseHost.platform === "ipados";
              if (responseHost.status === "online" || isIOSOrIPadOS) {
                setTimeout(() => {
                  refetchDupDetails();
                  refetchExtensions();
                }, REFETCH_HOST_DETAILS_POLLING_INTERVAL);
              } else {
                resetHostRefetchStates();
                renderFlash(
                  "error",
                  `This host is offline. Please try refetching host vitals later.`
                );
              }
            } else {
              // Timeout reached (3 minutes)
              resetHostRefetchStates();
              const isIOSOrIPadOS =
                responseHost.platform === "ios" ||
                responseHost.platform === "ipados";
              if (!isIOSOrIPadOS) {
                renderFlash(
                  "error",
                  "We're having trouble fetching fresh vitals for this host. Please try again later."
                );
              }
            }
          }
        } else {
          // Not refetching: reset spinner and timer
          resetHostRefetchStates();
        }
      },
    }
  );

  const isAuthenticationError =
    isDupDetailsError && isDupDetailsError.status === 401;

  const {
    host,
    license,
    org_logo_url_light_background: orgLogoURL = "",
    org_contact_url: orgContactURL = "",
    global_config: globalConfig = null as IDeviceGlobalConfig | null,
    self_service: hasSelfService = false,
  } = dupDetails || {};
  const isPremiumTier = license?.tier === "premium";
  const isAppleHost = isAppleDevice(host?.platform);
  const isIOSIPadOS = host?.platform === "ios" || host?.platform === "ipados";
  const isSetupExperienceSoftwareEnabledPlatform =
    isLinuxLike(host?.platform || "") ||
    host?.platform === "windows" ||
    isMacOS(host?.platform || "");

  const isFleetMdmManualUnenrolledMac =
    !!globalConfig?.mdm.enabled_and_configured &&
    !!host &&
    !host.dep_assigned_to_fleet &&
    host.platform === "darwin" &&
    (host.mdm.enrollment_status === "Off" ||
      host.mdm.enrollment_status === null);

  const checkForSetupExperienceSoftware =
    isSetupExperienceSoftwareEnabledPlatform && isPremiumTier;

  const summaryData = normalizeEmptyValues(pick(host, HOST_SUMMARY_DATA));

  const vitalsData = normalizeEmptyValues(pick(host, HOST_VITALS_DATA));

  const {
    data: setupStepStatuses,
    isLoading: isLoadingSetupSteps,
    isError: isErrorSetupSteps,
  } = useQuery<
    IGetSetupExperienceStatusesResponse,
    AxiosError,
    ISetupStep[] | null | undefined
  >(
    ["software-setup-statuses", deviceAuthToken],
    () => deviceUserAPI.getSetupExperienceStatuses({ token: deviceAuthToken }),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: checkForSetupExperienceSoftware, // this can only become true once the above `dupResponse` is defined by its associated API call response, ensuring this call only fires once the frontend knows if this is a Fleet Premium instance
      refetchInterval: (data) => (hasRemainingSetupSteps(data) ? 5000 : false), // refetch every 5s until finished
      refetchIntervalInBackground: true,
      select: (response) => {
        // Marshal the response to include a `type` property so we can differentiate
        // between software, script-only software, and script setup steps in the UI.
        return [
          ...(response.setup_experience_results.software ?? []).map((s) => ({
            ...s,
            type: isSoftwareScriptSetup(s)
              ? "software_script_run" // used for script-only software
              : "software_install",
          })),
          ...(response.setup_experience_results.scripts ?? []).map((s) => ({
            ...s,
            type: "script_run" as const,
          })),
        ];
      },
    }
  );

  const {
    data: mdmManualEnrollUrl,
    // isLoading, // not used; see related comment in onClickTurnOnMdm below
    error: mdmManualEnrollUrlError,
  } = useQuery<{ enroll_url: string }, Error, string>(
    ["mdm_mandual_enroll_url", deviceAuthToken],
    () => deviceUserAPI.getMdmManualEnrollUrl(deviceAuthToken),
    {
      enabled: !!deviceAuthToken && isFleetMdmManualUnenrolledMac,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data) => data.enroll_url,
    }
  );

  const { bypassConditionalAccess } = deviceUserAPI;

  const [isLoadingBypass, setIsLoadingBypass] = useState(false);

  const toggleShowBypassModal = useCallback(() => {
    setShowBypassModal(!showBypassModal);
  }, [showBypassModal, setShowBypassModal]);

  const toggleInfoModal = useCallback(() => {
    setShowInfoModal(!showInfoModal);
  }, [showInfoModal, setShowInfoModal]);

  const toggleEnrollMdmModal = useCallback(() => {
    setShowEnrollMdmModal(!showEnrollMdmModal);
  }, [showEnrollMdmModal, setShowEnrollMdmModal]);

  const onClickTurnOnMdm = useCallback(async () => {
    if (host?.dep_assigned_to_fleet) {
      // display the modal with auto-enroll instructions
      setShowEnrollMdmModal(true);
      return;
    }
    // if we have an enroll URL, DeviceUserBanners will display a CustomLink in place of the Button;
    // in some unexpected cases, may not have an enroll URL at this point (e.g., there was an error
    // fetching the URL from the API or the user clicked the link extremely quickly after page load
    // before the URL was fetched), we fallback to showing the Button and we'll display an error if
    // the user tries to click when we don't have an enroll URL.
    setEnrollUrlError(
      `Failed to get enrollment URL. ${mdmManualEnrollUrlError}`
    );
  }, [host?.dep_assigned_to_fleet, mdmManualEnrollUrlError]);

  const togglePolicyDetailsModal = useCallback(
    (policy: IHostPolicy) => {
      setShowPolicyDetailsModal(!showPolicyDetailsModal);
      setSelectedPolicy(policy);
    },
    [showPolicyDetailsModal, setShowPolicyDetailsModal, setSelectedPolicy]
  );

  const bootstrapPackageData = {
    status: host?.mdm.setup_experience?.bootstrap_package_status,
    details: host?.mdm.setup_experience?.details,
    name: host?.mdm.setup_experience?.bootstrap_package_name,
  };

  const toggleOSSettingsModal = useCallback(() => {
    setShowOSSettingsModal(!showOSSettingsModal);
  }, [showOSSettingsModal, setShowOSSettingsModal]);

  const onCancelPolicyDetailsModal = useCallback(() => {
    setShowPolicyDetailsModal(false);
    setSelectedPolicy(null);
  }, [setShowPolicyDetailsModal, setSelectedPolicy]);

  // User-initiated refetch always starts a new timer!
  const onRefetchHost = useCallback(async () => {
    if (!host) return;
    setShowRefetchSpinner(true);
    try {
      await deviceUserAPI.refetch(deviceAuthToken);
      setRefetchStartTime(Date.now());
      setTimeout(() => {
        refetchDupDetails();
        refetchExtensions();
      }, REFETCH_HOST_DETAILS_POLLING_INTERVAL);
    } catch (error) {
      renderFlash("error", getErrorMessage(error, host.display_name));
      resetHostRefetchStates();
    }
  }, [
    host,
    deviceAuthToken,
    refetchDupDetails,
    refetchExtensions,
    renderFlash,
  ]);

  // Handles the queue: If there's a queued refetch and not actively refetching, run refetch
  useEffect(() => {
    if (queuedSelfServiceRefetch && !showRefetchSpinner) {
      setQueuedSelfServiceRefetch(false);
      onRefetchHost();
    }
  }, [queuedSelfServiceRefetch, showRefetchSpinner, onRefetchHost]);

  // Triggered when a software update finishes
  const requestRefetch = () => {
    // If a refetch is already happening, queue this refetch
    if (showRefetchSpinner) {
      setQueuedSelfServiceRefetch(true);
    } else {
      // Otherwise, run it now
      onRefetchHost();
    }
  };

  // Updates title that shows up on browser tabs
  useEffect(() => {
    document.title = `My device | ${DOCUMENT_TITLE_SUFFIX}`;
  }, [location.pathname, host]);

  const renderActionButtons = () => {
    return (
      <div className={`${baseClass}__action-button-container`}>
        <InfoButton onClick={toggleInfoModal} />
      </div>
    );
  };

  const onTriggerEscrowLinuxKey = async () => {
    setIsTriggeringCreateLinuxKey(true);
    // modal opens in loading state
    setShowCreateLinuxKeyModal(true);
    try {
      await diskEncryptionAPI.triggerLinuxDiskEncryptionKeyEscrow(
        deviceAuthToken
      );
    } catch (e) {
      renderFlash("error", "Failed to trigger key creation.");
      setShowCreateLinuxKeyModal(false);
    } finally {
      setIsTriggeringCreateLinuxKey(false);
    }
  };

  const onSelectCertificate = (certificate: IHostCertificate) => {
    setSelectedCertificate(certificate);
  };

  // Handler for the "info" icon on an activity item. Mirrors the admin host
  // page handler (HostDetailsPage `onShowActivityDetails`) but only opens the
  // detail modals whose underlying API supports the device auth token. Ran
  // script details are not currently exposed to device-authenticated callers
  // so we leave that activity type without a modal.
  const onShowActivityDetails = useCallback(
    ({ type, details }: IShowActivityDetailsData) => {
      const hostDisplayName =
        host?.display_name || details?.host_display_name || "";
      switch (type) {
        case "installed_software":
          if (details?.command_uuid) {
            setIpaPackageInstallDetails({
              fleetInstallStatus: details?.status as SoftwareInstallUninstallStatus,
              hostDisplayName,
              appName: getDisplayedSoftwareName(
                details.software_title,
                details.software_display_name
              ),
              commandUuid: details?.command_uuid,
            });
          } else if (SCRIPT_PACKAGE_SOURCES.includes(details?.source || "")) {
            setScriptPackageDetails({
              ...details,
              host_display_name: hostDisplayName,
            });
          } else {
            setPackageInstallDetails({
              ...details,
              host_display_name: hostDisplayName,
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
            hostDisplayName,
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
            hostDisplayName,
            platform: details?.host_platform || host?.platform,
          });
          break;
        case "installed_certificate":
          setActivityCertificateInstallDetails({
            certificateName: details?.certificate_name || "",
            hostDisplayName,
            status: details?.status || "",
            detail: details?.detail || "",
          });
          break;
        default:
          // No device-mode modal for the remaining types. Items whose modal is
          // admin-only (ran_script, FailedEnrollmentProfileRenewal) hide their
          // show-info icon entirely via the ActivityCard's isMyDevicePage prop.
          break;
      }
    },
    [host?.display_name, host?.platform]
  );

  const resendProfile = useCallback(
    (profileUUID: string): Promise<void> => {
      return deviceUserAPI.resendProfile(deviceAuthToken, profileUUID);
    },
    [deviceAuthToken]
  );

  const renderDeviceUserPage = () => {
    const failingPoliciesCount = host?.issues?.failing_policies_count || 0;

    // TODO: We should probably have a standard way to handle this on all pages. Do we want to show
    // a premium-only message in the case that a user tries direct navigation to a premium-only page
    // or silently redirect as below?
    let tabPaths = (isPremiumTier
      ? PREMIUM_TAB_PATHS
      : FREE_TAB_PATHS
    ).map((t) => t(deviceAuthToken));
    if (!hasSelfService) {
      tabPaths = tabPaths.filter((path) => !path.includes("self-service"));
    }

    const findSelectedTab = (pathname: string) => {
      const cleanPath = pathname.split("?")[0];
      // Filter tabPaths that are prefix of cleanPath
      const matchingIndices = tabPaths
        .map((tabPath, idx) => ({ tabPath, idx }))
        .filter(({ tabPath }) => cleanPath.startsWith(tabPath));

      if (matchingIndices.length === 0) {
        return -1;
      }

      // Return the index of the longest matching prefix
      return matchingIndices.reduce((best, current) =>
        current.tabPath.length > best.tabPath.length ? current : best
      ).idx;
    };

    if (
      !isLoadingDupDetails &&
      host &&
      findSelectedTab(location.pathname) === -1
    ) {
      router.push(tabPaths[0]);
    }

    // Note: API response global_config is misnamed because the backend actually returns the global
    // or team config (as applicable)
    const isSoftwareEnabled = !!globalConfig?.features
      ?.enable_software_inventory;

    if (
      !host ||
      isLoadingDupDetails ||
      isLoadingDeviceCertificates ||
      isLoadingSetupSteps
    ) {
      return <Spinner {...(isMobileView && { variant: "mobile" })} />;
    }
    if (isErrorSetupSteps) {
      return (
        <div className={`${baseClass} main-content`}>
          <DeviceUserError
            isMobileView={isMobileView}
            isMobileDevice={isMobileDevice}
            isErrorSetupSteps={isErrorSetupSteps}
          />
        </div>
      );
    }
    if (
      checkForSetupExperienceSoftware &&
      (hasRemainingSetupSteps(setupStepStatuses) || location.query.setup_only)
    ) {
      // at this point, softwareSetupStatuses will be non-empty
      return (
        <SettingUpYourDevice
          setupSteps={setupStepStatuses || []}
          requireAllSoftware={
            (isAppleHost && globalConfig?.mdm?.require_all_software_macos) ??
            false
          }
          toggleInfoModal={toggleInfoModal}
          platform={host.platform}
        />
      );
    }

    // iOS/iPadOS devices or narrow screens should show mobile UI
    const shouldShowMobileUI = isIOSIPadOS || isMobileView;

    if (shouldShowMobileUI) {
      // Force redirect to self-service route for iOS/iPadOS devices
      if (
        isIOSIPadOS &&
        !location.pathname.includes("/self-service") &&
        hasSelfService
      ) {
        router.replace(PATHS.DEVICE_USER_DETAILS_SELF_SERVICE(deviceAuthToken));
        return <Spinner />;
      }

      // Render the simplified mobile version
      // For iOS/iPadOS and narrow screen devices
      return (
        <div className={`${baseClass} main-content`}>
          <div className="device-user-mobile">
            <SelfService
              contactUrl={orgContactURL}
              deviceToken={deviceAuthToken}
              isSoftwareEnabled
              pathname={location.pathname}
              queryParams={parseSelfServiceQueryParams(location.query)}
              router={router}
              refetchHostDetails={requestRefetch}
              isHostDetailsPolling={showRefetchSpinner}
              hostSoftwareUpdatedAt={host.software_updated_at}
              hostDisplayName={host?.hostname || ""}
              isMobileView={shouldShowMobileUI}
              mdmEnrollmentStatus={host.mdm.enrollment_status || "Off"}
            />
          </div>
        </div>
      );
    }

    const hasAnyCriticalFailingCAPolicy = host?.policies?.some(
      (p) => p.response === "fail" && p.conditional_access_enabled && p.critical
    );

    return (
      <>
        <div className={`${baseClass} main-content`}>
          <DeviceUserBanners
            hostPlatform={host.platform}
            hostOsVersion={host.os_version}
            mdmEnrollmentStatus={host.mdm.enrollment_status}
            mdmEnabledAndConfigured={!!globalConfig?.mdm.enabled_and_configured}
            connectedToFleetMdm={!!host.mdm.connected_to_fleet}
            macDiskEncryptionStatus={
              host.mdm.apple_settings?.disk_encryption ?? null
            }
            diskEncryptionActionRequired={
              host.mdm.apple_settings?.action_required ?? null
            }
            onClickCreatePIN={() => setShowBitLockerPINModal(true)}
            onClickTurnOnMdm={onClickTurnOnMdm}
            onTriggerEscrowLinuxKey={onTriggerEscrowLinuxKey}
            diskEncryptionOSSetting={host.mdm.os_settings?.disk_encryption}
            diskIsEncrypted={host.disk_encryption_enabled}
            diskEncryptionKeyAvailable={host.mdm.encryption_key_available}
            mdmManualEnrolmentUrl={mdmManualEnrollUrl}
            lastMdmEnrolledAt={host.last_mdm_enrolled_at}
          />
          <HostHeader
            summaryData={summaryData}
            showRefetchSpinner={showRefetchSpinner}
            onRefetchHost={onRefetchHost}
            renderActionsDropdown={renderActionButtons}
            deviceUser
          />
          <TabNav className={`${baseClass}__tab-nav`}>
            <Tabs
              selectedIndex={findSelectedTab(location.pathname)}
              onSelect={(i) => router.push(tabPaths[i])}
            >
              <TabList>
                {isPremiumTier && isSoftwareEnabled && hasSelfService && (
                  <Tab>
                    <TabText>Self-service</TabText>
                  </Tab>
                )}
                <Tab>
                  <TabText>Details</TabText>
                </Tab>
                {isSoftwareEnabled && (
                  <Tab>
                    <TabText>Software</TabText>
                  </Tab>
                )}
                {isPremiumTier && (
                  <Tab>
                    <TabText count={failingPoliciesCount} countVariant="alert">
                      Policies
                    </TabText>
                  </Tab>
                )}
              </TabList>
              {isPremiumTier && isSoftwareEnabled && hasSelfService && (
                <TabPanel>
                  <SelfService
                    contactUrl={orgContactURL}
                    deviceToken={deviceAuthToken}
                    isSoftwareEnabled
                    pathname={location.pathname}
                    queryParams={parseSelfServiceQueryParams(location.query)}
                    router={router}
                    refetchHostDetails={requestRefetch}
                    isHostDetailsPolling={showRefetchSpinner}
                    hostSoftwareUpdatedAt={host.software_updated_at}
                    hostDisplayName={host?.hostname || ""}
                    mdmEnrollmentStatus={host.mdm.enrollment_status || "Off"}
                  />
                </TabPanel>
              )}
              <TabPanel className={`${baseClass}__details-panel`}>
                <HostSummaryCard
                  className={fullWidthCardClass}
                  summaryData={summaryData}
                  bootstrapPackageData={bootstrapPackageData}
                  isPremiumTier={isPremiumTier}
                  toggleOSSettingsModal={toggleOSSettingsModal}
                  hostSettings={host?.mdm.profiles ?? []}
                  osSettings={host?.mdm.os_settings}
                />
                <VitalsCard
                  className={fullWidthCardClass}
                  vitalsData={vitalsData}
                  munki={deviceMacAdminsData?.munki}
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
                  canCancelActivities={false}
                  isUpcomingDisabled={host.platform === "android"}
                  isMyDevicePage
                  showMDMCommandsToggle={false}
                  showMDMCommands={false}
                  onChangeTab={(index: number) => {
                    setActiveActivityTab(index === 0 ? "past" : "upcoming");
                    setActivityPage(0);
                  }}
                  onNextPage={() => setActivityPage(activityPage + 1)}
                  onPreviousPage={() => setActivityPage(activityPage - 1)}
                  onShowDetails={onShowActivityDetails}
                  onShowCommandDetails={() => undefined}
                  onCancel={() => undefined}
                  onShowMDMCommands={() => undefined}
                  onHideMDMCommands={() => undefined}
                />
                <UserCard
                  canWriteEndUser={false}
                  endUsers={host.end_users ?? []}
                  disableFullNameTooltip
                  disableGroupsTooltip
                />
                {isAppleHost && !!deviceCertificates?.certificates.length && (
                  <CertificatesCard
                    className={fullWidthCardClass}
                    isMyDevicePage
                    data={deviceCertificates}
                    isError={isErrorDeviceCertificates}
                    page={certificatePage}
                    pageSize={DEFAULT_CERTIFICATES_PAGE_SIZE}
                    sortHeader={sortCerts.order_key}
                    sortDirection={sortCerts.order_direction}
                    hostPlatform={host.platform}
                    onSelectCertificate={onSelectCertificate}
                    onNextPage={() => setCertificatePage(certificatePage + 1)}
                    onPreviousPage={() =>
                      setCertificatePage(certificatePage - 1)
                    }
                    onSortChange={setSortCerts}
                  />
                )}
              </TabPanel>
              {isSoftwareEnabled && (
                <TabPanel>
                  <SoftwareCard
                    id={deviceAuthToken}
                    softwareUpdatedAt={host.software_updated_at}
                    router={router}
                    pathname={location.pathname}
                    queryParams={parseHostSoftwareQueryParams(location.query)}
                    isMyDevicePage
                    platform={host.platform}
                    hostTeamId={host.team_id || 0}
                    isSoftwareEnabled={isSoftwareEnabled}
                    onShowInventoryVersions={setHostSWForInventoryVersions}
                  />
                </TabPanel>
              )}
              {isPremiumTier && (
                <TabPanel>
                  <PoliciesCard
                    policies={host?.policies || []}
                    isLoading={isLoadingDupDetails}
                    deviceUser
                    togglePolicyDetailsModal={togglePolicyDetailsModal}
                    closePolicyDetailsModal={onCancelPolicyDetailsModal}
                    hostPlatform={host?.platform || ""}
                    conditionalAccessEnabled={
                      globalConfig?.features?.enable_conditional_access
                    }
                    conditionalAccessBypassed={
                      host?.conditional_access_bypassed
                    }
                  />
                </TabPanel>
              )}
            </Tabs>
          </TabNav>
          {showEnrollMdmModal && host.dep_assigned_to_fleet ? (
            <AutoEnrollMdmModal host={host} onCancel={toggleEnrollMdmModal} />
          ) : null}
          {showBitLockerPINModal && (
            <BitLockerPinModal
              onCancel={() => setShowBitLockerPINModal(false)}
            />
          )}
        </div>
        {!!host && showPolicyDetailsModal && (
          <PolicyDetailsModal
            onCancel={onCancelPolicyDetailsModal}
            policy={selectedPolicy}
            isDeviceUser
            onResolveLater={
              globalConfig?.features?.enable_conditional_access &&
              globalConfig.features?.enable_conditional_access_bypass &&
              !hasAnyCriticalFailingCAPolicy
                ? () => {
                    onCancelPolicyDetailsModal();
                    setShowBypassModal(true);
                  }
                : undefined
            }
          />
        )}
        {!!host && showOSSettingsModal && (
          <OSSettingsModal
            canResendProfiles={isAppleHost || isWindows(host.platform)}
            platform={host.platform}
            hostMDMData={host.mdm}
            resendRequest={resendProfile}
            onProfileResent={refetchDupDetails}
            onClose={toggleOSSettingsModal}
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
        {showCreateLinuxKeyModal && !!host && (
          <CreateLinuxKeyModal
            isTriggeringCreateLinuxKey={isTriggeringCreateLinuxKey}
            onExit={() => {
              setShowCreateLinuxKeyModal(false);
            }}
          />
        )}
        {hostSWForInventoryVersions && !!host && (
          <InventoryVersionsModal
            hostSoftware={hostSWForInventoryVersions}
            onExit={() => setHostSWForInventoryVersions(null)}
          />
        )}
        {selectedCertificate && (
          <CertificateDetailsModal
            certificate={selectedCertificate}
            onExit={() => setSelectedCertificate(null)}
          />
        )}
        {!!packageInstallDetails && (
          <SoftwareInstallDetailsModal
            details={packageInstallDetails}
            deviceAuthToken={deviceAuthToken}
            onCancel={() => setPackageInstallDetails(null)}
          />
        )}
        {scriptPackageDetails && (
          <SoftwareScriptDetailsModal
            details={scriptPackageDetails}
            deviceAuthToken={deviceAuthToken}
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
            deviceAuthToken={deviceAuthToken}
            onCancel={() => setIpaPackageInstallDetails(null)}
          />
        )}
        {packageUninstallDetails && (
          <SoftwareUninstallDetailsModal
            {...packageUninstallDetails}
            hostDisplayName={packageUninstallDetails.hostDisplayName || ""}
            deviceAuthToken={deviceAuthToken}
            onCancel={() => setPackageUninstallDetails(null)}
          />
        )}
        {!!activityVPPInstallDetails && (
          <VppInstallDetailsModal
            details={activityVPPInstallDetails}
            deviceAuthToken={deviceAuthToken}
            onCancel={() => setActivityVPPInstallDetails(null)}
          />
        )}
        {!!activityCertificateInstallDetails && (
          <CertificateInstallDetailsModal
            details={activityCertificateInstallDetails}
            onCancel={() => setActivityCertificateInstallDetails(null)}
          />
        )}
      </>
    );
  };

  const coreWrapperClassnames = classNames("core-wrapper", {
    "low-width-supported": !shouldShowUnsupportedScreen(location.pathname),
  });

  const siteNavContainerClassnames = classNames("site-nav-container", {
    "low-width-supported": !shouldShowUnsupportedScreen(location.pathname),
  });

  return (
    <div className="app-wrap">
      {shouldShowUnsupportedScreen(location.pathname) && (
        <UnsupportedScreenSize />
      )}
      <FlashMessage
        fullWidth
        notification={notification}
        onRemoveFlash={hideFlash}
        pathname={location.pathname}
      />
      <nav className={siteNavContainerClassnames}>
        <div className="site-nav-content">
          <ul className="site-nav-left">
            <li className="site-nav-item dup-org-logo" key="dup-org-logo">
              <div className="site-nav-item__logo-wrapper">
                <div className="site-nav-item__logo">
                  {isLoadingDupDetails ? (
                    <Spinner includeContainer={false} centered={false} />
                  ) : (
                    <OrgLogoIcon className="logo" src={orgLogoURL} />
                  )}
                </div>
              </div>
            </li>
          </ul>
          {isMobileView && (
            <div className="site-nav-better-link">
              <CustomLink
                url={PATHS.DEVICE_TRANSPARENCY(deviceAuthToken)}
                text="About Fleet"
                newTab
              />
            </div>
          )}
        </div>
      </nav>
      {isDupDetailsError || enrollUrlError ? (
        <DeviceUserError
          isMobileView={isMobileView}
          isMobileDevice={isMobileDevice}
          isAuthenticationError={!!isAuthenticationError}
        />
      ) : (
        <div className={coreWrapperClassnames}>{renderDeviceUserPage()}</div>
      )}
      {showInfoModal && (
        <InfoModal
          onCancel={toggleInfoModal}
          transparencyURL={PATHS.DEVICE_TRANSPARENCY(deviceAuthToken)}
        />
      )}
      {showBypassModal && (
        <BypassModal
          onCancel={toggleShowBypassModal}
          onResolveLater={async () => {
            setIsLoadingBypass(true);
            try {
              await bypassConditionalAccess(deviceAuthToken);
              renderFlash(
                "success",
                "Access has been temporarily restored. You may now attempt to sign in again."
              );
              refetchDupDetails();
            } catch {
              renderFlash(
                "error",
                `Couldn't restore access. Please click "Refetch" and try again.`
              );
            } finally {
              setIsLoadingBypass(false);
              setShowBypassModal(false);
            }
          }}
          isLoading={isLoadingBypass}
        />
      )}
    </div>
  );
};

export default DeviceUserPage;
