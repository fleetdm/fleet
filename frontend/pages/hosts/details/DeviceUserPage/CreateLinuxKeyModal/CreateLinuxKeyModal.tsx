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
          Close this window and select <b>Refetch</b> on your <b>My device</b>{" "}
          page. This shares the new key with your organization.
        </li>
      </ol>
      <div className="modal-cta-wrap">
        <Button
          type="submit"
          variant="brand"
          onClick={onExit}
          className="save-loading"
        >
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
