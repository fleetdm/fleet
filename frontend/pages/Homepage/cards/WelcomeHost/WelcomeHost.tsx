import React, { useState } from "react";
import PATHS from "router/paths";
import { Link } from "react-router";
import { useQuery } from "react-query";
import { useDispatch } from "react-redux";
import moment from "moment";

import { IHost } from "interfaces/host";
import { IHostPolicy } from "interfaces/host_policy";
import hostAPI from "services/entities/hosts"; // @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

import Spinner from "components/loaders/Spinner";
import Button from "components/buttons/Button";
import Modal from "components/modals/Modal";
import LaptopMac from "../../../../../assets/images/laptop-mac.png";
import LinkArrow from "../../../../../assets/images/icon-arrow-right-vibrant-blue-10x18@2x.png";
import IconDisabled from "../../../../../assets/images/icon-action-disable-red-16x16@2x.png";
import IconPassed from "../../../../../assets/images/icon-check-circle-green-16x16@2x.png";
import IconError from "../../../../../assets/images/icon-exclamation-circle-red-16x16@2x.png";
import IconChevron from "../../../../../assets/images/icon-chevron-purple-9x6@2x.png";

interface IHostResponse {
  host: IHost;
}

const baseClass = "welcome-host";
const HOST_ID = 8;

const WelcomeHost = (): JSX.Element => {
  const dispatch = useDispatch();
  const [refetchStartTime, setRefetchStartTime] = useState<number | null>(null);
  const [currentPolicyShown, setCurrentPolicyShown] = useState<IHostPolicy>();
  const [showPolicyModal, setShowPolicyModal] = useState<boolean>(false);
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
        setShowRefetchLoadingSpinner(returnedHost.refetch_requested);
        if (returnedHost.refetch_requested) {
          // Code duplicated from HostDetailsPage. See comments there.
          if (!refetchStartTime) {
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

  const onRefetchHost = async () => {
    if (host) {
      setShowRefetchLoadingSpinner(true);

      try {
        await hostAPI.refetch(host).then(() => {
          setRefetchStartTime(Date.now());
          setTimeout(() => fullyReloadHost(), 1000);
        });
      } catch (error) {
        console.log(error);
        dispatch(renderFlash("error", `Host "${host.hostname}" refetch error`));
        setShowRefetchLoadingSpinner(false);
      }
    }
  };

  const handlePolicyModal = (id: number) => {
    const policy = host?.policies.find((p) => p.id === id);

    if (policy) {
      setCurrentPolicyShown(policy);
      setShowPolicyModal(true);
    }
  };

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
          <p>
            <img alt="" src={IconDisabled} />
            Your device is not communicating with Fleet.
          </p>
          <p>Join the #fleet Slack channel for help troubleshooting.</p>
        </div>
      </div>
    );
  }

  if (host && !host.policies) {
    return (
      <div className={baseClass}>
        <div className={`${baseClass}__error`}>
          <p>
            <img alt="" src={IconDisabled} />
            No policies apply to your device.
          </p>
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
            <Link to={PATHS.HOST_DETAILS(host)} className="external-link">
              {host.hostname}
              <img alt="" src={LinkArrow} />
            </Link>
            <p>
              Your device is successully connected to this local preview of
              Fleet.
            </p>
          </div>
        </div>
        <div className={`${baseClass}__blurb`}>
          <p>
            Fleet already ran the following checks to assess the security of
            your device:{" "}
          </p>
        </div>
        <div className={`${baseClass}__policies`}>
          {host.policies?.slice(0, 10).map((p) => (
            <div className="policy-block">
              <div className="info">
                <img
                  alt={p.response}
                  src={p.response === "passing" ? IconPassed : IconError}
                />
                {p.query_name}
              </div>
              <Button
                variant="text-icon"
                onClick={() => handlePolicyModal(p.id)}
              >
                <img alt="" src={IconChevron} />
              </Button>
            </div>
          ))}
          {host.policies?.length > 10 && (
            <Link to={PATHS.HOST_DETAILS(host)} className="external-link">
              Go to Host details to see all checks
              <img alt="" src={LinkArrow} />
            </Link>
          )}
        </div>
        <div className={`${baseClass}__blurb`}>
          <p>
            Resolved a failing check? Refetch your device information to verify.
          </p>
        </div>
        <div className={`${baseClass}__refetch`}>
          <Button
            variant="blue-green"
            className={`refetch-spinner ${
              showRefetchLoadingSpinner ? "spin" : ""
            }`}
            onClick={onRefetchHost}
            disabled={showRefetchLoadingSpinner}
          >
            Refetch
          </Button>
          <span>Last updated {moment(host.detail_updated_at).fromNow()}</span>
        </div>
        {showPolicyModal && (
          <Modal
            title={currentPolicyShown?.query_name || ""}
            onExit={() => setShowPolicyModal(false)}
            className={`${baseClass}__policy-modal`}
          >
            <>
              <p>{currentPolicyShown?.query_description}</p>
              {currentPolicyShown?.resolution && (
                <p>
                  <b>Resolve:</b>
                  {currentPolicyShown.resolution}
                </p>
              )}
              <div className="done">
                <Button
                  variant="brand"
                  onClick={() => setShowPolicyModal(false)}
                >
                  Done
                </Button>
              </div>
            </>
          </Modal>
        )}
      </div>
    );
  }

  return <Spinner />;
};

export default WelcomeHost;
