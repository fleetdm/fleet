import React, { useState } from "react";

import Button from "components/buttons/Button";
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
      <PlatformWrapper certificate={"cool"} onCancel={onCancel} />
    </Modal>
  );
};

export default GenerateInstallerModal;
