import React from "react";

import CustomLink from "components/CustomLink";
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
          <p>Currently, Username (IdP) can be added in the following ways:</p>
          <ul style={{ listStyle: "disc", paddingLeft: "20px" }}>
            <li style={{ marginBottom: "10px" }}>
              <b>Automatically:</b> A username is added when the host
              automatically enrolls (ADE), if{" "}
              <CustomLink
                url={paths.CONTROLS_END_USER_AUTHENTICATION}
                text="end user authentication"
              />{" "}
              is enabled.
            </li>
            <li>
              <b>Manually:</b> Usernames can be added or updated via the{" "}
              <CustomLink
                url="https://fleetdm.com/learn-more-about/edit-idp-username"
                text="REST API"
                newTab
              />
            </li>
          </ul>
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onExit}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default AddEndUserModal;
