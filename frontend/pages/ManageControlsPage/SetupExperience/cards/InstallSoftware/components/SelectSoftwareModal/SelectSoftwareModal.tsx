import React, { useState } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

import SelectSoftwareTable from "../SelectSoftwareTable";

const baseClass = "select-software-modal";

interface ISelectSoftwareModalProps {
  onExit: () => void;
  onSave: () => void;
}

const SelectSoftwareModal = ({ onExit, onSave }: ISelectSoftwareModalProps) => {
  const [isSaving, setIsSaving] = useState(false);

  const onSaveSelectedSoftware = () => {
    onExit();
  };

  return (
    <Modal className={baseClass} title="Select software" onExit={onExit}>
      <>
        <SelectSoftwareTable />
        <div className="modal-cta-wrap">
          <Button
            variant="brand"
            onClick={onSaveSelectedSoftware}
            isLoading={isSaving}
          >
            Save
          </Button>
          <Button variant="inverse" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default SelectSoftwareModal;
