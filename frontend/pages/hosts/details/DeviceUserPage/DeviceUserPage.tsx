import React, { useState, useContext, useCallback, useEffect } from "react";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { useQuery } from "react-query";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";

import { pick, findIndex } from "lodash";

import { NotificationContext } from "context/notification";
import deviceUserAPI, {
  IGetDeviceCertificatesResponse,
} from "services/entities/device_user";
import diskEncryptionAPI from "services/entities/disk_encryption";
import {
  IDeviceMappingResponse,
  IMacadminsResponse,
  IDeviceUserResponse,
  IHostDevice,
} from "interfaces/host";
import { IHostPolicy } from "interfaces/policy";
import { IDeviceGlobalConfig } from "interfaces/config";
import { IHostSoftware } from "interfaces/software";
import { IHostCertificate } from "interfaces/certificates";
import { isAppleDevice } from "interfaces/platform";

import DeviceUserError from "components/DeviceUserError";
// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";
import Spinner from "components/Spinner";
import Button from "components/buttons/Button";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import Icon from "components/Icon/Icon";
import FlashMessage from "components/FlashMessage";

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
import { getErrorMessage } from "./helpers";

import FleetIcon from "../../../../../assets/images/fleet-avatar-24x24@2x.png";
import PolicyDetailsModal from "../cards/Policies/HostPoliciesTable/PolicyDetailsModal";
import AutoEnrollMdmModal from "./AutoEnrollMdmModal";
import ManualEnrollMdmModal from "./ManualEnrollMdmModal";
import CreateLinuxKeyModal from "./CreateLinuxKeyModal";
import OSSettingsModal from "../OSSettingsModal";
import BootstrapPackageModal from "../HostDetailsPage/modals/BootstrapPackageModal";
import { parseHostSoftwareQueryParams } from "../cards/Software/HostSoftware";
import SelfService from "../cards/Software/SelfService";
import SoftwareDetailsModal from "../cards/Software/SoftwareDetailsModal";
import DeviceUserBanners from "./components/DeviceUserBanners";
import CertificateDetailsModal from "../modals/CertificateDetailsModal";
import CertificatesCard from "../cards/Certificates";

const baseClass = "device-user";

