import React, { useState, useContext, useCallback, useEffect } from "react";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { useQuery } from "react-query";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";

import { pick, findIndex } from "lodash";

import { NotificationContext } from "context/notification";
import deviceUserAPI from "services/entities/device_user";
import {
  IDeviceMappingResponse,
  IMacadminsResponse,
  IDeviceUserResponse,
  IHostDevice,
} from "interfaces/host";
import { IHostPolicy } from "interfaces/policy";
import { IDeviceGlobalConfig } from "interfaces/config";
import DeviceUserError from "components/DeviceUserError";
// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";
import Spinner from "components/Spinner";
import Button from "components/buttons/Button";
import TabsWrapper from "components/TabsWrapper";
import InfoBanner from "components/InfoBanner";
import Icon from "components/Icon/Icon";
import { normalizeEmptyValues } from "utilities/helpers";
import PATHS from "router/paths";
import {
  DOCUMENT_TITLE_SUFFIX,
  HOST_ABOUT_DATA,
  HOST_SUMMARY_DATA,
} from "utilities/constants";

import HostSummaryCard from "../cards/HostSummary";
import AboutCard from "../cards/About";
import SoftwareCard from "../cards/Software";
import PoliciesCard from "../cards/Policies";
import InfoModal from "./InfoModal";

import FleetIcon from "../../../../../assets/images/fleet-avatar-24x24@2x.png";
import PolicyDetailsModal from "../cards/Policies/HostPoliciesTable/PolicyDetailsModal";
import AutoEnrollMdmModal from "./AutoEnrollMdmModal";
import ManualEnrollMdmModal from "./ManualEnrollMdmModal";
import OSSettingsModal from "../OSSettingsModal";
import ResetKeyModal from "./ResetKeyModal";
import BootstrapPackageModal from "../HostDetailsPage/modals/BootstrapPackageModal";

