import React from "react";
import { useQuery } from "react-query";
import { formatDistanceToNow } from "date-fns";
import { AxiosError } from "axios";

import { IActivityDetails } from "interfaces/activity";
import {
  ISoftwareInstallResult,
  ISoftwareInstallResults,
} from "interfaces/software";
import softwareAPI from "services/entities/software";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import Textarea from "components/Textarea";
import DataError from "components/DataError/DataError";
import Spinner from "components/Spinner/Spinner";
import {
  INSTALL_DETAILS_STATUS_ICONS,
  SOFTWARE_INSTALL_OUTPUT_DISPLAY_LABELS,
  getInstallDetailsStatusPredicate,
} from "../constants";

const baseClass = "software-install-details";

// TODO: Expand to include more details as needed
export type IPackageInstallDetails = Pick<
  IActivityDetails,
  "install_uuid" | "host_display_name"
>;

const StatusMessage = ({
  result: {
    host_display_name,
    software_package,
    software_title,
    status,
    updated_at,
    created_at,
  },
}: {
  result: ISoftwareInstallResult;
}) => {
  const formattedHost = host_display_name ? (
    <b>{host_display_name}</b>
  ) : (
    "the host"
  );

  const timeStamp = updated_at || created_at;
  const displayTimeStamp = ["failed_install", "installed"].includes(
    status || ""
  )
    ? ` (${formatDistanceToNow(new Date(timeStamp), {
        includeSeconds: true,
        addSuffix: true,
      })})`
    : "";
  return (
    <div className={`${baseClass}__status-message`}>
      <Icon name={INSTALL_DETAILS_STATUS_ICONS[status] ?? "pending-outline"} />
      <span>
        Fleet {getInstallDetailsStatusPredicate(status)} <b>{software_title}</b>{" "}
        ({software_package}) on {formattedHost}
        {status === "pending_install" ? " when it comes online" : ""}
        {displayTimeStamp}.
      </span>
    </div>
  );
};

const Output = ({
  displayKey,
  result,
}: {
  displayKey: keyof typeof SOFTWARE_INSTALL_OUTPUT_DISPLAY_LABELS;
  result: ISoftwareInstallResult;
}) => {
  return (
    <div className={`${baseClass}__script-output`}>
      {SOFTWARE_INSTALL_OUTPUT_DISPLAY_LABELS[displayKey]}:
      <Textarea className={`${baseClass}__output-textarea`}>
        {result[displayKey]}
      </Textarea>
    </div>
  );
};

export const SoftwareInstallDetails = ({
  host_display_name = "",
  install_uuid = "",
}: IPackageInstallDetails) => {
  const { data: result, isLoading, isError, error } = useQuery<
    ISoftwareInstallResults,
    AxiosError,
    ISoftwareInstallResult
  >(
    ["softwareInstallResults", install_uuid],
    () => {
      return softwareAPI.getSoftwareInstallResult(install_uuid);
    },
    {
      refetchOnWindowFocus: false,
      staleTime: 3000,
      select: (data) => data.results,
      retry: (failureCount, err) => err?.status !== 404 && failureCount < 3,
    }
  );

  if (isLoading) {
    return <Spinner />;
  } else if (isError && error?.status === 404) {
    return (
      <DataError
        description="Install details are no longer available for this activity."
        excludeIssueLink
      />
    );
  } else if (isError) {
    return <DataError description="Close this modal and try again." />;
  } else if (!result) {
    // FIXME: Find a better solution for this.
    return <DataError description="No data returned." />;
  }

  return (
    <>
      <div className={`${baseClass}__software-install-details`}>
        <StatusMessage
          result={
            result.host_display_name ? result : { ...result, host_display_name } // prefer result.host_display_name (it may be empty if the host was deleted) otherwise default to whatever we received via props
          }
        />
        {result.status !== "pending_install" && (
          <>
            {result.pre_install_query_output && (
              <Output displayKey="pre_install_query_output" result={result} />
            )}
            {result.output && <Output displayKey="output" result={result} />}
            {result.post_install_script_output && (
              <Output displayKey="post_install_script_output" result={result} />
            )}
          </>
        )}
      </div>
    </>
  );
};

export const SoftwareInstallDetailsModal = ({
  details,
  onCancel,
}: {
  details: IPackageInstallDetails;
  onCancel: () => void;
}) => {
  return (
    <Modal
      title="Install details"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      <>
        <div className={`${baseClass}__modal-content`}>
          <SoftwareInstallDetails {...details} />
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
