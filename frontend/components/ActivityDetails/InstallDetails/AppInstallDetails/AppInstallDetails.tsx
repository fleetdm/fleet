// Used on: Dashboard > activity, Host details > past activity
// Also used on Self-service failed install details

import React, { useState } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import { formatDistanceToNow } from "date-fns";

import mdmApi, { IGetMdmCommandResultsResponse } from "services/entities/mdm";
import deviceUserAPI, {
  IGetVppInstallCommandResultsResponse,
} from "services/entities/device_user";

import { IHostSoftware, SoftwareInstallStatus } from "interfaces/software";
import { IMdmCommandResult } from "interfaces/mdm";

import InventoryVersions from "pages/hosts/details/components/InventoryVersions";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import Textarea from "components/Textarea";
import DataError from "components/DataError/DataError";
import DeviceUserError from "components/DeviceUserError";
import Spinner from "components/Spinner/Spinner";
import RevealButton from "components/buttons/RevealButton";

import {
  getInstallDetailsStatusPredicate,
  INSTALL_DETAILS_STATUS_ICONS,
} from "../constants";

interface IGetStatusMessageProps {
  isDUP?: boolean;
  displayStatus: SoftwareInstallStatus | "pending";
  isMDMStatusNotNow: boolean;
  isMDMStatusAcknowledged: boolean;
  appName: string;
  hostDisplayName: string;
  commandUpdatedAt: string;
}

export const getStatusMessage = ({
  isDUP = false,
  displayStatus,
  isMDMStatusNotNow,
  isMDMStatusAcknowledged,
  appName,
  hostDisplayName,
  commandUpdatedAt,
}: IGetStatusMessageProps) => {
  const formattedHost = hostDisplayName ? <b>{hostDisplayName}</b> : "the host";
  const displayTimeStamp = ["failed_install", "installed"].includes(
    displayStatus || ""
  )
    ? ` (${formatDistanceToNow(new Date(commandUpdatedAt), {
        includeSeconds: true,
        addSuffix: true,
      })})`
    : null;

  // Handle NotNow case separately
  if (isMDMStatusNotNow) {
    return (
      <>
        Fleet tried to install <b>{appName}</b>
        {!isDUP && (
          <>
            {" "}
            on {formattedHost} but couldn&apos;t because the host was locked or
            was running on battery power while in Power Nap
            {displayTimeStamp && <> {displayTimeStamp}</>}
          </>
        )}
        . Fleet will try again.
      </>
    );
  }

  // VPP Verify command pending state
  if (displayStatus === "pending_install" && isMDMStatusAcknowledged) {
    return (
      <>
        The MDM command (request) to install <b>{appName}</b>
        {!isDUP && <> on {formattedHost}</>} was acknowledged but the
        installation has not been verified. To re-check, select <b>Refetch</b>
        {!isDUP && " for this host"}.
      </>
    );
  }

  // Verification failed (timeout)
  if (displayStatus === "failed_install" && isMDMStatusAcknowledged) {
    return (
      <>
        The MDM command (request) to install <b>{appName}</b>
        {!isDUP && <> on {formattedHost}</>} was acknowledged but the
        installation has not been verified. Please re-attempt this installation.
      </>
    );
  }

  // Install command failed
  if (displayStatus === "failed_install") {
    return (
      <>
        The MDM command (request) to install <b>{appName}</b>
        {!isDUP && <> on {formattedHost}</>} failed
        {!isDUP && displayTimeStamp && <> {displayTimeStamp}</>}. Please
        re-attempt this installation.
      </>
    );
  }

  const renderSuffix = () => {
    if (isDUP) {
      return null;
    }
    return (
      <>
        {" "}
        on {formattedHost}
        {displayStatus === "pending_install" && " when it comes online"}
        {displayTimeStamp && <> {displayTimeStamp}</>}
      </>
    );
  };
  // Create predicate and subordinate for other statuses
  return (
    <>
      Fleet {getInstallDetailsStatusPredicate(displayStatus)} <b>{appName}</b>
      {renderSuffix()}.
    </>
  );
};

const baseClass = "app-install-details";

export type IVppInstallDetails = {
  fleetInstallStatus: SoftwareInstallStatus;
  hostDisplayName: string;
  appName: string;
  commandUuid: string;
};

