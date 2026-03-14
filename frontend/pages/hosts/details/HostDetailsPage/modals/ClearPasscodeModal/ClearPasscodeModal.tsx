import React, { useContext } from "react";

import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";
import hostAPI from "services/entities/hosts";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";

const baseClass = "clear-passcode-modal";

interface IClearPasscodeModalProps {
  id: number;
  hostName: string;
  onSuccess: () => void;
  onClose: () => void;
}

const ClearPasscodeModal = ({
  id,
  hostName,
  onSuccess,
  onClose,
}: IClearPasscodeModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [clearChecked, setClearChecked] = React.useState(false);
  const [isClearing, setIsClearing] = React.useState(false);

  const onClearPasscode = async () => {
    setIsClearing(true);
    try {
      await hostAPI.clearPasscode(id);
      onSuccess();
      renderFlash("success", "Passcode cleared.");
    } catch (e) {
      renderFlash("error", getErrorReason(e));
    }
    setIsClearing(false);
  };

  return (
    <Modal className={baseClass} title="Clear passcode" onExit={onClose}>
      <div className={`${baseClass}__modal-content`}>
        <div className={`${baseClass}__description`}>
          <p>
            Clearing the passcode allows the user to set a new passcode on the
            device.
          </p>
        </div>
        <div className={`${baseClass}__confirm-message`}>
          <span>
            <b>Confirm:</b>
          </span>
          <Checkbox
            wrapperClassName={`${baseClass}__clear-passcode-checkbox`}
            value={clearChecked}
            onChange={(value: boolean) => setClearChecked(value)}
          >
            I wish to clear the passcode on <b>{hostName}</b>
          </Checkbox>
        </div>
      </div>
      <div className="modal-cta-wrap">
        <Button
          type="button"
          onClick={onClearPasscode}
          className="delete-loading"
          disabled={!clearChecked}
          isLoading={isClearing}
        >
          Clear passcode
        </Button>
        <Button onClick={onClose} variant="inverse">
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default ClearPasscodeModal;
