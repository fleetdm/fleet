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
  onUpdate: (username: string) => void;
  isUpdating?: boolean;
  onExit: () => void;
}

const UpdateEndUserModal = ({
  isPremiumTier,
  endUsers,
  onUpdate,
  isUpdating,
  onExit,
}: IUpdateEndUserModalProps) => {
  const userNameDisplayValues = generateUsernameValues(endUsers);
  const isEditing = userNameDisplayValues.length > 0;
  const renderContent = () => {
    if (!isPremiumTier) {
      return <PremiumFeatureMessage />;
    }
    const onSave = () => {
      // TODO - call passed-in update with current form value
    };
    return (
      <>
        <div className={`${baseClass}__content`}>TODO</div>
        <div className="modal-cta-wrap">
          <Button
            isLoading={isUpdating}
            disabled={isUpdating}
            onClick={onUpdate}
          >
            Save
          </Button>
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
