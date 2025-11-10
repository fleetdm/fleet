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
import diskEncryptionAPI from "services/entities/disk_encryption";
import {
  IMacadminsResponse,
  IDeviceUserResponse,
  IHostDevice,
} from "interfaces/host";
import { IListSort } from "interfaces/list_options";
import { IHostPolicy } from "interfaces/policy";
import { IDeviceGlobalConfig } from "interfaces/config";
import {
  IHostCertificate,
  CERTIFICATES_DEFAULT_SORT,
} from "interfaces/certificates";
import {
  isAndroid,
  isMacOS,
  isAppleDevice,
  isLinuxLike,
} from "interfaces/platform";
import { IHostSoftware } from "interfaces/software";
import { ISetupStep } from "interfaces/setup";

import shouldShowUnsupportedScreen from "layouts/UnsupportedScreenSize/helpers";

import DeviceUserError from "components/DeviceUserError";
// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";
import Spinner from "components/Spinner";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import FlashMessage from "components/FlashMessage";
import DataError from "components/DataError";
import CustomLink from "components/CustomLink";

import { normalizeEmptyValues } from "utilities/helpers";
import PATHS from "router/paths";
import {
  DEFAULT_USE_QUERY_OPTIONS,
  DOCUMENT_TITLE_SUFFIX,
  HOST_ABOUT_DATA,
  HOST_SUMMARY_DATA,
} from "utilities/constants";

import UnsupportedScreenSize from "layouts/UnsupportedScreenSize";

