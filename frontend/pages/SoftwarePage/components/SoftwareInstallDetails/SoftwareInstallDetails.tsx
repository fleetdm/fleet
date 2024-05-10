import React from "react";
import { useQuery } from "react-query";

import {
  ISoftwareInstallResult,
  ISoftwareInstallResults,
  SoftwareInstallStatus,
} from "interfaces/software";
import softwareAPI from "services/entities/software";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import Textarea from "components/Textarea";
import DataError from "components/DataError/DataError";
import Spinner from "components/Spinner/Spinner";
import { IconNames } from "components/icons";

const baseClass = "software-install-details";

const STATUS_ICONS: Record<SoftwareInstallStatus, IconNames> = {
  pending: "pending-outline",
  installed: "success-outline",
  failed: "error-outline",
} as const;

const STATUS_PREDICATES: Record<SoftwareInstallStatus, string> = {
  pending: "will install",
  installed: "installed",
  failed: "failed to install",
} as const;

const StatusMessage = ({
  result: { host_display_name, software_package, software_title, status },
}: {
  result: ISoftwareInstallResult;
}) => {
  return (
    <div className={`${baseClass}__status-message`}>
      <Icon name={STATUS_ICONS[status]} />
      <span>
        Fleet {STATUS_PREDICATES[status]} <b>{software_title}</b> (
        {software_package}) on <b>{host_display_name}</b>
        {status === "pending" ? " when it comes online" : ""}.
      </span>
    </div>
  );
};

const OUTPUT_DISPLAY_LABELS = {
  pre_install_query_output: "Pre-install condition",
  output: "Software install output",
  post_install_script_output: "Post-install script output",
} as const;

const Output = ({
  displayKey,
  result,
}: {
  displayKey: keyof typeof OUTPUT_DISPLAY_LABELS;
  result: ISoftwareInstallResult;
}) => {
  return (
    <div className={`${baseClass}__script-output`}>
      {OUTPUT_DISPLAY_LABELS[displayKey]}:
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
