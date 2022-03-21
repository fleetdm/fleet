import React, { useState, useCallback } from "react";
import { useDispatch } from "react-redux";
import { Params } from "react-router/lib/Router";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";

import classnames from "classnames";
import { pick } from "lodash";

import deviceUserAPI from "services/entities/device_user";
import { IHost, IDeviceMappingResponse } from "interfaces/host";
import { ISoftware } from "interfaces/software";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
import ReactTooltip from "react-tooltip";
import PageError from "components/PageError";
// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";
import Spinner from "components/Spinner";
import Button from "components/buttons/Button";
import TabsWrapper from "components/TabsWrapper";
import {
  humanHostDetailUpdated,
  normalizeEmptyValues,
  wrapFleetHelper,
} from "fleet/helpers";

import HostSummaryCard from "../cards/HostSummary";
import AboutCard from "../cards/About";
import SoftwareCard from "../cards/Software";
import InfoModal from "./InfoModal";

import InfoIcon from "../../../../../assets/images/icon-info-purple-14x14@2x.png";
import FleetIcon from "../../../../../assets/images/fleet-avatar-24x24@2x.png";

const baseClass = "device-user";

interface IDeviceUserPageProps {
  params: Params;
}

interface IHostResponse {
  host: IHost;
  org_logo_url: string;
}

const DeviceUserPage = ({
  params: { device_auth_token },
}: IDeviceUserPageProps): JSX.Element => {
  const deviceAuthToken = device_auth_token;
  const dispatch = useDispatch();
  const handlePageError = useErrorHandler();

  const [showInfoModal, setShowInfoModal] = useState<boolean>(false);

  const [refetchStartTime, setRefetchStartTime] = useState<number | null>(null);
  const [showRefetchSpinner, setShowRefetchSpinner] = useState<boolean>(false);
  const [hostSoftware, setHostSoftware] = useState<ISoftware[]>([]);
  const [host, setHost] = useState<IHost | null>();
  const [orgLogoURL, setOrgLogoURL] = useState<string>("");

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
      select: (data: IDeviceMappingResponse) =>
        data.device_mapping &&
        data.device_mapping.filter(
          (deviceUser) => deviceUser.email && deviceUser.email.length
        ),
    }
  );

  const refetchExtensions = () => {
    deviceMapping !== null && refetchDeviceMapping();
  };

  const {
    isLoading: isLoadingHost,
    error: loadingDeviceUserError,
    refetch: refetchHostDetails,
  } = useQuery<IHostResponse, Error, IHostResponse>(
    ["host", deviceAuthToken],
    () => deviceUserAPI.loadHostDetails(deviceAuthToken),
    {
      enabled: !!deviceAuthToken,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: IHostResponse) => data,
      onSuccess: (returnedHost) => {
        setShowRefetchSpinner(returnedHost.host.refetch_requested);
        if (returnedHost.host.refetch_requested) {
          // If the API reports that a Fleet refetch request is pending, we want to check back for fresh
          // host details. Here we set a one second timeout and poll the API again using
          // fullyReloadHost. We will repeat this process with each onSuccess cycle for a total of
          // 60 seconds or until the API reports that the Fleet refetch request has been resolved
          // or that the host has gone offline.
          if (!refetchStartTime) {
            // If our 60 second timer wasn't already started (e.g., if a refetch was pending when
            // the first page loads), we start it now if the host is online. If the host is offline,
            // we skip the refetch on page load.
            if (returnedHost.host.status === "online") {
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
              if (returnedHost.host.status === "online") {
                setTimeout(() => {
                  refetchHostDetails();
                  refetchExtensions();
                }, 1000);
              } else {
                dispatch(
                  renderFlash(
                    "error",
                    `This host is offline. Please try refetching host vitals later.`
                  )
                );
                setShowRefetchSpinner(false);
              }
            } else {
              dispatch(
                renderFlash(
                  "error",
                  `We're having trouble fetching fresh vitals for this host. Please try again later.`
                )
              );
              setShowRefetchSpinner(false);
            }
          }
          return; // exit early because refectch is pending so we can avoid unecessary steps below
        }
        setHostSoftware(returnedHost.host.software);
        setHost(returnedHost.host);
        setOrgLogoURL(returnedHost.org_logo_url);
      },
      onError: (error) => handlePageError(error),
    }
  );

  const titleData = normalizeEmptyValues(
    pick(host, [
      "status",
      "issues",
      "memory",
      "cpu_type",
      "os_version",
      "osquery_version",
      "enroll_secret_name",
      "detail_updated_at",
      "percent_disk_space_available",
      "gigs_disk_space_available",
    ])
  );

  const aboutData = normalizeEmptyValues(
    pick(host, [
      "seen_time",
      "uptime",
      "last_enrolled_at",
      "hardware_model",
      "hardware_serial",
      "primary_ip",
    ])
  );

  const toggleInfoModal = useCallback(() => {
    setShowInfoModal(!showInfoModal);
  }, [showInfoModal, setShowInfoModal]);

  const onRefetchHost = async () => {
    if (host) {
      // Once the user clicks to refetch, the refetch loading spinner should continue spinning
      // unless there is an error. The spinner state is also controlled in the fullyReloadHost
      // method.
      setShowRefetchSpinner(true);
      try {
        await deviceUserAPI.refetch(deviceAuthToken).then(() => {
          setRefetchStartTime(Date.now());
          setTimeout(() => {
            refetchHostDetails();
            refetchExtensions();
          }, 1000);
        });
      } catch (error) {
        console.log(error);
        dispatch(renderFlash("error", `Host "${host.hostname}" refetch error`));
        setShowRefetchSpinner(false);
      }
    }
  };

  const renderActionButtons = () => {
    return (
      <div className={`${baseClass}__action-button-container`}>
        <Button onClick={() => setShowInfoModal(true)} variant="text-icon">
          <>
            Info <img src={InfoIcon} alt="Host info icon" />
          </>
        </Button>
      </div>
    );
  };

  const renderRefetch = () => {
    const isOnline = host?.status === "online";

    return (
      <>
        <div
          className="refetch"
          data-tip
          data-for="refetch-tooltip"
          data-tip-disable={isOnline || showRefetchSpinner}
        >
          <Button
            className={`
              button
              button--unstyled
              ${!isOnline ? "refetch-offline" : ""} 
              ${showRefetchSpinner ? "refetch-spinner" : "refetch-btn"}
            `}
            disabled={!isOnline}
            onClick={onRefetchHost}
          >
            {showRefetchSpinner
              ? "Fetching fresh vitals...this may take a moment"
              : "Refetch"}
          </Button>
        </div>
        <ReactTooltip
          place="bottom"
          type="dark"
          effect="solid"
          id="refetch-tooltip"
          backgroundColor="#3e4771"
        >
          <span className={`${baseClass}__tooltip-text`}>
            You can’t fetch data from <br /> an offline host.
          </span>
        </ReactTooltip>
      </>
    );
  };

