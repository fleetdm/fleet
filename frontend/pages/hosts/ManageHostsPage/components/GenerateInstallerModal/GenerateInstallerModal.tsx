import React from "react";

import Modal from "components/Modal";
import { ITeam } from "interfaces/team";
import { IEnrollSecret } from "interfaces/enroll_secret";
import PlatformWrapper from "./PlatformWrapper/PlatformWrapper";

const baseClass = "generate-installer-modal";

interface IGenerateInstallerModal {
  onCancel: () => void;
  selectedTeam: ITeam | { name: string; secrets: IEnrollSecret[] | null };
}

const GenerateInstallerModal = ({
  onCancel,
  selectedTeam,
}: IGenerateInstallerModal): JSX.Element => {
  return (
    <Modal onExit={onCancel} title={"Generate installer"} className={baseClass}>
      <PlatformWrapper onCancel={onCancel} selectedTeam={selectedTeam} />
    </Modal>
  );
};

export default GenerateInstallerModal;
