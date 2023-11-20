import React, { useMemo } from "react";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { IHostMdmData } from "interfaces/host";

import OSSettingsTable from "./OSSettingsTable";
import { generateTableData } from "./OSSettingsTable/OSSettingsTableConfig";

interface IOSSettingsModalProps {
  platform?: string;
  hostMDMData?: IHostMdmData;
  onClose: () => void;
}

const baseClass = "os-settings-modal";

const OSSettingsModal = ({
  platform,
  hostMDMData,
  onClose,
}: IOSSettingsModalProps) => {
  // the caller should ensure that hostMDMData is not undefined and that platform is "windows" or
  // "darwin", otherwise we will allow an empty modal will be rendered.
  // https://fleetdm.com/handbook/company/why-this-way#why-make-it-obvious-when-stuff-breaks

  const memoizedTableData = useMemo(
    () => generateTableData(hostMDMData, platform),
    [hostMDMData, platform]
  );

  return (
    <Modal
      title="OS settings"
      onExit={onClose}
      className={baseClass}
      width="large"
    >
      <>
        <OSSettingsTable tableData={memoizedTableData || []} />
        <div className="modal-cta-wrap">
          <Button variant="brand" onClick={onClose}>
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default OSSettingsModal;
