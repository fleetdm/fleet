import React, { useCallback, useContext, useMemo, useState } from "react";

import { SetupExperiencePlatform } from "interfaces/platform";
import { ISoftwareTitle } from "interfaces/software";
import { NotificationContext } from "context/notification";
import mdmAPI from "services/entities/mdm";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

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
  platform: SetupExperiencePlatform;
  onExit: () => void;
  onSave: () => void;
}

const SelectSoftwareModal = ({
  currentTeamId,
  softwareTitles,
  platform,
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
        platform,
        currentTeamId,
        selectedSoftwareIds
      );
      renderFlash("success", "Updated software for install on setup.");
    } catch (e) {
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
          platform={platform}
        />
        <div className="modal-cta-wrap">
          <GitOpsModeTooltipWrapper
            tipOffset={6}
            renderChildren={(disableChildren) => (
              <Button
                disabled={disableChildren}
                onClick={onSaveSelectedSoftware}
                isLoading={isSaving}
              >
                Save
              </Button>
            )}
          />
          <Button variant="inverse" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default SelectSoftwareModal;
