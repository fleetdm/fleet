import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";

const baseClass = "add-software-modal";

interface IAllTeamsMessageProps {
  onExit: () => void;
}

const AllTeamsMessage = ({ onExit }: IAllTeamsMessageProps) => {
  return (
    <>
      <p>
        Please select a team first. Software can&apos;t be added when{" "}
        <b>All teams</b> is selected.
      </p>
      <div className="modal-cta-wrap">
        <Button variant="brand" onClick={onExit}>
          Done
        </Button>
      </div>
    </>
  );
};

interface IAddSoftwareModalProps {
  onExit: () => void;
  isFreeTier?: boolean;
}

const AddSoftwareModal = ({ onExit, isFreeTier }: IAddSoftwareModalProps) => {
  const renderModalContent = () => {
    if (isFreeTier) {
      return (
        <>
          <PremiumFeatureMessage alignment="left" />{" "}
          <div className="modal-cta-wrap">
            <Button variant="brand" onClick={onExit}>
              Done
            </Button>
          </div>
        </>
      );
    }

    return <AllTeamsMessage onExit={onExit} />;
  };

  return (
    <Modal title="Add software" onExit={onExit} className={baseClass}>
      {renderModalContent()}
    </Modal>
  );
};

export default AddSoftwareModal;
