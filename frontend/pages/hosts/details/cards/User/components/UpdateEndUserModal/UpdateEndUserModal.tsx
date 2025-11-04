import React from "react";

import { IHostEndUser } from "interfaces/host";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";

import { generateUsernameValues } from "../../helpers";

const baseClass = "update-end-user-modal";

interface IUpdateEndUserModalProps {
  isPremiumTier: boolean;
  endUsers: IHostEndUser[];
  onExit: () => void;
}

const UpdateEndUserModal = ({
  isPremiumTier,
  endUsers,
  onExit,
}: IUpdateEndUserModalProps) => {
  const userNameDisplayValues = generateUsernameValues(endUsers);
  const isEditing = userNameDisplayValues.length > 0;
  const renderContent = () => {
    if (!isPremiumTier) {
      return <PremiumFeatureMessage />;
    }
    return (
      <>
        <div className={`${baseClass}__content`}>TODO</div>
        <div className="modal-cta-wrap">
          <Button onClick={onExit}>Save</Button>
        </div>
      </>
    );
  };

  return (
    <Modal
      title={isEditing ? "Edit user" : "Add user"}
      onExit={onExit}
      className={baseClass}
    >
      {renderContent()}
    </Modal>
  );
};

export default UpdateEndUserModal;
