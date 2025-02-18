import React, { useState, useContext } from "react";

import DataError from "components/DataError";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { NotificationContext } from "context/notification";

import mdmAPI from "services/entities/mdm";

interface IUnenrollMdmModalProps {
  hostId: number;
  onClose: () => void;
}

const baseClass = "unenroll-mdm-modal";

const UnenrollMdmModal = ({ hostId, onClose }: IUnenrollMdmModalProps) => {
  const [requestState, setRequestState] = useState<
    undefined | "unenrolling" | "error"
  >(undefined);

  const { renderFlash } = useContext(NotificationContext);

  const submitUnenrollMdm = async () => {
    setRequestState("unenrolling");
    try {
      await mdmAPI.unenrollHostFromMdm(hostId, 5000);
      renderFlash(
        "success",
        "Turning off MDM or will turn off when the host comes online."
      );
      onClose();
    } catch (unenrollMdmError: unknown) {
      renderFlash("error", "Couldn't turn off MDM. Please try again.");
      console.log(unenrollMdmError);
      onClose();
    }
  };

  const renderModalContent = () => {
    if (requestState === "error") {
      return <DataError />;
    }
    return (
      <>
        <p className={`${baseClass}__description`}>
          Settings configured by Fleet will be removed.
          <br />
          <br />
          To turn on MDM again, ask the device user to follow the{" "}
          <b>Turn on MDM</b> instructions on their <b>My device</b> page.
        </p>
        <div className="modal-cta-wrap">
          <Button
            type="submit"
            variant="alert"
            onClick={submitUnenrollMdm}
            isLoading={requestState === "unenrolling"}
          >
            Turn off
          </Button>
          <Button onClick={onClose} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </>
    );
  };

  return (
    <Modal
      title="Turn off MDM"
      onExit={onClose}
      className={baseClass}
      width="large"
    >
      {renderModalContent()}
    </Modal>
  );
};

export default UnenrollMdmModal;
