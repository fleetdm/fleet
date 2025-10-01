import React from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

import ModalFooter from "components/ModalFooter";

interface IBitLockerPinModalProps {
  onCancel: () => void;
}

const baseClass = "bit-locker-pin-modal";

const BitLockerPinModal = ({
  onCancel,
}: IBitLockerPinModalProps): JSX.Element => {
  return (
    <Modal
      title="Create PIN"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
      width="large"
    >
      <>
        <div>
          <p>
            <ol>
              <li>
                <p>
                  Open the <b>Start menu</b>.
                </p>
              </li>
              <li>
                <p>Type &ldquo;Manage BitLocker&rdquo; and launch.</p>
              </li>
              <li>
                <p>
                  Click <b>Change how the drive is unlocked at startup</b>. If 
                  this option doesn&apos;t show up, wait a minute and
                  relaunch <b>Manage BitLocker</b>.
                </p>
              </li>
              <li>
                <p>
                  Choose <b>Enter a PIN (recommended)</b> and follow the prompts
                  to create a PIN.
                </p>
              </li>
              <li>
                <p>
                  Close this window and select <b>Refetch</b> on your{" "}
                  <b>My device</b> page. This informs your organization that you
                  have set a BitLocker PIN.
                </p>
              </li>
            </ol>
          </p>
        </div>
        <ModalFooter
          primaryButtons={<Button onClick={onCancel}>Done</Button>}
        />
      </>
    </Modal>
  );
};

export default BitLockerPinModal;
