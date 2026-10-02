import React from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

import exampleImage from "../../../../../../../../../../assets/images/windows-eula-example-600x390@2x.png";

const baseClass = "example-windows-eula-modal";

interface IExampleWindowsEulaModalProps {
  onExit: () => void;
}

const ExampleWindowsEulaModal = ({ onExit }: IExampleWindowsEulaModalProps) => {
  return (
    <Modal
      className={baseClass}
      title="Example Windows EULA"
      onExit={onExit}
      onEnter={onExit}
      width="large"
    >
      <>
        <p>
          An example of how the Windows <b>End user agreement (EULA)</b> will
          display to end users.
        </p>
        <img
          className={`${baseClass}__image`}
          src={exampleImage}
          alt="Windows setup showing a terms and conditions page with an Agree and continue button"
        />
        <div className="modal-cta-wrap">
          <Button onClick={onExit}>Close</Button>
        </div>
      </>
    </Modal>
  );
};

export default ExampleWindowsEulaModal;
