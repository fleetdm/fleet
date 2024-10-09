import React, { useContext, useState } from "react";

import { ISoftwareTitle } from "interfaces/software";
import { NotificationContext } from "context/notification";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

import SelectSoftwareTable from "../SelectSoftwareTable";

const baseClass = "select-software-modal";

const initializeSelectedSoftwareIds = (softwareTitles: ISoftwareTitle[]) => {
  return softwareTitles.reduce<number[]>((acc, software) => {
    if (software.install_during_setup) {
      acc.push(software.id);
    }
    return acc;
  }, []);
};

interface ISelectSoftwareModalProps {
  softwareTitles: ISoftwareTitle[];
  onExit: () => void;
  onSave: () => void;
}

const SelectSoftwareModal = ({
  softwareTitles,
  onExit,
  onSave,
}: ISelectSoftwareModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const [isSaving, setIsSaving] = useState(false);
  const [selectedSoftwareIds, setSelectedSoftwareIds] = useState<number[]>(() =>
    initializeSelectedSoftwareIds(softwareTitles)
  );

  const onSaveSelectedSoftware = () => {
    console.log(selectedSoftwareIds);
    onExit();
  };

  const onChangeSoftwareSelect = (select: boolean, id: number) => {
    setSelectedSoftwareIds((prevSelectedSoftwareIds) => {
      if (select) {
        return [...prevSelectedSoftwareIds, id];
      }
      return prevSelectedSoftwareIds.filter((selectedId) => selectedId !== id);
    });
  };

  const onChangeSelectAll = (selectAll: boolean) => {
    setSelectedSoftwareIds(selectAll ? softwareTitles.map((s) => s.id) : []);
  };

  return (
    <Modal className={baseClass} title="Select software" onExit={onExit}>
      <>
        <SelectSoftwareTable
          softwareTitles={softwareTitles}
          onChangeSoftwareSelect={onChangeSoftwareSelect}
          onChangeSelectAll={onChangeSelectAll}
        />
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
