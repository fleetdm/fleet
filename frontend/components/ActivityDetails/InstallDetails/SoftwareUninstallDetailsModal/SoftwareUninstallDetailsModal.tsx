/** 
Uninstall data model

data needed from props:
  Always:
  - `softwareName` - from `activity.details.software_title` or host `software.name`
  - `uninstallStatus` - from `hostSoftware` or `activity.details`
  - `onCancel`
  - `scriptExecutionId` - from `activity` or host `software`
  
  Optional:
  - `host_display_name` - provided by either the parent state `activity.details` or the parent `host`

  DUP SS only:
  - `onRetry` - from `selfService`
  - `hostSoftware: IHostSoftwareWithUiStatus` - from parent `InstallStatusCell`
  - `deviceAuthToken`

---details---
- `host_display_name`
  - used for rendering name of host in description. needs to be present in all cases,
  - source from parent `host` or `activity.details.host_display_name`

- `uninstallStatus` - used to determine description copy as well as whether or not to show details
  section. Always needed.
  - source from parent `activity.details.status` OR parent `hostSoftware`

- `onRetry` and local wrapper
  - retries the uninstall operation
  - method sourced from parent, argument to it sourced from parent, which is set
    by`InstallStatusCell` from its config `hostSoftware`
- `hostSoftware`
  - passed in to `onRetry`
  - from parent `software` on self-service page only

- `onCancel`
  - close the modal
  - comes from parent
  - `deviceAuthToken`
  - used to
  - determine which API to GET
  - conditionally include Retry button
  - `scriptExecutionId/script_execution_id`
  - defines with uninstall (script) result to get from the API call
  - comes from parent `activity.details.script_execution_id` or
  `hostSoftware.software_package.last_uninstall.script_execution_id`
  
  from API call:
  - `uninstallResult` - used to show `.output` in details and `.created_at` in description
  - comes from response to API call
  */

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
  host_display_name: string;
  status: SoftwareUninstallStatus;
  softwareName: string;
  softwarePackageName?: string;
  timestamp: string;
  isDUP: boolean;
  contactUrl?: string;
}

const StatusMessage = ({
  host_display_name,
  status,
  softwareName,
  softwarePackageName,
  timestamp,
  isDUP,
  contactUrl,
}: IUninstallStatusMessage) => {
  const formattedHost = host_display_name ? (
    <b>{host_display_name}</b>
  ) : (
    "the host"
  );

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
    if (isDUP) {
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

export interface ISWUninstallDetailsParentState {
  softwareName: string;
  uninstallStatus: SoftwareUninstallStatus; // TODO - type massage for "pending"?
  scriptExecutionId: string;
  softwarePackageName?: string;
  /** Optional since may come from dedicated state, may come from elsewhere */
  hostDisplayName?: string;

  /** Optional since DUP only */
  hostSoftware?: IHostSoftwareWithUiStatus; // UI status not necessary in this modal, but type aligns with onRetry argument
}
export interface ISoftwareUninstallDetailsModalProps {
  hostDisplayName: string; // optional to facilitate use as state type, must always be provided
  softwareName: string;
  uninstallStatus: SoftwareUninstallStatus; // TODO - type massage for "pending"?
  scriptExecutionId: string;
  onCancel: () => void; // optional to facilitate use as state type, must always be provided
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

  const onClickRetry = () => {
    // on DUP, where this is relevant, both will be defined
    if (onRetry && hostSoftware) {
      onRetry(hostSoftware);
    }
    onCancel();
  };

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
    }
  );

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
  } else if (!uninstallResult) {
    // FIXME: Find a better solution for this.
    return <DataError description="No data returned." />;
  }

  const renderDetailsSection = () => (
    <>
      <RevealButton
        isShowing={showDetails}
        showText="Details"
        hideText="Details"
        caretPosition="after"
        onClick={toggleDetails}
      />
      {showDetails && uninstallResult.output && (
        <Textarea label="Uninstall script output:" variant="code">
          {uninstallResult.output}
        </Textarea>
      )}
    </>
  );

  const renderCta = () => {
    if (deviceAuthToken && uninstallStatus === "failed_uninstall") {
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
      title="Uninstall details"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
      // width="large"
    >
      <>
        <div className={`${baseClass}__modal-content`}>
          <StatusMessage
            host_display_name={hostDisplayName || ""}
            status={
              (uninstallStatus ||
                "pending_uninstall") as SoftwareUninstallStatus
            }
            softwareName={softwareName}
            softwarePackageName={softwarePackageName}
            timestamp={uninstallResult.created_at}
            isDUP={!!deviceAuthToken}
            contactUrl={contactUrl}
          />
          {uninstallStatus !== "pending_uninstall" && renderDetailsSection()}
        </div>
        {renderCta()}
      </>
    </Modal>
  );
};

export default SoftwareUninstallDetailsModal;