import HostSummaryCard from "../cards/HostSummary";
import AboutCard from "../cards/About";
import SoftwareCard from "../cards/Software";
import PoliciesCard from "../cards/Policies";
import InfoModal from "./InfoModal";
import {
  getErrorMessage,
  hasRemainingSetupSteps,
  isSoftwareScriptSetup,
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
import UserCard from "../cards/User";
import {
  generateChromeProfilesValues,
  generateOtherEmailsValues,
} from "../cards/User/helpers";
import HostHeader from "../cards/HostHeader/HostHeader";
import InventoryVersionsModal from "../modals/InventoryVersionsModal";
import { REFETCH_HOST_DETAILS_POLLING_INTERVAL } from "../HostDetailsPage/HostDetailsPage";

import SettingUpYourDevice from "./components/SettingUpYourDevice";
import InfoButton from "./components/InfoButton";

const baseClass = "device-user";

const defaultCardClass = `${baseClass}__card`;
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

  const { renderFlash, notification, hideFlash } = useContext(
    NotificationContext
  );

  const [showBitLockerPINModal, setShowBitLockerPINModal] = useState(false);
  const [showInfoModal, setShowInfoModal] = useState(false);
  const [showEnrollMdmModal, setShowEnrollMdmModal] = useState(false);
  const [enrollUrlError, setEnrollUrlError] = useState<string | null>(null);
  const [refetchStartTime, setRefetchStartTime] = useState<number | null>(null);
  const [showRefetchSpinner, setShowRefetchSpinner] = useState(false);
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

  const refetchExtensions = () => {
    deviceCertificates && refetchDeviceCertificates();
  };

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
    data: dupResponse,
    isLoading: isLoadingHost,
    error: isDeviceUserError,
    refetch: refetchHostDetails,
  } = useQuery<IDeviceUserResponse, AxiosError>(
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
            if (responseHost.status === "online") {
              setRefetchStartTime(Date.now());
              setTimeout(() => {
                refetchHostDetails();
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
            if (totalElapsedTime < 60000) {
              if (responseHost.status === "online") {
                setTimeout(() => {
                  refetchHostDetails();
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
              resetHostRefetchStates();
              renderFlash(
                "error",
                "We're having trouble fetching fresh vitals for this host. Please try again later."
              );
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
    isDeviceUserError && isDeviceUserError.status === 401;

  const {
    host,
    license,
    org_logo_url_light_background: orgLogoURL = "",
    org_contact_url: orgContactURL = "",
    global_config: globalConfig = null as IDeviceGlobalConfig | null,
    self_service: hasSelfService = false,
  } = dupResponse || {};
  const isPremiumTier = license?.tier === "premium";
  const isAppleHost = isAppleDevice(host?.platform);
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

  const aboutData = normalizeEmptyValues(pick(host, HOST_ABOUT_DATA));

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
        // between software, payload-free software, and script setup steps in the UI.
        return [
          ...(response.setup_experience_results.software ?? []).map((s) => ({
            ...s,
            type: isSoftwareScriptSetup(s)
              ? "software_script_run" // used for payload-free software
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
    status: host?.mdm.macos_setup?.bootstrap_package_status,
    details: host?.mdm.macos_setup?.details,
    name: host?.mdm.macos_setup?.bootstrap_package_name,
  };

  const toggleOSSettingsModal = useCallback(() => {
    setShowOSSettingsModal(!showOSSettingsModal);
  }, [showOSSettingsModal, setShowOSSettingsModal]);

  const onCancelPolicyDetailsModal = useCallback(() => {
    setShowPolicyDetailsModal(!showPolicyDetailsModal);
    setSelectedPolicy(null);
  }, [showPolicyDetailsModal, setShowPolicyDetailsModal, setSelectedPolicy]);

  // User-initiated refetch always starts a new timer!
  const onRefetchHost = async () => {
    if (host) {
      setShowRefetchSpinner(true);
      try {
        await deviceUserAPI.refetch(deviceAuthToken);
        setRefetchStartTime(Date.now()); // Always reset on user action
        setTimeout(() => {
          refetchHostDetails();
          refetchExtensions();
        }, REFETCH_HOST_DETAILS_POLLING_INTERVAL);
      } catch (error) {
        renderFlash("error", getErrorMessage(error, host.display_name));
        resetHostRefetchStates();
      }
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

  const resendProfile = (profileUUID: string): Promise<void> => {
    if (!host) {
      return new Promise(() => undefined);
    }
    return deviceUserAPI.resendProfile(deviceAuthToken, profileUUID);
  };

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

    if (!isLoadingHost && host && findSelectedTab(location.pathname) === -1) {
      router.push(tabPaths[0]);
    }

    // Note: API response global_config is misnamed because the backend actually returns the global
    // or team config (as applicable)
    const isSoftwareEnabled = !!globalConfig?.features
      ?.enable_software_inventory;

    const showUsersCard =
      isMacOS(host?.platform || "") ||
      isAndroid(host?.platform || "") ||
      generateChromeProfilesValues(host?.end_users ?? []).length > 0 ||
      generateOtherEmailsValues(host?.end_users ?? []).length > 0;

    if (
      !host ||
      isLoadingHost ||
      isLoadingDeviceCertificates ||
      isLoadingSetupSteps
    ) {
      return <Spinner {...(isMobileView && { variant: "mobile" })} />;
    }
    if (isErrorSetupSteps) {
      return <DataError description="Could not get software setup status." />;
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

    if (isMobileView) {
      // Render the simplified mobile version
      // Currently only available for self-service page
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
              refetchHostDetails={refetchHostDetails}
              isHostDetailsPolling={showRefetchSpinner}
              hostSoftwareUpdatedAt={host.software_updated_at}
              hostDisplayName={host?.hostname || ""}
              isMobileView={isMobileView}
            />
          </div>
        </div>
      );
    }

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
              host.mdm.macos_settings?.disk_encryption ?? null
            }
            diskEncryptionActionRequired={
              host.mdm.macos_settings?.action_required ?? null
            }
            onClickCreatePIN={() => setShowBitLockerPINModal(true)}
            onClickTurnOnMdm={onClickTurnOnMdm}
            onTriggerEscrowLinuxKey={onTriggerEscrowLinuxKey}
            diskEncryptionOSSetting={host.mdm.os_settings?.disk_encryption}
            diskIsEncrypted={host.disk_encryption_enabled}
            diskEncryptionKeyAvailable={host.mdm.encryption_key_available}
            mdmManualEnrolmentUrl={mdmManualEnrollUrl}
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
                    refetchHostDetails={refetchHostDetails}
                    isHostDetailsPolling={showRefetchSpinner}
                    hostSoftwareUpdatedAt={host.software_updated_at}
                    hostDisplayName={host?.hostname || ""}
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
                <AboutCard
                  className={
                    showUsersCard ? defaultCardClass : fullWidthCardClass
                  }
                  aboutData={aboutData}
                  munki={deviceMacAdminsData?.munki}
                />
                {showUsersCard && (
                  <UserCard
                    className={defaultCardClass}
                    platform={host.platform}
                    endUsers={host.end_users ?? []}
                    enableAddEndUser={false}
                    disableFullNameTooltip
                    disableGroupsTooltip
                  />
                )}
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
                    isLoading={isLoadingHost}
                    deviceUser
                    togglePolicyDetailsModal={togglePolicyDetailsModal}
                    hostPlatform={host?.platform || ""}
                    router={router}
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
          />
        )}
        {!!host && showOSSettingsModal && (
          <OSSettingsModal
            canResendProfiles={host.platform === "darwin"}
            platform={host.platform}
            hostMDMData={host.mdm}
            resendRequest={resendProfile}
            onProfileResent={refetchHostDetails}
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
                  {isLoadingHost ? (
                    <Spinner />
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
                url="https://www.fleetdm.com/better"
                text="About Fleet"
                newTab
              />
            </div>
          )}
        </div>
      </nav>
      {isDeviceUserError || enrollUrlError ? (
        <DeviceUserError
          isMobileView={isMobileView}
          isAuthenticationError={!!isAuthenticationError}
        />
      ) : (
        <div className={coreWrapperClassnames}>{renderDeviceUserPage()}</div>
      )}
      {showInfoModal && <InfoModal onCancel={toggleInfoModal} />}
    </div>
  );
};

export default DeviceUserPage;
