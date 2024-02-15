import React, { useContext } from "react";

import hostAPI from "services/entities/hosts";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import { NotificationContext } from "context/notification";
import { AxiosError } from "axios";

const baseClass = "wipe-modal";

interface IWipeModalProps {
  id: number;
  hostName: string;
  onSuccess: () => void;
  onClose: () => void;
}

const WipeModal = ({ id, hostName, onSuccess, onClose }: IWipeModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [lockChecked, setLockChecked] = React.useState(false);
  const [isLocking, setIsLocking] = React.useState(false);

  const onLock = async () => {
    setIsLocking(true);
    try {
      await hostAPI.wipeHost(id);
      onSuccess();
      renderFlash("success", "Success! Host is wiping.");
    } catch (error) {
      const err = error as AxiosError;
      renderFlash("error", err.message);
    }
    onClose();
    setIsLocking(false);
  };

  return (
    <Modal className={baseClass} title="Wipe host" onExit={onClose}>
      <>
        <div className={`${baseClass}__modal-content`}>
          <p>All content will be erased on this host.</p>
          <div className={`${baseClass}__confirm-message`}>
            <span>
              <b>Please check to confirm:</b>
            </span>
            <Checkbox
              wrapperClassName={`${baseClass}__wipe-checkbox`}
              value={lockChecked}
              onChange={(value: boolean) => setLockChecked(value)}
            >
              I wish to wipe <b>{hostName}</b>
            </Checkbox>
          </div>
        </div>

        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={onLock}
            variant="alert"
            className="delete-loading"
            disabled={!lockChecked}
            isLoading={isLocking}
          >
            Wipe
          </Button>
          <Button onClick={onClose} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default WipeModal;
