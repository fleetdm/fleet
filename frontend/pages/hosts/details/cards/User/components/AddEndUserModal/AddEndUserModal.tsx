import React from "react";
import { Link } from "react-router";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

import paths from "router/paths";

const baseClass = "add-end-user-modal";

interface IAddEndUserModalProps {
  onExit: () => void;
}

const AddEndUserModal = ({ onExit }: IAddEndUserModalProps) => {
  return (
    <Modal title="Add user" onExit={onExit} className={baseClass}>
      <>
        <div className={`${baseClass}__content`}>
          <p>
            Currently, <b>Username (IdP)</b> is only added when the host
            automatically enrolls (ADE).{" "}
          </p>
          <p>
            To add username when hosts enroll in the future, enable{" "}
            <Link to={paths.CONTROLS_END_USER_AUTHENTICATION}>
              end user authentication
            </Link>
            .
          </p>
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onExit}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default AddEndUserModal;
