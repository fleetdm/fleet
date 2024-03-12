import React from "react";
import { AxiosError } from "axios";
import { useQuery } from "react-query";

import { IMdmCommandResult } from "interfaces/mdm";
import mdmAPI, { IMdmCommandReultResponse } from "services/entities/mdm";

import DataError from "components/DataError";
import Modal from "components/Modal";
import Spinner from "components/Spinner";
import Button from "components/buttons/Button";

const baseClass = "activity-details-modal";

const CommandPayload = ({ payload }: { payload: string }) => {
  return <>payload</>;
};

const CommandResult = ({ result }: { result: string }) => {
  return <>result</>;
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
    IMdmCommandReultResponse,
    AxiosError,
    IMdmCommandResult
  >("command-uuid", () => mdmAPI.getCommandResult(commandUUID), {
    retry: false,
    refetchOnWindowFocus: false,
  });

  const renderContent = () => {
    let content = <></>;

    if (false) {
      content = <Spinner />;
    } else if (false) {
      content = (
        <DataError
          className={`${baseClass}__error-message`}
          description="Close this modal and try again."
        />
      );
    } else if (data) {
      content = (
        <>
          <CommandPayload payload="test" />
          <CommandResult result="test" />
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