const baseClass = "device-user";

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
  const queryParams = location.query;
  const { renderFlash } = useContext(NotificationContext);

  const [isPremiumTier, setIsPremiumTier] = useState(false);
  const [showInfoModal, setShowInfoModal] = useState(false);
  const [showEnrollMdmModal, setShowEnrollMdmModal] = useState(false);
  const [showResetKeyModal, setShowResetKeyModal] = useState(false);
  const [refetchStartTime, setRefetchStartTime] = useState<number | null>(null);
  const [showRefetchSpinner, setShowRefetchSpinner] = useState(false);
  const [orgLogoURL, setOrgLogoURL] = useState("");
  const [selectedPolicy, setSelectedPolicy] = useState<IHostPolicy | null>(
    null
  );
  const [showPolicyDetailsModal, setShowPolicyDetailsModal] = useState(false);
  const [showOSSettingsModal, setShowOSSettingsModal] = useState(false);
  const [showBootstrapPackageModal, setShowBootstrapPackageModal] = useState(
    false
  );
  const [globalConfig, setGlobalConfig] = useState<IDeviceGlobalConfig | null>(
    null
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

  const refetchExtensions = () => {
    deviceMapping !== null && refetchDeviceMapping();
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
    data: { host } = { host: undefined },
    isLoading: isLoadingHost,
    error: loadingDeviceUserError,
    refetch: refetchHostDetails,
  } = useQuery<IDeviceUserResponse, Error>(
    ["host", deviceAuthToken],
    () => deviceUserAPI.loadHostDetails(deviceAuthToken),
    {
      enabled: !!deviceAuthToken,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      onSuccess: ({
        license,
        org_logo_url,
        global_config,
        host: responseHost,
      }) => {
        setShowRefetchSpinner(isRefetching(responseHost));
        setIsPremiumTier(license.tier === "premium");
        setOrgLogoURL(org_logo_url);
        setGlobalConfig(global_config);
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

  const summaryData = normalizeEmptyValues(pick(host, HOST_SUMMARY_DATA));

  const aboutData = normalizeEmptyValues(pick(host, HOST_ABOUT_DATA));

  const toggleInfoModal = useCallback(() => {
    setShowInfoModal(!showInfoModal);
  }, [showInfoModal, setShowInfoModal]);

  const toggleEnrollMdmModal = useCallback(() => {
    setShowEnrollMdmModal(!showEnrollMdmModal);
  }, [showEnrollMdmModal, setShowEnrollMdmModal]);

  const toggleResetKeyModal = useCallback(() => {
    setShowResetKeyModal(!showResetKeyModal);
  }, [showResetKeyModal, setShowResetKeyModal]);

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
        console.log(error);
        renderFlash("error", `Host "${host.display_name}" refetch error`);
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

  const turnOnMdmButton = (
    <Button variant="unstyled" onClick={toggleEnrollMdmModal}>
      <b>Turn on MDM</b>
    </Button>
  );

  const renderEnrollMdmModal = () => {
    return host?.dep_assigned_to_fleet ? (
      <AutoEnrollMdmModal onCancel={toggleEnrollMdmModal} />
    ) : (
      <ManualEnrollMdmModal
        onCancel={toggleEnrollMdmModal}
        token={deviceAuthToken}
      />
    );
  };

  const resetKeyButton = (
    <Button variant="unstyled" onClick={toggleResetKeyModal}>
      <b>Reset key</b>
    </Button>
  );

  const renderDeviceUserPage = () => {
    const failingPoliciesCount = host?.issues?.failing_policies_count || 0;
    const isMdmUnenrolled =
      host?.mdm.enrollment_status === "Off" || !host?.mdm.enrollment_status;

    const diskEncryptionBannersEnabled =
      globalConfig?.mdm.enabled_and_configured && host?.mdm.name === "Fleet";

    const showDiskEncryptionLogoutRestart =
      diskEncryptionBannersEnabled &&
      host?.mdm.macos_settings?.disk_encryption === "action_required" &&
      host?.mdm.macos_settings?.action_required === "log_out";
    const showDiskEncryptionKeyResetRequired =
      diskEncryptionBannersEnabled &&
      host?.mdm.macos_settings?.disk_encryption === "action_required" &&
      host?.mdm.macos_settings?.action_required === "rotate_key";

    const tabPaths = [
      PATHS.DEVICE_USER_DETAILS(deviceAuthToken),
      PATHS.DEVICE_USER_DETAILS_SOFTWARE(deviceAuthToken),
      PATHS.DEVICE_USER_DETAILS_POLICIES(deviceAuthToken),
    ];

    const findSelectedTab = (pathname: string) =>
      findIndex(tabPaths, (x) => x.startsWith(pathname.split("?")[0]));

    return (
      <div className="core-wrapper">
        {!host || isLoadingHost ? (
          <Spinner />
        ) : (
          <div className={`${baseClass} main-content`}>
            {host?.platform === "darwin" &&
              isMdmUnenrolled &&
              globalConfig?.mdm.enabled_and_configured && (
                // Turn on MDM banner
                <InfoBanner color="yellow" cta={turnOnMdmButton}>
                  Mobile device management (MDM) is off. MDM allows your
                  organization to change settings and install software. This
                  lets your organization keep your device up to date so you
                  don’t have to.
                </InfoBanner>
              )}
            {showDiskEncryptionLogoutRestart && (
              // MDM - Disk Encryption: Logout or restart banner
              <InfoBanner color="yellow">
                Disk encryption: Log out of your device or restart to turn on
                disk encryption. Then, select <strong>Refetch</strong>. This
                prevents unauthorized access to the information on your device.
              </InfoBanner>
            )}
            {showDiskEncryptionKeyResetRequired && (
              // MDM - Disk Encryption: Reset key required banner
              <InfoBanner color="yellow" cta={resetKeyButton}>
                Disk encryption: Reset your disk encryption key. This lets your
                organization help you unlock your device if you forget your
                password.
              </InfoBanner>
            )}
            <HostSummaryCard
              summaryData={summaryData}
              bootstrapPackageData={bootstrapPackageData}
              isPremiumTier={isPremiumTier}
              toggleOSSettingsModal={toggleOSSettingsModal}
              hostMdmProfiles={host?.mdm.profiles ?? []}
              mdmName={deviceMacAdminsData?.mobile_device_management?.name}
              showRefetchSpinner={showRefetchSpinner}
              onRefetchHost={onRefetchHost}
              renderActionButtons={renderActionButtons}
              osSettings={host?.mdm.os_settings}
              deviceUser
            />
            <TabsWrapper className={`${baseClass}__tabs-wrapper`}>
              <Tabs
                selectedIndex={findSelectedTab(location.pathname)}
                onSelect={(i) => router.push(tabPaths[i])}
              >
                <TabList>
                  <Tab>Details</Tab>
                  <Tab>Software</Tab>
                  {isPremiumTier && (
                    <Tab>
                      <div>
                        {failingPoliciesCount > 0 && (
                          <span className="count">{failingPoliciesCount}</span>
                        )}
                        Policies
                      </div>
                    </Tab>
                  )}
                </TabList>
                <TabPanel>
                  <AboutCard
                    aboutData={aboutData}
                    deviceMapping={deviceMapping}
                    munki={deviceMacAdminsData?.munki}
                  />
                </TabPanel>
                <TabPanel>
                  <SoftwareCard
                    id={deviceAuthToken}
                    isFleetdHost={!!host.orbit_version}
                    router={router}
                    pathname={location.pathname}
                    queryParams={queryParams}
                    isMyDevicePage
                    teamId={host.team_id || 0}
                  />
                </TabPanel>
                {isPremiumTier && (
                  <TabPanel>
                    <PoliciesCard
                      policies={host?.policies || []}
                      isLoading={isLoadingHost}
                      deviceUser
                      togglePolicyDetailsModal={togglePolicyDetailsModal}
                    />
                  </TabPanel>
                )}
              </Tabs>
            </TabsWrapper>
            {showInfoModal && <InfoModal onCancel={toggleInfoModal} />}
            {showEnrollMdmModal && renderEnrollMdmModal()}
            {showResetKeyModal && (
              <ResetKeyModal
                onClose={toggleResetKeyModal}
                deviceAuthToken={deviceAuthToken}
              />
            )}
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
      </div>
    );
  };

  return (
    <div className="app-wrap">
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
