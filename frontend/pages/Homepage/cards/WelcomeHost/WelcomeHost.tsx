import React, { useState } from "react";
import PATHS from "router/paths";
import { Link } from "react-router";
import { useQuery } from "react-query";
import { useDispatch } from "react-redux";

import { IHost } from "interfaces/host";
import hostAPI from "services/entities/hosts"; //@ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

import Spinner from "components/loaders/Spinner";
import LaptopMac from "../../../../../assets/images/laptop-mac.png";
import LinkArrow from "../../../../../assets/images/icon-arrow-right-vibrant-blue-10x18@2x.png";
import ErrorIcon from "../../../../../assets/images/icon-action-disable-red-16x16@2x.png";

interface IHostResponse {
  host: IHost;
}

const baseClass = "welcome-host";
const HOST_ID = 37;

const WelcomeHost = (): JSX.Element => {
  const dispatch = useDispatch();
  const [refetchStartTime, setRefetchStartTime] = useState<number | null>(null);
  const [
    showRefetchLoadingSpinner,
    setShowRefetchLoadingSpinner,
  ] = useState<boolean>(false);

  const {
    isLoading: isLoadingHost,
    data: host,
    error: loadingHostError,
    refetch: fullyReloadHost,
  } = useQuery<IHostResponse, Error, IHost>(
    ["host"],
    () => hostAPI.load(HOST_ID),
    {
      select: (data: IHostResponse) => data.host,
      onSuccess: (returnedHost) => {
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
                fullyReloadHost();
              }, 1000);
            } else {
              setShowRefetchLoadingSpinner(false);
            }
          } else {
            const totalElapsedTime = Date.now() - refetchStartTime;
            if (totalElapsedTime < 60000) {
              if (returnedHost.status === "online") {
                setTimeout(() => {
                  fullyReloadHost();
                }, 1000);
              } else {
                dispatch(
                  renderFlash(
                    "error",
                    `This host is offline. Please try refetching host vitals later.`
                  )
                );
                setShowRefetchLoadingSpinner(false);
              }
            } else {
              dispatch(
                renderFlash(
                  "error",
                  `We're having trouble fetching fresh vitals for this host. Please try again later.`
                )
              );
              setShowRefetchLoadingSpinner(false);
            }
          }
        }
      },
      onError: (error) => {
        console.log(error);
        dispatch(
          renderFlash("error", `Unable to load host. Please try again.`)
        );
      },
    }
  );
  
  if (isLoadingHost) {
    return (
      <div className={baseClass}>
        <div className={`${baseClass}__loading`}>
          <p>Adding your device to Fleet</p>
          <Spinner />
        </div>
      </div>
    );
  }

  if (loadingHostError) {
    return (
      <div className={baseClass}>
        <div className={`${baseClass}__error`}>
          <p><img alt="" src={ErrorIcon} />Your device is not communicating with Fleet.</p>
          <p>Join the #fleet Slack channel for help troubleshooting.</p>
        </div>
      </div>
    );
  }

  if (host && !host.policies) {
    return (
      <div className={baseClass}>
        <div className={`${baseClass}__error`}>
          <p><img alt="" src={ErrorIcon} />No policies apply to your device.</p>
          <p>Join the #fleet Slack channel for help troubleshooting.</p>
        </div>
      </div>
    );
  }

  if (host) {
    return (
      <div className={baseClass}>
        <div className={`${baseClass}__intro`}>
          <img alt="" src={LaptopMac} />
          <div className="info">
            <Link to={PATHS.HOST_DETAILS(host)}>
              {host.hostname}
              <img src={LinkArrow} />
            </Link>
            <p>Your device is successully connected to this local preview of Fleet.</p>
          </div>
        </div>
        <div className={`${baseClass}__blurb`}>
          <p>Fleet already ran the following checks to assess the security of your device: </p>
        </div>
        <div className={`${baseClass}__policies`}>
          {host.policies?.map((p) => (
            <div className="policy-block">{p.query_name}</div>
          ))}
        </div>
        <div className={`${baseClass}__blurb`}>
          <p>Resolved a failing check? Refetch your device information to verify.</p>
        </div>
        <div className={`${baseClass}__refetch`}></div>
      </div>
    );
  }

  return <Spinner />;
};

export default WelcomeHost;
