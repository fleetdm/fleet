import React from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

import OpenNewTabIcon from "../../../../../../assets/images/open-new-tab-12x12@2x.png";

export interface IInfoModalProps {
  onCancel: () => void;
}

const baseClass = "device-user-info";

const InfoModal = ({ onCancel }: IInfoModalProps): JSX.Element => {
  return (
    <Modal
      title="Welcome to Fleet"
      onExit={onCancel}
      className={`${baseClass}__modal`}
    >
      <div>
        <p>
          Your organization uses Fleet to check if all devices meet its security
          policies.
        </p>
        <p>With Fleet, you and your team can secure your device, together.</p>
        <p>
          Want to know what your organization can see?&nbsp;
          <a
            href="https://fleetdm.com/transparency"
            className={`${baseClass}__learn-more ${baseClass}__learn-more--inline`}
            target="_blank"
            rel="noopener noreferrer"
          >
            Read about transparency&nbsp;
            <img className="icon" src={OpenNewTabIcon} alt="open new tab" />
          </a>
        </p>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={onCancel} variant="brand">
            Ok
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default InfoModal;
