/** Modal for VPP app installs only.
 * For iOS/iPadOS .ipa packages (software source: ios_apps or ipados_apps),
 * use SoftwareIpaInstallDetailsModal with the command_uuid instead. */

import React, { useState } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import { formatDistanceToNow } from "date-fns";

import commandAPI, {
  IGetCommandResultsResponse,
} from "services/entities/command";
import deviceUserAPI, {
  IGetVppInstallCommandResultsResponse,
} from "services/entities/device_user";

import {
  IHostSoftware,
  SoftwareInstallUninstallStatus,
} from "interfaces/software";
import { ICommandResult } from "interfaces/command";
import { isAppleDevice, isMacOS } from "interfaces/platform";
import { secondsToDhms } from "utilities/helpers";

import InventoryVersions from "pages/hosts/details/components/InventoryVersions";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Button from "components/buttons/Button";
import IconStatusMessage from "components/IconStatusMessage";
import Textarea from "components/Textarea";
import DataError from "components/DataError/DataError";
import DeviceUserError from "components/DeviceUserError";
import Spinner from "components/Spinner/Spinner";
import TooltipWrapper from "components/TooltipWrapper";
import RevealButton from "components/buttons/RevealButton";

import {
  getInstallDetailsStatusPredicate,
  INSTALL_DETAILS_STATUS_ICONS,
} from "../constants";
import decodeBase64Utf8 from "../helpers";

interface IGetStatusMessageProps {
  isMyDevicePage?: boolean;
  /** "pending" is an edge case here where VPP install activities that were added to the feed prior to v4.57
   * (when we split pending into pending_install/pending_uninstall) will list the status as "pending" rather than "pending_install" */
  displayStatus: SoftwareInstallUninstallStatus | "pending";
  isMDMStatusNotNow: boolean;
  isMDMStatusAcknowledged: boolean;
  appName: string;
  hostDisplayName: string;
  commandUpdatedAt: string;
  platform?: string;
  vppVerifyTimeoutSeconds?: number;
  /**  Used only for overriding failed_install/failed_uninstall -> "is installed."
   - From Host -> Software: override based on inventory.
   - From Activity feed: never override (always show the failure).
   Parity with SoftwareInstallDetailsModal/SoftwareIpaInstallDetailsModal */
  canOverrideFailureWithInstalled?: boolean;
  /** Used to show warning to close an app if failed to install with
   * detected installed version on host */
  hasInstalledVersionsOnHost?: boolean;
}

export const getStatusMessage = ({
  isMyDevicePage = false,
  displayStatus,
  isMDMStatusNotNow,
  isMDMStatusAcknowledged,
  appName,
  hostDisplayName,
  commandUpdatedAt,
  platform,
  vppVerifyTimeoutSeconds,
  canOverrideFailureWithInstalled = false,
  hasInstalledVersionsOnHost = false,
}: IGetStatusMessageProps) => {
  const formattedHost = hostDisplayName ? <b>{hostDisplayName}</b> : "the host";
  const formattedVerifyTimeout = secondsToDhms(vppVerifyTimeoutSeconds || 600);
  const displayTimestamp =
    ["failed_install", "installed"].includes(displayStatus || "") &&
    commandUpdatedAt
      ? ` (${formatDistanceToNow(new Date(commandUpdatedAt), {
          includeSeconds: true,
          addSuffix: true,
        })})`
      : null;

  // Handles "pending" value prior to 4.57
  const isPendingInstall = ["pending_install", "pending"].includes(
    displayStatus
  );

  // Treat failed_install / failed_uninstall with installed versions as installed
  // as the host still reports installed versions (4.82 #31663)
  const overrideFailureWithInstalled =
    canOverrideFailureWithInstalled &&
    ["failed_install", "failed_uninstall"].includes(displayStatus || "");

  if (overrideFailureWithInstalled) {
    return (
      <>
        <b>{appName}</b> is installed.
      </>
    );
  }

  // Handles the case where software is installed manually by the user and not through Fleet
  // This app_store_app modal matches software_packages installed manually shown with SoftwareInstallDetailsModal
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
        {displayTimestamp && <> {displayTimestamp}</>}. Fleet will try again.
      </>
    );
  }

  // VPP Verify command pending state
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
        {isAppleDevice(platform) ? (
          <>
            <div>
              The host acknowledged the MDM command to install <b>{appName}</b>
              {!isMyDevicePage && <> on {formattedHost}</>}, but the install
              took longer than {formattedVerifyTimeout}, so Fleet marked it as
              failed.
            </div>
            {platform && isMacOS(platform) && hasInstalledVersionsOnHost && (
              <div className="vpp-install-details-modal__update-tip">
                If you&apos;re updating the app and the app is open,{" "}
                <TooltipWrapper
                  tipContent="For updates, App Store (VPP) apps on macOS need to be closed."
                  position="top"
                >
                  close it
                </TooltipWrapper>{" "}
                and try again.
              </div>
            )}
          </>
        ) : (
          <>
            The MDM command (request) to install <b>{appName}</b>
            {!isMyDevicePage && <> on {formattedHost}</>} was acknowledged but
            the installation has not been verified. Please re-attempt this
            installation.
          </>
        )}
      </>
    );
  }

  // Install command failed
  if (displayStatus === "failed_install") {
    return (
      <>
        {isAppleDevice(platform) ? (
          <>
            The MDM command to install <b>{appName}</b>
            {!isMyDevicePage && <> on {formattedHost}</>} failed. Please try
            again.
          </>
        ) : (
          <>
            The MDM command (request) to install <b>{appName}</b>
            {!isMyDevicePage && <> on {formattedHost}</>} failed
            {displayTimestamp && <> {displayTimestamp}</>}. Please re-attempt
            this installation.
          </>
        )}
      </>
    );
  }

  const renderSuffix = () => {
    if (isMyDevicePage) {
      return <> {displayTimestamp && <> {displayTimestamp}</>}</>;
    }
    return (
      <>
        {" "}
        on {formattedHost}
        {isPendingInstall && " when it comes online"}
        {displayTimestamp && <> {displayTimestamp}</>}
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
    <ModalFooter primaryButtons={<Button onClick={onCancel}>Close</Button>} />
  );
};

