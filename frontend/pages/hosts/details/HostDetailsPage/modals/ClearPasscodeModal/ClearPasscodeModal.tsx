import React, { useContext } from "react";

import { NotificationContext } from "context/notification";
import hostAPI from "services/entities/hosts";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "clear-passcode-modal";

interface IClearPasscodeModalProps {
  id: number;
  onExit: () => void;
}

const ClearPasscodeModal = ({ id, onExit }: IClearPasscodeModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isClearingPasscode, setIsClearingPasscode] = React.useState(false);

  const onClearPasscode = async () => {
    setIsClearingPasscode(true);
    try {
      await hostAPI.clearPasscode(id);
      renderFlash(
        "success",
        "Successfully sent request to clear passcode on this host."
      );
    } catch (e) {
      renderFlash(
        "error",
        "Couldn't send request to clear passcode on this host. Please try again."
      );
    } finally {
      onExit();
      setIsClearingPasscode(false);
    }
  };

  const renderModalContent = () => {
    return (
      <p>
        This will remove the current passcode and allow anyone with physical
        access to unlock the host.
      </p>
    );
  };

  const renderModalButtons = () => {
    return (
      <>
        <Button
          type="button"
          onClick={onClearPasscode}
          className="clear-passcode-loading"
          variant="alert"
          isLoading={isClearingPasscode}
        >
          Clear Passcode
        </Button>
        <Button onClick={onExit} variant="inverse-alert">
          Cancel
        </Button>
      </>
    );
  };

  return (
    <Modal className={baseClass} title="Clear passcode" onExit={onExit}>
      <div className={`${baseClass}__modal-content`}>
        {renderModalContent()}
      </div>

      <div className="modal-cta-wrap">{renderModalButtons()}</div>
    </Modal>
  );
};

export default ClearPasscodeModal;
