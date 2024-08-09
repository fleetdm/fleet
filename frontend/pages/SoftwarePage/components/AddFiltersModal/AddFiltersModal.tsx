import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

import PremiumFeatureMessage from "components/PremiumFeatureMessage";

const baseClass = "add-software-modal";

interface IAddFiltersModalProps {
  onExit: () => void;
  onSubmit: (filters: any) => void;
}

const AddFiltersModal = ({ onExit, onSubmit }: IAddFiltersModalProps) => {
  const renderModalContent = () => {
    return (
      <>
        <PremiumFeatureMessage alignment="left" />{" "}
        <div className="modal-cta-wrap">
          <Button variant="inverse" onClick={onExit}>
            Cancel
          </Button>
          <Button variant="brand" onClick={onSubmit}>
            Apply
          </Button>
        </div>
      </>
    );
  };

  return (
    <Modal title="Filters" onExit={onExit} className={baseClass}>
      {renderModalContent()}
    </Modal>
  );
};

export default AddFiltersModal;