interface IVPPInstallDetailsModalProps {
  details: IVppInstallDetails;
  hostSoftware?: IHostSoftware; // for inventory versions, not present on activity feeds
  deviceAuthToken?: string; // DUP only
  onCancel: () => void;
  onRetry?: (id: number) => void; // DUP only
}
export const AppInstallDetailsModal = ({
  details,
  onCancel,
  deviceAuthToken,
  hostSoftware,
  onRetry,
}: IVPPInstallDetailsModalProps) => {
  const {
    fleetInstallStatus,
    commandUuid = "",
    hostDisplayName = "",
    appName = "",
  } = details;

  const [showInstallDetails, setShowInstallDetails] = useState(false);
  const toggleInstallDetails = () => {
    setShowInstallDetails((prev) => !prev);
  };

  const responseHandler = (
    response:
      | IGetVppInstallCommandResultsResponse
      | IGetMdmCommandResultsResponse
  ) => {
    const results = response.results?.[0];
    if (!results) {
      // FIXME: It's currently possible that the command results API response is empty for pending
      // commands. As a temporary workaround to handle this case, we'll ignore the empty response and
      // display some minimal pending UI. This should be removed once the API response is fixed.
      return {} as IMdmCommandResult;
    }
    return {
      ...results,
      payload: atob(results.payload),
      result: atob(results.result),
    };
  };

  const onClickRetry = () => {
    // on DUP, where this is relevant, both will be defined
    if (onRetry && hostSoftware?.id) {
      onRetry(hostSoftware.id);
    }
    onCancel();
  };
  const {
    data: vppCommandResult,
    isLoading: isLoadingVPPCommandResult,
    isError: isErrorVPPCommandResult,
    error: errorVPPCommandResult,
  } = useQuery<IMdmCommandResult, AxiosError>(
    ["mdm_command_results", commandUuid],
    async () => {
      return deviceAuthToken
        ? deviceUserAPI
            .getVppCommandResult(deviceAuthToken, commandUuid)
            .then(responseHandler)
        : mdmApi.getCommandResults(commandUuid).then(responseHandler);
    },
    {
      refetchOnWindowFocus: false,
      staleTime: 3000,
    }
  );

  if (isLoadingVPPCommandResult) {
    return <Spinner />;
  }

  if (isErrorVPPCommandResult) {
    if (errorVPPCommandResult?.status === 404) {
      return deviceAuthToken ? (
        <DeviceUserError />
      ) : (
        <DataError
          description="Install details are no longer available for this activity."
          excludeIssueLink
        />
      );
    }

    if (errorVPPCommandResult?.status === 401) {
      return deviceAuthToken ? (
        <DeviceUserError />
      ) : (
        <DataError description="Close this modal and try again." />
      );
    }
  } else if (!vppCommandResult) {
    // FIXME: It's currently possible that the command results API response is empty for pending
    // commands. As a temporary workaround to handle this case, we'll ignore the empty response and
    // display some minimal pending UI. This should be updated once the API response is fixed.
  }

  const displayStatus =
    (fleetInstallStatus as SoftwareInstallStatus) || "pending_install";
  const iconName = INSTALL_DETAILS_STATUS_ICONS[displayStatus];

  // Note: We need to reconcile status values from two different sources. From props, we
  // get the status of the Fleet install operation (which can be "failed", "pending", or
  // "installed"). From the command results API response, we also receive the raw status
  // from the MDM protocol, e.g., "NotNow" or "Acknowledged". We need to display some special
  // messaging for the "NotNow" status, which otherwise would be treated as "pending".
  const isMDMStatusNotNow = vppCommandResult?.status === "NotNow";
  const isMDMStatusAcknowledged = vppCommandResult?.status === "Acknowledged";

  const excludeVersions =
    !deviceAuthToken &&
    ["pending_install", "failed_install"].includes(fleetInstallStatus);

  const isInstalledByFleet = hostSoftware
    ? !!hostSoftware.app_store_app?.last_install
    : true; // if no hostSoftware passed in, can assume this is the activity feed, meaning this can only refer to a Fleet-handled install

  const statusMessage = getStatusMessage({
    isDUP: !!deviceAuthToken,
    displayStatus,
    isMDMStatusNotNow,
    isMDMStatusAcknowledged,
    appName,
    hostDisplayName,
    commandUpdatedAt: vppCommandResult?.updated_at || "",
  });

  const renderInventoryVersionsSection = () => {
    if (hostSoftware?.installed_versions?.length) {
      return <InventoryVersions hostSoftware={hostSoftware} />;
    }
    return "If you uninstalled it outside of Fleet it will still show as installed.";
  };

  const renderInstallDetailsSection = () => {
    return (
      <>
        <RevealButton
          isShowing={showInstallDetails}
          showText="Details"
          hideText="Details"
          caretPosition="after"
          onClick={toggleInstallDetails}
        />
        {showInstallDetails && (
          <>
            {vppCommandResult?.result && (
              <Textarea label="MDM command output:" variant="code">
                {vppCommandResult.result}
              </Textarea>
            )}
            {vppCommandResult?.payload && (
              <Textarea label="MDM command:" variant="code">
                {vppCommandResult.payload}
              </Textarea>
            )}
          </>
        )}
      </>
    );
  };

  const renderCta = () => {
    if (deviceAuthToken && fleetInstallStatus === "failed_install") {
      return (
        <div className="modal-cta-wrap">
          <Button type="submit" onClick={onClickRetry}>
            Retry
          </Button>
          <Button variant="inverse" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      );
    }
    return (
      <div className="modal-cta-wrap">
        <Button onClick={onCancel}>Done</Button>
      </div>
    );
  };

  return (
    <Modal
      title="Install details"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      <>
        <div className={`${baseClass}__modal-content`}>
          <div className={`${baseClass}__status-message`}>
            {!!iconName && <Icon name={iconName} />}
            <span>{statusMessage}</span>
          </div>
          {hostSoftware && !excludeVersions && renderInventoryVersionsSection()}
          {fleetInstallStatus !== "pending_install" &&
            isInstalledByFleet &&
            renderInstallDetailsSection()}
        </div>
        {renderCta()}
      </>
    </Modal>
  );
};

export default AppInstallDetailsModal;
