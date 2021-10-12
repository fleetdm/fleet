import React, { useState } from "react";

import Button from "components/buttons/Button";
import Modal from "components/modals/Modal";
import { ITeam } from "interfaces/team";
import { IEnrollSecret } from "interfaces/enroll_secret";
import PlatformWrapper from "./PlatformWrapper/PlatformWrapper";

const baseClass = "generate-installer-modal";

interface IGenerateInstallerModal {
  onCancel: () => void;
  selectedTeam: ITeam | { name: string; secrets: IEnrollSecret[] };
}

const GenerateInstallerModal = ({
  onCancel,
  selectedTeam,
}: IGenerateInstallerModal): JSX.Element => {
  return (
    <Modal onExit={onCancel} title={"Generate installer"} className={baseClass}>
      <PlatformWrapper
        certificate={"cool"}
        onCancel={onCancel}
        selectedTeam={selectedTeam}
      />
    </Modal>
  );
};

export default GenerateInstallerModal;