<<<<<<< HEAD:frontend/pages/hosts/DeviceUserPage/DeviceUserPage.tsx
  const renderDeviceUser = () => {
    const numUsers = deviceMapping?.length;
    if (numUsers) {
      return (
        <div className="info-grid__block">
          <span className="info-grid__header">Used by</span>
          <span className="info-grid__data">
            {numUsers === 1 && deviceMapping ? (
              deviceMapping[0].email || "---"
            ) : (
              <span className={`${baseClass}__device-mapping`}>
                <span
                  className="device-user-list"
                  data-tip
                  data-for="device-user-tooltip"
                >
                  {`${numUsers} users`}
                </span>
                <ReactTooltip
                  place="top"
                  type="dark"
                  effect="solid"
                  id="device-user-tooltip"
                  backgroundColor="#3e4771"
                >
                  <div
                    className={`${baseClass}__tooltip-text device-user-tooltip`}
                  >
                    {deviceMapping?.map((user, i, arr) => (
                      <span key={user.email}>{`${user.email}${
                        i < arr.length - 1 ? ", " : ""
                      }`}</span>
                    ))}
                  </div>
                </ReactTooltip>
              </span>
            )}
          </span>
        </div>
      );
    }
    return null;
  };

  const renderDiskSpace = () => {
    if (
      host &&
      (host.gigs_disk_space_available > 0 ||
        host.percent_disk_space_available > 0)
    ) {
      return (
        <span className="info-flex__data">
          <div className="info-flex__disk-space">
            <div
              className={
                titleData.percent_disk_space_available > 20
                  ? "info-flex__disk-space-used"
                  : "info-flex__disk-space-warning"
              }
              style={{
                width: `${100 - titleData.percent_disk_space_available}%`,
              }}
            />
          </div>
          {titleData.gigs_disk_space_available} GB available
        </span>
      );
    }
    return <span className="info-flex__data">No data available</span>;
  };

=======
>>>>>>> 8c170f37a (Refactor device user to have components with host details):frontend/pages/hosts/details/DeviceUserPage/DeviceUserPage.tsx
  const renderShowInfoModal = () => <InfoModal onCancel={toggleInfoModal} />;

  const statusClassName = classnames("status", `status--${host?.status}`);

  const renderDeviceUserPage = () => {
    return (
      <div className="fleet-desktop-wrapper">
        {isLoadingHost ? (
          <Spinner />
        ) : (
          <div className={`${baseClass} body-wrap`}>
            <div className="header title">
              <div className="title__inner">
                <div className="hostname-container">
                  <h1 className="hostname">My device</h1>
                  <p className="last-fetched">
                    {`Last reported vitals ${humanHostDetailUpdated(
                      titleData.detail_updated_at
                    )}`}
                    &nbsp;
                  </p>
                  {renderRefetch()}
                </div>
              </div>
              {renderActionButtons()}
            </div>
            <HostSummaryCard
              statusClassName={statusClassName}
              titleData={titleData}
              wrapFleetHelper={wrapFleetHelper}
              deviceUser
            />
            <TabsWrapper>
              <Tabs>
                <TabList>
                  <Tab>Details</Tab>
                  <Tab>Software</Tab>
                </TabList>
                <TabPanel>
                  <AboutCard
                    aboutData={aboutData}
                    deviceMapping={deviceMapping}
                    wrapFleetHelper={wrapFleetHelper}
                    deviceUser
                  />
                </TabPanel>
                <TabPanel>
                  <SoftwareCard
                    isLoading={isLoadingHost}
                    software={hostSoftware}
                    deviceUser
                  />
                </TabPanel>
              </Tabs>
            </TabsWrapper>
            {showInfoModal && renderShowInfoModal()}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="app-wrap">
      <nav className="site-nav">
        <div className="site-nav-container">
          <ul className="site-nav-list">
            <li className={`site-nav-item--logo`} key={`nav-item`}>
              <OrgLogoIcon className="logo" src={orgLogoURL || FleetIcon} />
            </li>
          </ul>
        </div>
      </nav>
      {loadingDeviceUserError ? <PageError /> : renderDeviceUserPage()}
    </div>
  );
};

export default DeviceUserPage;
