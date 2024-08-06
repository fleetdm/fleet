import React from "react";
import { useQuery } from "react-query";

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

const StatusMessage = ({
  result: { host_display_name, software_package, software_title, status },
}: {
  result: ISoftwareInstallResult;
}) => {
  return (
    <div className={`${baseClass}__status-message`}>
      <Icon name={INSTALL_DETAILS_STATUS_ICONS[status]} />
      <span>
        Fleet {getInstallDetailsStatusPredicate(status)} <b>{software_title}</b>{" "}
        ({software_package}) on <b>{host_display_name}</b>
        {status === "pending" ? " when it comes online" : ""}.
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
  installUuid,
}: {
  installUuid: string;
}) => {
  const { data: result, isLoading, isError } = useQuery<
    ISoftwareInstallResults,
    Error,
    ISoftwareInstallResult
  >(
    ["softwareInstallResults", installUuid],
    () => {
      return softwareAPI.getSoftwareInstallResult(installUuid);
    },
    {
      refetchOnWindowFocus: false,
      staleTime: 3000,
      select: (data) => data.results,
    }
  );

  if (isLoading) {
    return <Spinner />;
  } else if (isError) {
    return <DataError description="Close this modal and try again." />;
  } else if (!result) {
    // FIXME: Find a better solution for this.
    return <DataError description="No data returned." />;
  }

  return (
    <>
      <div className={`${baseClass}__software-install-details`}>
        <StatusMessage result={result} />
        {result.status !== "pending" && (
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
  installUuid,
  onCancel,
}: {
  installUuid: string;
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
          <SoftwareInstallDetails installUuid={installUuid} />
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
