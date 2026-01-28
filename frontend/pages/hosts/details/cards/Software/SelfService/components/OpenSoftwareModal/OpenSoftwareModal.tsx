import React, { ReactNode } from "react";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Button from "components/buttons/Button";

const baseClass = "software-instructions-modal";

const getOpenSoftwareInstructions = (
  softwareName: string,
  softwareSource: string
): ReactNode => {
  if (softwareSource === "apps")
    return (
      <p>
        Find <b>{softwareName}</b> in <b>Finder &gt; Applications</b> and
        double-click it, or search <b>{softwareName}</b> in <b>Spotlight</b>.
      </p>
    );
  else if (softwareSource === "programs") {
    return (
      <p>
        Find <b>{softwareName}</b> in <b>Start Menu</b> and click it, or search
        for it using the taskbar search box.
      </p>
    );
  }
  return <></>;
};
interface ISoftwareInstructionsModalProps {
  softwareName: string;
  softwareSource: string;
  onExit: () => void;
}

const OpenSoftwareModal = ({
  softwareName,
  softwareSource,
  onExit,
}: ISoftwareInstructionsModalProps) => {
  return (
    <Modal className={baseClass} title="How to open" onExit={onExit}>
      <>
        {getOpenSoftwareInstructions(softwareName, softwareSource)}
        <ModalFooter primaryButtons={<Button onClick={onExit}>Done</Button>} />
      </>
    </Modal>
  );
};

export default OpenSoftwareModal;
