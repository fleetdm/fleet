import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "turn-off-android-mdm-modal";

interface ITurnOffAndroidMdmModalProps {
  onExit: () => void;
}

const TurnOffAndroidMdmModal = ({ onExit }: ITurnOffAndroidMdmModalProps) => {
  return (
    <Modal title="Turn off Android MDM" className={baseClass} onExit={onExit}>
      <>
        <p>
          If you want to use MDM features again, you&apos;ll have to reconnect
          Android Enterprise.
        </p>
        <p>
          End users will lose access to organization resources and all data in
          their Android work partition.
        </p>
        <div className="modal-cta-wrap">
          <Button onClick={onExit} variant="alert">
            Turn off
          </Button>
          <Button onClick={onExit} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default TurnOffAndroidMdmModal;
