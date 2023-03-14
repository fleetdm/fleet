import React, { useState } from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

interface IResetKeyModalProps {
  onCancel: () => void;
}

const baseClass = "reset-key-modal";

const ResetKeyModal = ({ onCancel }: IResetKeyModalProps): JSX.Element => {
  const [success, setSuccess] = useState<boolean>(false);
  const [isLoading, setIsLoading] = useState<boolean>(false);

  //  TODO: actually make this work: https://www.figma.com/file/hdALBDsrti77QuDNSzLdkx/%F0%9F%9A%A7-Fleet-EE-(dev-ready%2C-scratchpad)?node-id=11728%3A323033&t=GbmGwTkgjENhmJmO-1
  const startNativeKeyReset = () => {
    setIsLoading(true);
    setTimeout(() => {
      setSuccess(true);
    }, 1000);
  };

  return (
    <Modal title="Reset key" onExit={onCancel} className={baseClass}>
      <div>
        <ol>
          <li>
            Click <b>Start</b> and enter your username and password.
            {success ? (
              <div className={`${baseClass}__success`}>Success!</div>
            ) : (
              <Button
                type="button"
                onClick={startNativeKeyReset}
                variant="brand"
                className={`${baseClass}__start-button`}
                isLoading={isLoading}
              >
                Start
              </Button>
            )}
          </li>
          <li>
            Close this window and select <b>Refetch</b> on your My device page.
            This tells your organization that you reset your key.
          </li>
        </ol>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default ResetKeyModal;
