import React, { useState } from "react";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import { getErrorReason } from "interfaces/errors";
import hostAPI from "services/entities/hosts";

const baseClass = "reset-modal";

interface IResetModalProps {
  id: number;
  hostName: string;
  onSuccess: () => void;
  onClose: () => void;
}

const ResetModal = ({ id, hostName, onSuccess, onClose }: IResetModalProps) => {
  const [isConfirmed, setIsConfirmed] = useState(false);
  const [isResetting, setIsResetting] = useState(false);

  const onReset = async () => {
    setIsResetting(true);
    try {
      await hostAPI.resetHost(id);
      onSuccess();
      notify.success(
        "Resetting host or will reset when the host comes online."
      );
    } catch (e) {
      notify.error(getErrorReason(e), { response: e });
    }
    onClose();
    setIsResetting(false);
  };

  return (
    <Modal className={baseClass} title="Reset" onExit={onClose}>
      <div className={`${baseClass}__modal-content`}>
        <p>Repurpose the host for another end user.</p>
        <p>
          For lost or stolen hosts, use <b>Actions &gt; Wipe</b>.
        </p>
        <div>
          <span>
            <b>Confirm:</b>
          </span>
          <Checkbox
            wrapperClassName={`${baseClass}__reset-checkbox`}
            value={isConfirmed}
            onChange={(value: boolean) => setIsConfirmed(value)}
            variant="danger"
          >
            I wish to reset <b>{hostName}</b>
          </Checkbox>
        </div>
      </div>

      <div className="modal-cta-wrap">
        <Button
          type="button"
          onClick={onReset}
          variant="alert"
          className="delete-loading"
          disabled={!isConfirmed || isResetting}
          isLoading={isResetting}
        >
          Reset
        </Button>
        <Button onClick={onClose} variant="secondary">
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default ResetModal;
