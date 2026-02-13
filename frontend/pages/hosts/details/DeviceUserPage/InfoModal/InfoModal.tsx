import React from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import CustomLink from "components/CustomLink";

export interface IInfoModalProps {
  onCancel: () => void;
  transparencyURL?: string;
}

const baseClass = "device-user-info";

const InfoModal = ({
  onCancel,
  transparencyURL,
}: IInfoModalProps): JSX.Element => {
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
          <CustomLink
            url={transparencyURL || "https://fleetdm.com/transparency"}
            text="Read about transparency"
            newTab
            multiline
          />
        </p>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={onCancel}>
            OK
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default InfoModal;
