import React, { useContext, useState } from "react";
import PATHS from "router/paths";
import { Link } from "react-router";
import { useQuery } from "react-query";
import { formatDistanceToNow } from "date-fns";

import { NotificationContext } from "context/notification";
import { IHost } from "interfaces/host";
import { IHostPolicy } from "interfaces/policy";
import hostAPI from "services/entities/hosts";

import Spinner from "components/Spinner";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import LaptopMac from "../../../../../assets/images/laptop-mac.png";
import LinkArrow from "../../../../../assets/images/icon-arrow-right-vibrant-blue-10x18@2x.png";
import IconDisabled from "../../../../../assets/images/icon-action-disable-red-16x16@2x.png";
import IconPassed from "../../../../../assets/images/icon-check-circle-green-16x16@2x.png";
import IconError from "../../../../../assets/images/icon-exclamation-circle-red-16x16@2x.png";
import IconChevron from "../../../../../assets/images/icon-chevron-purple-9x6@2x.png";
import SlackButton from "../../../../../assets/images/slack-button-get-help.png";

interface IHostResponse {
  host: IHost;
}

interface IWelcomeHostCardProps {
  totalsHostsCount: number;
  toggleAddHostsModal: (showAddHostsModal: boolean) => void;
}

const baseClass = "welcome-host";
const HOST_ID = 1;
const policyPass = "pass";
const policyFail = "fail";

const WelcomeHost = ({
  totalsHostsCount,
  toggleAddHostsModal,
}: IWelcomeHostCardProps): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);
  const [refetchStartTime, setRefetchStartTime] = useState<number | null>(null);
  const [currentPolicyShown, setCurrentPolicyShown] = useState<IHostPolicy>();
  const [showPolicyModal, setShowPolicyModal] = useState<boolean>(false);
  const [isPoliciesEmpty, setIsPoliciesEmpty] = useState<boolean>(false);
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
    () => hostAPI.loadHostDetails(HOST_ID),
    {
      retry: false,
      select: (data: IHostResponse) => data.host,
      onSuccess: (returnedHost) => {
        setShowRefetchLoadingSpinner(returnedHost.refetch_requested);

        const anyPassingOrFailingPolicy = returnedHost?.policies?.find(
          (p) => p.response === policyPass || p.response === policyFail
        );
        setIsPoliciesEmpty(typeof anyPassingOrFailingPolicy === "undefined");

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
                renderFlash(
                  "error",
                  `This host is offline. Please try refetching host vitals later.`
                );
                setShowRefetchLoadingSpinner(false);
              }
            } else {
              renderFlash(
                "error",
                `We're having trouble fetching fresh vitals for this host. Please try again later.`
              );
              setShowRefetchLoadingSpinner(false);
            }
          }
        }
      },
      onError: (error) => {
        console.error(error);
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
        console.error(error);
        renderFlash("error", `Host "${host.hostname}" refetch error`);
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
          <Spinner />
        </div>
      </div>
    );
  }

  if (loadingHostError) {
    return (
      <div className={baseClass}>
        <div className={`${baseClass}__empty-hosts`}>
          <p>Add your personal device to assess the security of your device.</p>
          <p>
            In Fleet, laptops, workstations, and servers are referred to as
            &quot;hosts.&quot;
          </p>
          <Button
            onClick={toggleAddHostsModal}
            className={`${baseClass}__add-host`}
            variant="brand"
          >
            <span>Add hosts</span>
          </Button>
        </div>
      </div>
    );
  }

  if (totalsHostsCount === 1 && host && host.status === "offline") {
    return (
      <div className={baseClass}>
        <div className={`${baseClass}__error`}>
          <p>
            <img
              alt="Disabled icon"
              className="icon-disabled"
              src={IconDisabled}
            />
            Your device is not communicating with Fleet.
          </p>
          <p>Join the #fleet Slack channel for help troubleshooting.</p>
          <a
            target="_blank"
            rel="noreferrer"
            href="https://osquery.slack.com/archives/C01DXJL16D8"
          >
            <img
              alt="Get help on Slack"
              className="button-slack"
              src={SlackButton}
            />
          </a>
        </div>
      </div>
    );
  }

  if (isPoliciesEmpty) {
    return (
      <div className={baseClass}>
        <div className={`${baseClass}__error`}>
          <p className="error-message">
            <img
              alt="Disabled icon"
              className="icon-disabled"
              src={IconDisabled}
            />
            No policies apply to your device.
          </p>
          <p>Join the #fleet Slack channel for help troubleshooting.</p>
          <a
            target="_blank"
            rel="noreferrer"
            href="https://osquery.slack.com/archives/C01DXJL16D8"
          >
            <img
              alt="Get help on Slack"
              className="button-slack"
              src={SlackButton}
            />
          </a>
        </div>
      </div>
    );
  }

  if (totalsHostsCount === 1 && host && host.status === "online") {
    return (
      <div className={baseClass}>
        <div className={`${baseClass}__intro`}>
          <img alt="" src={LaptopMac} />
          <div className="info">
            <Link to={PATHS.HOST_DETAILS(host)} className="external-link">
              {host.hostname}
              <img alt="" src={LinkArrow} />
            </Link>
            <p>Your host is successfully connected to Fleet.</p>
          </div>
        </div>
        <div className={`${baseClass}__blurb`}>
          <p>
            Fleet already ran the following policies to assess the security of
            your device:{" "}
          </p>
        </div>
        <div className={`${baseClass}__policies`}>
          {host.policies?.slice(0, 3).map((p) => {
            if (p.response) {
              return (
                <Button
                  variant="text-icon"
                  onClick={() => handlePolicyModal(p.id)}
                >
                  <div className="policy-block">
                    <div className="info">
                      <img
                        alt={p.response}
                        src={p.response === policyPass ? IconPassed : IconError}
                      />
                      {p.name}
                    </div>
                    <img alt="" src={IconChevron} />
                  </div>
                </Button>
              );
            }

            return null;
          })}
          {host.policies?.length > 3 && (
            <Link to={PATHS.HOST_DETAILS(host)} className="external-link">
              Go to Host details to see all policies
              <img alt="" src={LinkArrow} />
            </Link>
          )}
        </div>
        <div className={`${baseClass}__blurb`}>
          <p>Resolved a failing policy? Refetch your host vitals to verify.</p>
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
          <span>
            Last updated{" "}
            {formatDistanceToNow(new Date(host.detail_updated_at), {
              addSuffix: true,
            })}
          </span>
        </div>
        {showPolicyModal && (
          <Modal
            title={currentPolicyShown?.name || ""}
            onExit={() => setShowPolicyModal(false)}
            onEnter={() => setShowPolicyModal(false)}
            className={`${baseClass}__policy-modal`}
          >
            <>
              <p>{currentPolicyShown?.description}</p>
              {currentPolicyShown?.resolution && (
                <p>
                  <b>Resolve:</b> {currentPolicyShown.resolution}
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