const PREMIUM_TAB_PATHS = [
  PATHS.DEVICE_USER_DETAILS,
  PATHS.DEVICE_USER_DETAILS_SELF_SERVICE,
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

  const { renderFlash, notification, hideFlash } = useContext(
    NotificationContext
  );

  const [showInfoModal, setShowInfoModal] = useState(false);
  const [showEnrollMdmModal, setShowEnrollMdmModal] = useState(false);
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
    selectedSoftwareDetails,
    setSelectedSoftwareDetails,
  ] = useState<IHostSoftware | null>(null);

  // certificates states
  const [
    selectedCertificate,
    setSelectedCertificate,
  ] = useState<IHostCertificate | null>(null);
  const [certificatePage, setCertificatePage] = useState(
    DEFAULT_CERTIFICATES_PAGE
  );

  const { data: deviceMapping, refetch: refetchDeviceMapping } = useQuery(
    ["deviceMapping", deviceAuthToken],
    () =>
      deviceUserAPI.loadHostDetailsExtension(deviceAuthToken, "device_mapping"),
    {
      enabled: !!deviceAuthToken,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: IDeviceMappingResponse) => data.device_mapping,
    }
  );

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
    Array<{ scope: string; token: string; page: number; perPage: number }>
  >(
    [
      {
        scope: "device-certificates",
        token: deviceAuthToken,
        page: certificatePage,
        perPage: DEFAULT_CERTIFICATES_PAGE_SIZE,
      },
    ],
    ({ queryKey: [{ token, page, perPage }] }) =>
      deviceUserAPI.getDeviceCertificates(token, page, perPage),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      // FIXME: is it worth disabling for unsupported platforms? we'd have to workaround the a
      // catch-22 where we need to know the platform to know if it's supported but we also need to
      // be able to include the cert refetch in the hosts query hook.
      enabled: !!deviceUserAPI,
    }
  );

  const refetchExtensions = () => {
    deviceMapping !== null && refetchDeviceMapping();
    deviceCertificates && refetchDeviceCertificates();
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
    error: loadingDeviceUserError,
    refetch: refetchHostDetails,
  } = useQuery<IDeviceUserResponse, Error>(
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
        setShowRefetchSpinner(isRefetching(responseHost));
        if (isRefetching(responseHost)) {
          // If the API reports that a Fleet refetch request is pending, we want to check back for fresh
          // host details. Here we set a one second timeout and poll the API again using
          // fullyReloadHost. We will repeat this process with each onSuccess cycle for a total of
          // 60 seconds or until the API reports that the Fleet refetch request has been resolved
          // or that the host has gone offline.
          if (!refetchStartTime) {
            // If our 60 second timer wasn't already started (e.g., if a refetch was pending when
            // the first page loads), we start it now if the host is online. If the host is offline,
            // we skip the refetch on page load.
            if (responseHost.status === "online") {
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
              if (responseHost.status === "online") {
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
          // exit early because refectch is pending so we can avoid unecessary steps below
        }
      },
    }
  );

  const {
    host,
    license,
    org_logo_url: orgLogoURL = "",
    org_contact_url: orgContactURL = "",
    global_config: globalConfig = null as IDeviceGlobalConfig | null,
    self_service: hasSelfService = false,
  } = dupResponse || {};
  const isPremiumTier = license?.tier === "premium";
  const isAppleHost = isAppleDevice(host?.platform);

  const summaryData = normalizeEmptyValues(pick(host, HOST_SUMMARY_DATA));

  const aboutData = normalizeEmptyValues(pick(host, HOST_ABOUT_DATA));

  const toggleInfoModal = useCallback(() => {
    setShowInfoModal(!showInfoModal);
  }, [showInfoModal, setShowInfoModal]);

  const toggleEnrollMdmModal = useCallback(() => {
    setShowEnrollMdmModal(!showEnrollMdmModal);
  }, [showEnrollMdmModal, setShowEnrollMdmModal]);

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

  const onRefetchHost = async () => {
    if (host) {
      setShowRefetchSpinner(true);
      try {
        await deviceUserAPI.refetch(deviceAuthToken);
        setRefetchStartTime(Date.now());
        setTimeout(() => {
          refetchHostDetails();
          refetchExtensions();
        }, 1000);
      } catch (error) {
        renderFlash("error", getErrorMessage(error, host.display_name));
        setShowRefetchSpinner(false);
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
        <Button onClick={() => setShowInfoModal(true)} variant="text-icon">
          <>
            Info <Icon name="info" size="small" />
          </>
        </Button>
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

    const findSelectedTab = (pathname: string) =>
      findIndex(tabPaths, (x) => x.startsWith(pathname.split("?")[0]));
    if (!isLoadingHost && host && findSelectedTab(location.pathname) === -1) {
      router.push(tabPaths[0]);
    }

    // Note: API response global_config is misnamed because the backend actually returns the global
    // or team config (as applicable)
    const isSoftwareEnabled = !!globalConfig?.features
      ?.enable_software_inventory;

    return (
      <div className="core-wrapper">
        {!host || isLoadingHost || isLoadingDeviceCertificates ? (
          <Spinner />
        ) : (
          <div className={`${baseClass} main-content`}>
            <DeviceUserBanners
              hostPlatform={host.platform}
              hostOsVersion={host.os_version}
              mdmEnrollmentStatus={host.mdm.enrollment_status}
              mdmEnabledAndConfigured={
                !!globalConfig?.mdm.enabled_and_configured
              }
              connectedToFleetMdm={!!host.mdm.connected_to_fleet}
              macDiskEncryptionStatus={
                host.mdm.macos_settings?.disk_encryption ?? null
              }
              diskEncryptionActionRequired={
                host.mdm.macos_settings?.action_required ?? null
              }
              onTurnOnMdm={toggleEnrollMdmModal}
              onTriggerEscrowLinuxKey={onTriggerEscrowLinuxKey}
              diskEncryptionOSSetting={host.mdm.os_settings?.disk_encryption}
              diskIsEncrypted={host.disk_encryption_enabled}
              diskEncryptionKeyAvailable={host.mdm.encryption_key_available}
            />
            <HostSummaryCard
              summaryData={summaryData}
              bootstrapPackageData={bootstrapPackageData}
              isPremiumTier={isPremiumTier}
              toggleOSSettingsModal={toggleOSSettingsModal}
              hostSettings={host?.mdm.profiles ?? []}
              showRefetchSpinner={showRefetchSpinner}
              onRefetchHost={onRefetchHost}
              renderActionDropdown={renderActionButtons}
              osSettings={host?.mdm.os_settings}
              deviceUser
            />
            <TabNav className={`${baseClass}__tab-nav`}>
              <Tabs
                selectedIndex={findSelectedTab(location.pathname)}
                onSelect={(i) => router.push(tabPaths[i])}
              >
                <TabList>
                  <Tab>
                    <TabText>Details</TabText>
                  </Tab>
                  {isPremiumTier && isSoftwareEnabled && hasSelfService && (
                    <Tab>
                      <TabText>Self-service</TabText>
                    </Tab>
                  )}
                  {isSoftwareEnabled && (
                    <Tab>
                      <TabText>Software</TabText>
                    </Tab>
                  )}
                  {isPremiumTier && (
                    <Tab>
                      <TabText count={failingPoliciesCount} isErrorCount>
                        Policies
                      </TabText>
                    </Tab>
                  )}
                </TabList>
                <TabPanel className={`${baseClass}__details-panel`}>
                  <AboutCard
                    aboutData={aboutData}
                    deviceMapping={deviceMapping}
                    munki={deviceMacAdminsData?.munki}
                  />
                  {isAppleHost && !!deviceCertificates?.certificates.length && (
                    <CertificatesCard
                      isMyDevicePage
                      data={deviceCertificates}
                      isError={isErrorDeviceCertificates}
                      page={certificatePage}
                      pageSize={DEFAULT_CERTIFICATES_PAGE_SIZE}
                      hostPlatform={host.platform}
                      onSelectCertificate={onSelectCertificate}
                      onNextPage={() => setCertificatePage(certificatePage + 1)}
                      onPreviousPage={() =>
                        setCertificatePage(certificatePage - 1)
                      }
                    />
                  )}
                </TabPanel>
                {isPremiumTier && isSoftwareEnabled && hasSelfService && (
                  <TabPanel>
                    <SelfService
                      contactUrl={orgContactURL}
                      deviceToken={deviceAuthToken}
                      isSoftwareEnabled
                      pathname={location.pathname}
                      queryParams={parseHostSoftwareQueryParams(location.query)}
                      router={router}
                    />
                  </TabPanel>
                )}
                {isSoftwareEnabled && (
                  <TabPanel>
                    <SoftwareCard
                      id={deviceAuthToken}
                      softwareUpdatedAt={host.software_updated_at}
                      hostCanWriteSoftware={!!host.orbit_version}
                      router={router}
                      pathname={location.pathname}
                      queryParams={parseHostSoftwareQueryParams(location.query)}
                      isMyDevicePage
                      platform={host.platform}
                      hostTeamId={host.team_id || 0}
                      isSoftwareEnabled={isSoftwareEnabled}
                      onShowSoftwareDetails={setSelectedSoftwareDetails}
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
            {showInfoModal && <InfoModal onCancel={toggleInfoModal} />}
            {showEnrollMdmModal &&
              (host.dep_assigned_to_fleet ? (
                <AutoEnrollMdmModal
                  host={host}
                  onCancel={toggleEnrollMdmModal}
                />
              ) : (
                <ManualEnrollMdmModal
                  host={host}
                  onCancel={toggleEnrollMdmModal}
                  token={deviceAuthToken}
                />
              ))}
          </div>
        )}
        {!!host && showPolicyDetailsModal && (
          <PolicyDetailsModal
            onCancel={onCancelPolicyDetailsModal}
            policy={selectedPolicy}
          />
        )}
        {!!host && showOSSettingsModal && (
          <OSSettingsModal
            canResendProfiles={false}
            hostId={host.id}
            platform={host.platform}
            hostMDMData={host.mdm}
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
        {selectedSoftwareDetails && !!host && (
          <SoftwareDetailsModal
            hostDisplayName={host.display_name}
            software={selectedSoftwareDetails}
            onExit={() => setSelectedSoftwareDetails(null)}
            hideInstallDetails
          />
        )}
        {selectedCertificate && (
          <CertificateDetailsModal
            certificate={selectedCertificate}
            onExit={() => setSelectedCertificate(null)}
          />
        )}
      </div>
    );
  };

  return (
    <div className="app-wrap">
      <UnsupportedScreenSize />
      <FlashMessage
        fullWidth
        notification={notification}
        onRemoveFlash={hideFlash}
        pathname={location.pathname}
      />
      <nav className="site-nav-container">
        <div className="site-nav-content">
          <ul className="site-nav-list">
            <li className="site-nav-item dup-org-logo" key="dup-org-logo">
              <div className="site-nav-item__logo-wrapper">
                <div className="site-nav-item__logo">
                  <OrgLogoIcon className="logo" src={orgLogoURL || FleetIcon} />
                </div>
              </div>
            </li>
          </ul>
        </div>
      </nav>
      {loadingDeviceUserError ? <DeviceUserError /> : renderDeviceUserPage()}
    </div>
  );
};

export default DeviceUserPage;
