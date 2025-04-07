import Button from "components/buttons/Button";
import DataError from "components/DataError";
import Icon from "components/Icon";
import Modal from "components/Modal";
import Spinner from "components/Spinner";
import Textarea from "components/Textarea";
import { formatDistanceToNow } from "date-fns";
import { IActivityDetails } from "interfaces/activity";
import { isPendingStatus, SoftwareInstallStatus } from "interfaces/software";
import React from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import scriptsAPI, { IScriptResultResponse } from "services/entities/scripts";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import {
  getInstallDetailsStatusPredicate,
  INSTALL_DETAILS_STATUS_ICONS,
} from "../constants";

const baseClass = "software-uninstall-details-modal";

type ISoftwareUninstallDetails = Pick<
  IActivityDetails,
  "script_execution_id" | "host_display_name" | "software_title" | "status"
>;
// TODO - rely on activity created_at for timestamp? what else?

interface IUninstallStatusMessage {
  host_display_name: string;
  // TODO - improve status typing
  status: string;
  software_title: string;
  timestamp: string;
}

const StatusMessage = ({
  host_display_name,
  status,
  software_title,
  timestamp,
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
  return (
    <div className={`${baseClass}__status-message`}>
      <Icon
        name={
          INSTALL_DETAILS_STATUS_ICONS[status as SoftwareInstallStatus] ??
          "pending-outline"
        }
      />
      <span>
        Fleet {getInstallDetailsStatusPredicate(status)} <b>{software_title}</b>{" "}
        from {formattedHost}
        {isPending ? " when it comes online" : ""}
        {displayTimeStamp}.
      </span>
    </div>
  );
};

const SoftwareUninstallDetailsModal = ({
  details,
  onCancel,
}: {
  details: ISoftwareUninstallDetails;
  onCancel: () => void;
}) => {
  const SoftwareUninstallDetails = ({
    script_execution_id = "",
    host_display_name = "",
    software_title = "",
    status = "",
  }: ISoftwareUninstallDetails) => {
    const { data: scriptResult, isLoading, isError, error } = useQuery<
      IScriptResultResponse,
      AxiosError
    >(
      ["uninstallResult", details.script_execution_id],
      () => {
        return scriptsAPI.getScriptResult(script_execution_id);
      },
      {
        ...DEFAULT_USE_QUERY_OPTIONS,
        retry: (failureCount, err) => err?.status !== 404 && failureCount < 3,
      }
    );

    if (isLoading) {
      return <Spinner />;
    } else if (isError && error?.status === 404) {
      return (
        <DataError
          description="Uninstall details are no longer available for this activity."
          excludeIssueLink
        />
      );
    } else if (isError) {
      return <DataError description="Close this modal and try again." />;
    } else if (!scriptResult) {
      // FIXME: Find a better solution for this.
      return <DataError description="No data returned." />;
    }

    status = status === "failed" ? "failed_uninstall" : status;

    return (
      <>
        <StatusMessage
          host_display_name={host_display_name}
          status={status}
          software_title={software_title}
          timestamp={scriptResult.created_at}
        />
        {!isPendingStatus(status) && scriptResult && (
          <>
            <div className={`${baseClass}__script-output`}>
              Uninstall script content:
              <Textarea className={`${baseClass}__output-textarea`}>
                {scriptResult.script_contents}
              </Textarea>
            </div>

            <div className={`${baseClass}__script-output`}>
              Uninstall script output:
              <Textarea className={`${baseClass}__output-textarea`}>
                {scriptResult.output}
              </Textarea>
            </div>
          </>
        )}
      </>
    );
  };

  return (
    <Modal
      title="Uninstall details"
      onExit={onCancel}
      onEnter={onCancel}
      width="large"
      className={baseClass}
    >
      <>
        <div className={`${baseClass}__modal-content`}>
          <SoftwareUninstallDetails {...details} />
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default SoftwareUninstallDetailsModal;
