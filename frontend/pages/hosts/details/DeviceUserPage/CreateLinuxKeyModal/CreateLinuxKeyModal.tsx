import Button from "components/buttons/Button";
import Modal from "components/Modal";
import React from "react";

const baseClass = "create-linux-key-modal";

interface ICreateLinuxKeyModal {
  isTriggeringCreateLinuxKey: boolean;
  onExit: () => void;
}

const CreateLinuxKeyModal = ({
  isTriggeringCreateLinuxKey,
  onExit,
}: ICreateLinuxKeyModal) => {
  const renderModalBody = () => (
    <>
      <ol>
        <li>
          Wait 30 seconds for the <b>Enter disk encryption passphrase</b> pop-up
          to open.
        </li>
        <li>
          In the pop-up, enter the passphrase used to encrypt your device during
          setup.
        </li>
        <li>
          You&apos;re done. The yellow <b>Disk Encryption</b> banner will go
          away in 1 hour. To remove this banner sooner, wait 10 minutes for
          Fleet to create a new key. Then, close this window and select{" "}
          <b>Refetch</b> on your <b>My Device</b> page.
        </li>
        <li>
          If the banner doesn&apos;t go away after 1 hour, please contact your
          IT admin.
        </li>
      </ol>
      <div className="modal-cta-wrap">
        <Button type="submit" onClick={onExit} className="save-loading">
          Done
        </Button>
      </div>
    </>
  );
  return (
    <Modal
      title="Create key"
      onExit={onExit}
      onEnter={onExit}
      className={baseClass}
      isLoading={isTriggeringCreateLinuxKey}
    >
      {renderModalBody()}
    </Modal>
  );
};

export default CreateLinuxKeyModal;
