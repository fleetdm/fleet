import React from "react";

import Modal from "components/Modal";
import { ITeam } from "interfaces/team";
import { IEnrollSecret } from "interfaces/enroll_secret";
import PlatformWrapper from "./PlatformWrapper/PlatformWrapper";

const baseClass = "add-hosts-modal";

interface IAddHostsModal {
  onCancel: () => void;
  selectedTeam: ITeam | { name: string; secrets: IEnrollSecret[] | null };
}

const AddHostsModal = ({
  onCancel,
  selectedTeam,
}: IAddHostsModal): JSX.Element => {
  return (
    <Modal onExit={onCancel} title={"Add hosts"} className={baseClass}>
      <PlatformWrapper onCancel={onCancel} selectedTeam={selectedTeam} />
    </Modal>
  );
};

export default AddHostsModal;
