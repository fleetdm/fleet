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
    let timeout;
    try {
      timeout = setTimeout(() => {
        throw new Error("Unenroll request timed out");
      }, 5000);
      console.log(`inital timeout: ${timeout}`);

      // await mdmAPI.unenrollHostFromMdm(hostId);
      // simulate slow network response
      const response = await new Promise((resolve) =>
        setTimeout(resolve, 6000)
      );

      clearTimeout(timeout);
      renderFlash("success", "Successfully turned off MDM.");
      onClose();
    } catch (unenrollMdmError: unknown) {
      clearTimeout(timeout);
      console.log(unenrollMdmError);
      setRequestState("error");
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
            variant="brand"
            onClick={submitUnenrollMdm}
            isLoading={requestState === "unenrolling"}
          >
            Turn off
          </Button>
          <Button onClick={onClose} variant="inverse">
            Cancel
          </Button>
        </div>
      </>
    );
  };

  return (
    <Modal title="Turn off MDM" onExit={onClose} className={baseClass}>
      {renderModalContent()}
    </Modal>
  );
};

export default UnenrollMdmModal;
