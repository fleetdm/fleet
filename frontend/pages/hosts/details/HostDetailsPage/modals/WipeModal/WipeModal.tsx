import React, { useContext } from "react";

import hostAPI from "services/entities/hosts";
import { getErrorReason } from "interfaces/errors";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import { NotificationContext } from "context/notification";

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
  const [isWiping, setIsWiping] = React.useState(false);

  const onWipe = async () => {
    setIsWiping(true);
    try {
      await hostAPI.wipeHost(id);
      onSuccess();
      renderFlash(
        "success",
        "Wiping host or will wipe when the host comes online."
      );
    } catch (e) {
      renderFlash("error", getErrorReason(e));
    }
    onClose();
    setIsWiping(false);
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
            onClick={onWipe}
            variant="alert"
            className="delete-loading"
            disabled={!lockChecked}
            isLoading={isWiping}
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
