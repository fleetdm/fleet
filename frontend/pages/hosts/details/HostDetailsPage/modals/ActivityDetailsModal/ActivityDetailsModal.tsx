import React from "react";
import { AxiosError } from "axios";
import { useQuery } from "react-query";

import { IMdmCommandResult } from "interfaces/mdm";
import mdmAPI, { IMdmCommandResultResponse } from "services/entities/mdm";

import DataError from "components/DataError";
import Modal from "components/Modal";
import Spinner from "components/Spinner";
import Button from "components/buttons/Button";
import Textarea from "components/Textarea";
import Icon from "components/Icon";

const baseClass = "activity-details-modal";

interface ICommandPayloadProps {
  payload: string;
}

const CommandPayload = ({ payload }: ICommandPayloadProps) => {
  return (
    <div className={`${baseClass}__command-payload`}>
      <span>Payload:</span>
      <Textarea className={`${baseClass}__payload-textarea`}>
        {atob(payload)}
      </Textarea>
    </div>
  );
};

interface ICommandResultProps {
  result: string;
  hostname: string;
  status: React.ReactNode;
}

const CommandResult = ({ result, hostname, status }: ICommandResultProps) => {
  return (
    <div className={`${baseClass}__command-result`}>
      <div>{status}</div>
      <p>
        The result from <b>{hostname}</b>:
      </p>
      <Textarea className={`${baseClass}__result-textarea`}>
        {atob(result)}
      </Textarea>
    </div>
  );
};

interface ICommandResultMessageProps {
  requestType: string;
  status: string;
}

const CommandResultMessage = ({
  requestType,
  status,
}: ICommandResultMessageProps) => {
  let statusIcon: "success-outline" | "error-outline" | "pending-outline" =
    "pending-outline";
  let message = "";

  const is200 = status.startsWith("2");
  const is400 = status.startsWith("4");
  const is500 = status.startsWith("5");
  const isSuccessful = is200 || status === "acknowledged";
  const isPending = status === "pending";
  const isFailed = is400 || is500 || status === "failed";

  if (requestType === "diskEncryption") {
    statusIcon = "error-outline";
    message = "Disk encryption failed.";
  } else if (isSuccessful) {
    statusIcon = "success-outline";
    message = "The host acknowledged the MDM command.";
  } else if (isPending) {
    statusIcon = "pending-outline";
    message =
      "The host will receive the MDM command when the host comes online.";
  } else if (isFailed) {
    statusIcon = "error-outline";
    message = "Failed.";
  }

  if (is200 || is400 || is500) {
    message = `${message} (Status Code: ${status})`;
  }

  return (
    <div className={`${baseClass}__command-result-message`}>
      <Icon name={statusIcon} />
      <p>{message}</p>
    </div>
  );
};

interface IActivityDetailsModalProps {
  commandUUID: string;
  onCancel: () => void;
}

const ActivityDetailsModal = ({
  commandUUID,
  onCancel,
}: IActivityDetailsModalProps) => {
  const { data, isLoading, isError } = useQuery<
    IMdmCommandResultResponse,
    AxiosError,
    IMdmCommandResult
  >("command-uuid", () => mdmAPI.getCommandResult(commandUUID), {
    retry: false,
    refetchOnWindowFocus: false,
    select: (res) => res.results[0],
  });

  const renderContent = () => {
    let content = <></>;

    if (isLoading) {
      content = <Spinner />;
    } else if (isError) {
      content = (
        <DataError
          className={`${baseClass}__error-message`}
          description="Close this modal and try again."
        />
      );
    } else if (data) {
      console.log(data);
      content = (
        <>
          <CommandPayload payload={data.payload} />
          <CommandResult
            result={data.result}
            hostname={data.hostname}
            status={
              <CommandResultMessage
                requestType={data.request_type}
                status={data.status}
              />
            }
          />
        </>
      );
    }

    return (
      <>
        <div className={`${baseClass}__modal-content`}>{content}</div>
        <div className="modal-cta-wrap">
          <Button onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </>
    );
  };

  return (
    <Modal
      title="Script details"
      width="large"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      {renderContent()}
    </Modal>
  );
};

export default ActivityDetailsModal;
