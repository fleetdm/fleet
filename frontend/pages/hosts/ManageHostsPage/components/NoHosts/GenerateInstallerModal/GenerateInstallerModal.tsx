import React, { useState } from "react";

import Modal from "components/modals/Modal";
import PlatformWrapper from "./PlatformWrapper/PlatformWrapper";

const baseClass = "generate-installer-modal";

interface IGenerateInstallerModal {
  onCancel: () => void;
}

const GenerateInstallerModal = ({
  onCancel,
}: IGenerateInstallerModal): JSX.Element => {
  const [selectedMembers, setSelectedMembers] = useState([]);

  return (
    <Modal onExit={onCancel} title={"Generate installer"} className={baseClass}>
      <PlatformWrapper />
    </Modal>
  );
};

export default GenerateInstallerModal;