const baseClass = "vpp-install-details-modal";

export type IVppInstallDetails = {
  /** Status: null when a host manually installed not using Fleet */
  fleetInstallStatus: SoftwareInstallUninstallStatus | null;
  hostDisplayName: string;
  appName: string;
  commandUuid?: string;
  platform?: string;
};

interface IVPPInstallDetailsModalProps {
  details: IVppInstallDetails;
  /** for inventory versions, not present on activity feeds */
  hostSoftware?: IHostSoftware;
  /** My Device Page only */
  deviceAuthToken?: string;
  onCancel: () => void;
  /** My Device Page only */
  onRetry?: (id: number) => void;
}
export const VppInstallDetailsModal = ({
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
    platform: detailsPlatform,
  } = details;

  const [showInstallDetails, setShowInstallDetails] = useState(false);
  const toggleInstallDetails = () => {
    setShowInstallDetails((prev) => !prev);
  };

  const responseHandler = (
    response: IGetVppInstallCommandResultsResponse | IGetCommandResultsResponse
  ) => {
    const results = response.results?.[0];
    if (!results) {
      // FIXME: It's currently possible that the command results API response is empty for pending
      // commands. As a temporary workaround to handle this case, we'll ignore the empty response and
      // display some minimal pending UI. This should be removed once the API response is fixed.
      return {} as ICommandResult;
    }
    return {
      ...results,
      payload: results.payload ? decodeBase64Utf8(results.payload) : "",
      result: results.result ? decodeBase64Utf8(results.result) : "",
    };
  };

  const {
    data: vppCommandResult,
    isLoading: isLoadingVPPCommandResult,
    isError: isErrorVPPCommandResult,
    error: errorVPPCommandResult,
  } = useQuery<ICommandResult, AxiosError>(
    ["mdm_command_results", commandUuid],
    async () => {
      return deviceAuthToken
        ? deviceUserAPI
            .getVppCommandResult(deviceAuthToken, commandUuid)
            .then(responseHandler)
        : commandAPI.getCommandResults(commandUuid).then(responseHandler);
    },
    {
      refetchOnWindowFocus: false,
      staleTime: 3000,
      enabled: !!commandUuid,
    }
  );

  // Reconcile "installed" state from inventory vs command results.

  // True when host inventory reports at least one installed version for this app.
  const inventoryReportsInstalled = !!hostSoftware?.installed_versions?.length;

  // True when the VPP command result metadata says the app is installed on the host.
  const commandReportsInstalled =
    (vppCommandResult?.results_metadata?.software_installed as boolean) ??
    false;

  // This modal is opened in two contexts:
  // - From Host -> Software: hostSoftware is defined (we trust inventory to override failures).
  // - From the Activity feed: hostSoftware is undefined (we trust command result status).
  const openedFromHostSoftwarePage = !!hostSoftware;

  // Used only for overriding failed_install/failed_uninstall -> "is installed."
  // - From Host -> Software: override based on inventory.
  // - From Activity feed: never override (always show the failure).
  const canOverrideFailureWithInstalled = openedFromHostSoftwarePage
    ? inventoryReportsInstalled
    : false;

  // Used to
  const hasInstalledVersionsOnHost =
    commandReportsInstalled || inventoryReportsInstalled;

  // Fallback to "installed" if no status is provided
  const displayStatus = fleetInstallStatus ?? "installed";

  // Treat failed_install / failed_uninstall with installed versions as installed
  const overrideFailedMessageWithInstalledMessage =
    canOverrideFailureWithInstalled &&
    ["failed_install", "failed_uninstall"].includes(displayStatus || "");

  const commandUpdatedAt = vppCommandResult?.updated_at;

  // Handles the case where software is installed manually by the user and not through Fleet
  const isManuallyInstalled =
    displayStatus === "installed" && !commandUpdatedAt; // using same condition as in getStatusMessage

  // Use success icon when we show “is installed”
  const iconName =
    overrideFailedMessageWithInstalledMessage || isManuallyInstalled
      ? "success"
      : INSTALL_DETAILS_STATUS_ICONS[displayStatus];

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
  const isMDMStatusNotNow = vppCommandResult?.status === "NotNow";
  const isMDMStatusAcknowledged = vppCommandResult?.status === "Acknowledged";
  const platform = hostSoftware?.app_store_app?.platform || detailsPlatform;
  const vppVerifyTimeoutSeconds = Number(
    vppCommandResult?.results_metadata?.vpp_verify_timeout_seconds
  );
  const isVerificationTimedOut =
    displayStatus === "failed_install" &&
    isMDMStatusAcknowledged &&
    isAppleDevice(platform);

  // Hide version section from pending installs or failures that aren't overridden to installed (4.82 #31663)
  const shouldShowInventoryVersions =
    (!!hostSoftware &&
      deviceAuthToken &&
      ![
        "pending_install",
        "failed_install",
        "failed_uninstall",
        "pending",
      ].includes(displayStatus)) ||
    overrideFailedMessageWithInstalledMessage;

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
    commandUpdatedAt: vppCommandResult?.updated_at || "",
    platform,
    vppVerifyTimeoutSeconds: Number.isFinite(vppVerifyTimeoutSeconds)
      ? vppVerifyTimeoutSeconds
      : undefined,
    canOverrideFailureWithInstalled,
    hasInstalledVersionsOnHost,
  });

  const renderInstallDetailsSection = () => {
    // Hide section if there's no details to display
    if (!vppCommandResult?.result && !vppCommandResult?.payload) {
      return null;
    }

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

  // Hide failed details if host shows installed versions (4.82 #31663)
  // NOTE: Currently no uninstall VPP but added for symmetry with SoftwareInstallDetailsModal
  const excludeInstallDetails =
    canOverrideFailureWithInstalled &&
    [
      "failed_install_installed",
      "failed_uninstall_installed",
      "failed_install",
      "failed_uninstall",
    ].includes(displayStatus || "");

  const renderContent = () => {
    if (isLoadingVPPCommandResult) {
      return <Spinner />;
    }

    if (isErrorVPPCommandResult && !isPendingInstall) {
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
    return (
      <div className={`${baseClass}__modal-content`}>
        <IconStatusMessage
          className={`${baseClass}__status-message`}
          iconName={iconName}
          message={<span>{statusMessage}</span>}
        />
        {isVerificationTimedOut && (
          <p>
            If the install finishes later, Fleet will update the status when the
            host is refetched.
          </p>
        )}
        {shouldShowInventoryVersions &&
        hostSoftware?.installed_versions?.length ? (
          <InventoryVersions hostSoftware={hostSoftware} />
        ) : null}
        {!isPendingInstall &&
          isInstalledByFleet &&
          !excludeInstallDetails &&
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
      {renderContent()}
      <ModalButtons
        deviceAuthToken={deviceAuthToken}
        hostSoftwareId={hostSoftware?.id}
        onRetry={onRetry}
        onCancel={onCancel}
        displayStatus={displayStatus}
      />
    </Modal>
  );
};

export default VppInstallDetailsModal;
