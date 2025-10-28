/** Similar look and feel to the VppInstallDetailsModal, but this modal
 * is rendered instead of the SoftwareInstallDetailsModal when the package is
 * an .ipa for iOS/iPadOS */

import React, { useState } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import { formatDistanceToNow } from "date-fns";

import mdmApi, { IGetMdmCommandResultsResponse } from "services/entities/mdm";
import deviceUserAPI, {
  IGetVppInstallCommandResultsResponse,
} from "services/entities/device_user";

import {
  IHostSoftware,
  SoftwareInstallUninstallStatus,
} from "interfaces/software";
import { IMdmCommandResult } from "interfaces/mdm";

import InventoryVersions from "pages/hosts/details/components/InventoryVersions";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
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
  isMyDevicePage?: boolean;
  displayStatus: SoftwareInstallUninstallStatus;
  isMDMStatusNotNow: boolean;
  isMDMStatusAcknowledged: boolean;
  appName: string;
  hostDisplayName: string;
  commandUpdatedAt: string;
}

export const getStatusMessage = ({
  isMyDevicePage = false,
  displayStatus,
  isMDMStatusNotNow,
  isMDMStatusAcknowledged,
  appName,
  hostDisplayName,
  commandUpdatedAt,
}: IGetStatusMessageProps) => {
  const formattedHost = hostDisplayName ? <b>{hostDisplayName}</b> : "the host";
  const displayTimeStamp =
    ["failed_install", "installed"].includes(displayStatus || "") &&
    commandUpdatedAt
      ? ` (${formatDistanceToNow(new Date(commandUpdatedAt), {
          includeSeconds: true,
          addSuffix: true,
        })})`
      : null;

  const isPendingInstall = displayStatus === "pending_install";

  // Handles the case where software is installed manually by the user and not through Fleet
  // This IPA software_packages modal matches app_store_app modal and software_packages modal
  // for software installed manually shown with VppInstallDetailsModal and SoftwareInstallDetailsModal
  if (displayStatus === "installed" && !commandUpdatedAt) {
    return (
      <>
        <b>{appName}</b> is installed.
      </>
    );
  }

  // Handle NotNow case separately
  if (isMDMStatusNotNow) {
    return (
      <>
        Fleet tried to install <b>{appName}</b>
        {!isMyDevicePage && (
          <>
            {" "}
            on {formattedHost} but couldn&apos;t because the host was locked or
            was running on battery power while in Power Nap
          </>
        )}
        {displayTimeStamp && <> {displayTimeStamp}</>}. Fleet will try again.
      </>
    );
  }

  // IPA Verify command pending state
  if (isPendingInstall && isMDMStatusAcknowledged) {
    return (
      <>
        The MDM command (request) to install <b>{appName}</b>
        {!isMyDevicePage && <> on {formattedHost}</>} was acknowledged but the
        installation has not been verified. To re-check, select <b>Refetch</b>
        {!isMyDevicePage && " for this host"}.
      </>
    );
  }

  // Verification failed (timeout)
  if (displayStatus === "failed_install" && isMDMStatusAcknowledged) {
    return (
      <>
        The MDM command (request) to install <b>{appName}</b>
        {!isMyDevicePage && <> on {formattedHost}</>} was acknowledged but the
        installation has not been verified. Please re-attempt this installation.
      </>
    );
  }

  // Install command failed
  if (displayStatus === "failed_install") {
    return (
      <>
        The MDM command (request) to install <b>{appName}</b>
        {!isMyDevicePage && <> on {formattedHost}</>} failed
        {displayTimeStamp && <> {displayTimeStamp}</>}. Please re-attempt this
        installation.
      </>
    );
  }

  const renderSuffix = () => {
    if (isMyDevicePage) {
      return <> {displayTimeStamp && <> {displayTimeStamp}</>}</>;
    }
    return (
      <>
        {" "}
        on {formattedHost}
        {isPendingInstall && " when it comes online"}
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

interface IModalButtonsProps {
  displayStatus: SoftwareInstallUninstallStatus | "pending";
  deviceAuthToken?: string;
  onCancel: () => void;
  onRetry?: (id: number) => void;
  hostSoftwareId?: number;
}

export const ModalButtons = ({
  displayStatus,
  deviceAuthToken,
  onCancel,
  onRetry,
  hostSoftwareId,
}: IModalButtonsProps) => {
  const onClickRetry = () => {
    // on My Device Page, where this is relevant, both will be defined
    if (onRetry && hostSoftwareId) {
      onRetry(hostSoftwareId);
    }
    onCancel();
  };

  if (deviceAuthToken && displayStatus === "failed_install") {
    return (
      <ModalFooter
        primaryButtons={
          <>
            <Button variant="inverse" onClick={onCancel}>
              Cancel
            </Button>
            <Button type="submit" onClick={onClickRetry}>
              Retry
            </Button>
          </>
        }
      />
    );
  }
  return (
    <ModalFooter primaryButtons={<Button onClick={onCancel}>Done</Button>} />
  );
};

const baseClass = "software-ipa-install-details-modal";

export type ISoftwareIpaInstallDetails = {
  /** Status: null when a host manually installed not using Fleet */
  fleetInstallStatus: SoftwareInstallUninstallStatus | null;
  hostDisplayName: string;
  appName: string;
  commandUuid?: string;
};

interface ISoftwareIpaInstallDetailsModal {
  details: ISoftwareIpaInstallDetails;
  /** for inventory versions, not present on activity feeds */
  hostSoftware?: IHostSoftware;
  /** My Device Page only */
  deviceAuthToken?: string;
  onCancel: () => void;
  /** My Device Page only */
  onRetry?: (id: number) => void;
}
export const SoftwareIpaInstallDetailsModal = ({
  details,
  onCancel,
  deviceAuthToken,
  hostSoftware,
  onRetry,
}: ISoftwareIpaInstallDetailsModal) => {
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

  const { data: swInstallResult, isLoading, isError, error } = useQuery<
    IMdmCommandResult,
    AxiosError
  >(
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
      enabled: !!commandUuid,
    }
  );

  console.log("\n\n\n\nswInstallResult", swInstallResult);

  // Fallback to "installed" if no status is provided
  const displayStatus = fleetInstallStatus ?? "installed";
  const iconName = INSTALL_DETAILS_STATUS_ICONS[displayStatus];

  // Handles "pending" value prior to 4.57 AND never shows error state on pending_install
  // as some cases have command results not available for pending_installs
  // which we don't want to show a UI error state for
  const isPendingInstall = ["pending_install", "pending"].includes(
    displayStatus
  );

  // Note: We need to reconcile status values from two different sources. From props, we
  // get the status of the Fleet install operation (which can be "failed", "pending", or
  // "installed"). From the command results API response, we also receive the raw status
  // from the MDM protocol, e.g., "NotNow" or "Acknowledged". We need to display some special
  // messaging for the "NotNow" status, which otherwise would be treated as "pending".
  const isMDMStatusNotNow = swInstallResult?.status === "NotNow";
  const isMDMStatusAcknowledged = swInstallResult?.status === "Acknowledged";

  const excludeVersions =
    !deviceAuthToken &&
    ["pending_install", "failed_install", "pending"].includes(displayStatus);

  const isInstalledByFleet = hostSoftware
    ? !!hostSoftware.app_store_app?.last_install
    : true; // if no hostSoftware passed in, can assume this is the activity feed, meaning this can only refer to a Fleet-handled install

  const statusMessage = getStatusMessage({
    isMyDevicePage: !!deviceAuthToken,
    displayStatus,
    isMDMStatusNotNow,
    isMDMStatusAcknowledged,
    appName,
    hostDisplayName,
    commandUpdatedAt: swInstallResult?.updated_at || "",
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
            {swInstallResult?.result && (
              <Textarea label="MDM command output:" variant="code">
                {swInstallResult.result}
              </Textarea>
            )}
            {swInstallResult?.payload && (
              <Textarea label="MDM command:" variant="code">
                {swInstallResult.payload}
              </Textarea>
            )}
          </>
        )}
      </>
    );
  };

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (isError && !isPendingInstall) {
      if (error?.status === 404) {
        return deviceAuthToken ? (
          <DeviceUserError />
        ) : (
          <DataError
            description="Install details are no longer available for this activity."
            excludeIssueLink
          />
        );
      }

      if (error?.status === 401) {
        return deviceAuthToken ? (
          <DeviceUserError />
        ) : (
          <DataError description="Close this modal and try again." />
        );
      }
    } else if (!swInstallResult) {
      // FIXME: It's currently possible that the command results API response is empty for pending
      // commands. As a temporary workaround to handle this case, we'll ignore the empty response and
      // display some minimal pending UI. This should be updated once the API response is fixed.
    }
    return (
      <div className={`${baseClass}__modal-content`}>
        <div className={`${baseClass}__status-message`}>
          {!!iconName && <Icon name={iconName} />}
          <span>{statusMessage}</span>
        </div>
        {hostSoftware && !excludeVersions && renderInventoryVersionsSection()}
        {!isPendingInstall &&
          isInstalledByFleet &&
          renderInstallDetailsSection()}
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
        {renderContent()}
        <ModalButtons
          deviceAuthToken={deviceAuthToken}
          hostSoftwareId={hostSoftware?.id}
          onRetry={onRetry}
          onCancel={onCancel}
          displayStatus={displayStatus}
        />
      </>
    </Modal>
  );
};

export default SoftwareIpaInstallDetailsModal;
