import React, { useState, useContext } from "react";

import DataError from "components/DataError";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { NotificationContext } from "context/notification";

interface IUnenrollMdmModalProps {
  onClose: () => void;
}

const baseClass = "unenroll-mdm-modal";

const UnenrollMdmModal = ({ onClose }: IUnenrollMdmModalProps) => {
  const [requestState, setRequestState] = useState<
    undefined | "unenrolling" | "error"
  >(undefined);

  const { renderFlash } = useContext(NotificationContext);

  const submitUnenrollMdm = async () => {
    setRequestState("unenrolling");
    try {
      const timeout = setTimeout(() => {
        throw new Error("Unenroll request timed out");
      }, 5000);
      const response = await (timeout) => {
        clearInterval(timeout);
      }; 
      renderFlash("success", "Successfully turned off MDM.");
      onClose();
    } catch (unenrollMdmError: unknown) {
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
            isLoading={requestState === "loading"}
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
