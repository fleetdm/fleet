import React, { useState } from "react";

import { IHostEndUser } from "interfaces/host";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
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
  // TODO - reconcile impliciation of multiple end users
  const userNameDisplayValues = generateUsernameValues(endUsers);
  const isEditing = userNameDisplayValues.length > 0;
  const [idpUsername, setIdpUsername] = useState(
    isEditing ? userNameDisplayValues[0] : ""
  );

  const onSave = () => {
    onUpdate(idpUsername);
  };

  const renderContent = () => {
    if (!isPremiumTier) {
      return <PremiumFeatureMessage />;
    }
    return (
      <>
        <form>
          <InputField
            label="Username (IdP)"
            name="username_idp"
            value={idpUsername}
            onChange={(val: string) => setIdpUsername(val)}
            helpText="This will be used to populate additional user data, e.g. full name and department."
          />

          <div className="modal-cta-wrap">
            <Button
              isLoading={isUpdating}
              disabled={isUpdating}
              onClick={onSave}
            >
              Save
            </Button>
          </div>
        </form>
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
