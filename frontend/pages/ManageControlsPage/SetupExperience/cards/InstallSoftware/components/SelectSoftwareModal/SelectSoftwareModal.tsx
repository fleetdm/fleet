import React, { useState } from "react";

import { ISoftwareTitle } from "interfaces/software";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

import SelectSoftwareTable from "../SelectSoftwareTable";

const baseClass = "select-software-modal";

interface ISelectSoftwareModalProps {
  software: ISoftwareTitle[];
  onExit: () => void;
  onSave: () => void;
}

const SelectSoftwareModal = ({
  software,
  onExit,
  onSave,
}: ISelectSoftwareModalProps) => {
  const [isSaving, setIsSaving] = useState(false);

  const onSaveSelectedSoftware = () => {
    onExit();
  };

  return (
    <Modal className={baseClass} title="Select software" onExit={onExit}>
      <>
        <SelectSoftwareTable software={software} />
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
