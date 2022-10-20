import React from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

import ExternalLinkIcon from "../../../../../../assets/images/icon-external-link-12x12@2x.png";

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
            target="_blank"
            rel="noopener noreferrer"
          >
            Read about{" "}
            <span className="no-wrap">
              transparency
              <img
                className="external-link-icon"
                src={ExternalLinkIcon}
                alt="Open external link"
              />
            </span>
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
