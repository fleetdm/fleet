import React, { useCallback, useContext, useMemo, useState } from "react";

import { ISoftwareTitle } from "interfaces/software";
import { NotificationContext } from "context/notification";
import mdmAPI from "services/entities/mdm";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

import SelectSoftwareTable from "../SelectSoftwareTable";

const baseClass = "select-software-modal";

const initializeSelectedSoftwareIds = (softwareTitles: ISoftwareTitle[]) => {
  return softwareTitles.reduce<number[]>((acc, software) => {
    if (
      software.software_package?.install_during_setup ||
      software.app_store_app?.install_during_setup
    ) {
      acc.push(software.id);
    }
    return acc;
  }, []);
};

interface ISelectSoftwareModalProps {
  currentTeamId: number;
  softwareTitles: ISoftwareTitle[];
  onExit: () => void;
  onSave: () => void;
}

const SelectSoftwareModal = ({
  currentTeamId,
  softwareTitles,
  onExit,
  onSave,
}: ISelectSoftwareModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const initalSelectedSoftware = useMemo(
    () => initializeSelectedSoftwareIds(softwareTitles),
    [softwareTitles]
  );
  const [isSaving, setIsSaving] = useState(false);
  const [selectedSoftwareIds, setSelectedSoftwareIds] = useState<number[]>(
    initalSelectedSoftware
  );

  const onSaveSelectedSoftware = async () => {
    setIsSaving(true);
    try {
      await mdmAPI.updateSetupExperienceSoftware(
        currentTeamId,
        selectedSoftwareIds
      );
    } catch (e) {
      console.log("error");
      renderFlash("error", "Couldn't save software. Please try again.");
    }
    setIsSaving(false);
    onSave();
  };

  const onChangeSoftwareSelect = useCallback((select: boolean, id: number) => {
    setSelectedSoftwareIds((prevSelectedSoftwareIds) => {
      if (select) {
        return [...prevSelectedSoftwareIds, id];
      }
      return prevSelectedSoftwareIds.filter((selectedId) => selectedId !== id);
    });
  }, []);

  const onChangeSelectAll = useCallback(
    (selectAll: boolean) => {
      setSelectedSoftwareIds(selectAll ? softwareTitles.map((s) => s.id) : []);
    },
    [softwareTitles]
  );

  return (
    <Modal
      className={baseClass}
      title="Select software"
      width="large"
      onExit={onExit}
    >
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
