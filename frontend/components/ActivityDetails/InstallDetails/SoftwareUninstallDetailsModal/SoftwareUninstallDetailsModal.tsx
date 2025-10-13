import React, { useState } from "react";
import { AxiosError } from "axios";
import { useQuery } from "react-query";
import { formatDistanceToNow } from "date-fns";

import deviceUserAPI from "services/entities/device_user";
import scriptsAPI, { IScriptResultResponse } from "services/entities/scripts";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import {
  IHostSoftwareWithUiStatus,
  isPendingStatus,
  SoftwareUninstallStatus,
} from "interfaces/software";

import Button from "components/buttons/Button";
import DataError from "components/DataError";
import Icon from "components/Icon";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Spinner from "components/Spinner";
import Textarea from "components/Textarea";
import RevealButton from "components/buttons/RevealButton";
import {
  getInstallDetailsStatusPredicate,
  INSTALL_DETAILS_STATUS_ICONS,
} from "../constants";
import { renderContactOption } from "../SoftwareInstallDetailsModal/SoftwareInstallDetailsModal";

const baseClass = "software-uninstall-details-modal";

interface IUninstallStatusMessage {
  hostDisplayName: string;
  status: SoftwareUninstallStatus;
  softwareName: string;
  softwarePackageName?: string;
  timestamp?: string;
  isMyDevicePage: boolean;
  contactUrl?: string;
}

export const StatusMessage = ({
  hostDisplayName,
  status,
  softwareName,
  softwarePackageName,
  timestamp,
  isMyDevicePage,
  contactUrl,
}: IUninstallStatusMessage) => {
  const formattedHost = hostDisplayName ? <b>{hostDisplayName}</b> : "the host";

  const isPending = isPendingStatus(status);
  const displayTimeStamp =
    !isPending && timestamp
      ? ` (${formatDistanceToNow(new Date(timestamp), {
          includeSeconds: true,
          addSuffix: true,
        })})`
      : "";

  const renderStatusCopy = () => {
    const prefix = (
      <>
        Fleet {getInstallDetailsStatusPredicate(status)} <b>{softwareName}</b>
      </>
    );
    let suffix = null;
    if (isMyDevicePage) {
      if (status === "failed_uninstall") {
        suffix = <>. You can retry{renderContactOption(contactUrl)}</>;
      }
    } else {
      // host details page
      suffix = (
        <>
          {softwarePackageName && <> ({softwarePackageName})</>} from{" "}
          {formattedHost}
          {status === "pending_uninstall" ? " when it comes online" : ""}
          {displayTimeStamp}.
        </>
      );
    }
    return (
      <span>
        {prefix}
        {suffix}
      </span>
    );
  };
  return (
    <div className={`${baseClass}__status-message`}>
      <Icon name={INSTALL_DETAILS_STATUS_ICONS[status] ?? "pending-outline"} />
      {renderStatusCopy()}
    </div>
  );
};

interface IModalButtonsProps {
  uninstallStatus: SoftwareUninstallStatus;
  deviceAuthToken?: string;
  onCancel: () => void;
  onRetry?: (s: IHostSoftwareWithUiStatus) => void;
  hostSoftware?: IHostSoftwareWithUiStatus;
}

export const ModalButtons = ({
  uninstallStatus,
  deviceAuthToken,
  onCancel,
  onRetry,
  hostSoftware,
}: IModalButtonsProps) => {
  const onClickRetry = () => {
    if (onRetry && hostSoftware) {
      onRetry(hostSoftware);
    }
    onCancel();
  };

  if (deviceAuthToken && uninstallStatus === "failed_uninstall") {
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

export interface ISWUninstallDetailsParentState {
  softwareName: string;
  uninstallStatus: SoftwareUninstallStatus;
  scriptExecutionId: string;
  softwarePackageName?: string;
  /** Optional since may come from dedicated state, may come from elsewhere */
  hostDisplayName?: string;

  /** Optional since DUP only */
  hostSoftware?: IHostSoftwareWithUiStatus; // UI status not necessary in this modal, but type aligns with onRetry argument
}
export interface ISoftwareUninstallDetailsModalProps {
  hostDisplayName: string;
  softwareName: string;
  uninstallStatus: SoftwareUninstallStatus;
  scriptExecutionId: string;
  onCancel: () => void;
  softwarePackageName?: string;

  /** DUP only */
  onRetry?: (s: IHostSoftwareWithUiStatus) => void;
  hostSoftware?: IHostSoftwareWithUiStatus; // UI status not necessary in this modal, but type aligns with onRetry argument
  deviceAuthToken?: string;
  contactUrl?: string;
}
const SoftwareUninstallDetailsModal = ({
  hostDisplayName,
  softwareName,
  softwarePackageName,
  uninstallStatus,
  scriptExecutionId,
  onCancel,

  onRetry,
  hostSoftware,
  deviceAuthToken,
  contactUrl,
}: ISoftwareUninstallDetailsModalProps) => {
  const [showDetails, setShowDetails] = useState(false);

  const toggleDetails = () => setShowDetails((prev) => !prev);

  const { data: uninstallResult, isLoading, isError, error } = useQuery<
    IScriptResultResponse,
    AxiosError
  >(
    ["uninstallResult", scriptExecutionId],
    () => {
      return deviceAuthToken
        ? deviceUserAPI.getSoftwareUninstallResult(
            deviceAuthToken,
            scriptExecutionId
          )
        : scriptsAPI.getScriptResult(scriptExecutionId);
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      // are 4xx errors other than 404 expected intermittently?
      retry: (failureCount, err) => err?.status !== 404 && failureCount < 3,
      // Prevent any error UI with pending uninstall
      enabled: uninstallStatus !== "pending_uninstall",
    }
  );

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    } else if (isError && error?.status === 404) {
      return (
        <DataError
          description="These uninstall details are no longer available."
          excludeIssueLink
        />
      );
    } else if (isError) {
      return <DataError description="Close this modal and try again." />;
    } else if (!uninstallResult && uninstallStatus !== "pending_uninstall") {
      // FIXME: Find a better solution for this.
      return <DataError description="No data returned." />;
    }

    return (
      <div className={`${baseClass}__modal-content`}>
        <StatusMessage
          hostDisplayName={hostDisplayName || ""}
          status={
            (uninstallStatus || "pending_uninstall") as SoftwareUninstallStatus
          }
          softwareName={softwareName}
          softwarePackageName={softwarePackageName}
          timestamp={uninstallResult?.created_at}
          isMyDevicePage={!!deviceAuthToken}
          contactUrl={contactUrl}
        />
        {uninstallStatus !== "pending_uninstall" && (
          <RevealButton
            isShowing={showDetails}
            showText="Details"
            hideText="Details"
            caretPosition="after"
            onClick={toggleDetails}
          />
        )}
        {showDetails && uninstallResult?.script_contents && (
          <Textarea label="Uninstall script content:" variant="code">
            {uninstallResult.script_contents}
          </Textarea>
        )}
        {showDetails && uninstallResult?.output && (
          <Textarea label="Uninstall script output:" variant="code">
            {uninstallResult.output}
          </Textarea>
        )}
      </div>
    );
  };

  return (
    <Modal
      title="Uninstall details"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      <>
        {renderContent()}
        <ModalButtons
          uninstallStatus={uninstallStatus}
          deviceAuthToken={deviceAuthToken}
          onCancel={onCancel}
          onRetry={onRetry}
          hostSoftware={hostSoftware}
        />
      </>
    </Modal>
  );
};

export default SoftwareUninstallDetailsModal;
