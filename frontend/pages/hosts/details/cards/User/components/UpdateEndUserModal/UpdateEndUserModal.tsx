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
  /** There will be at most 1 end user */
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
  // at most 1
  const userNameDisplayValue = generateUsernameValues(endUsers)[0];
  const isEditing = !!userNameDisplayValue;
  const [idpUsername, setIdpUsername] = useState(userNameDisplayValue || "");

  const onSave = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    onUpdate(idpUsername);
  };

  const renderContent = () => {
    if (!isPremiumTier) {
      return <PremiumFeatureMessage />;
    }
    return (
      <>
        <form onSubmit={onSave}>
          <InputField
            label="Username (IdP)"
            name="username_idp"
            value={idpUsername}
            onChange={(val: string) => setIdpUsername(val)}
            helpText="This will be used to populate additional user data, e.g. full name and department."
            autofocus
          />

          <div className="modal-cta-wrap">
            <Button
              isLoading={isUpdating}
              disabled={isUpdating || (!isEditing && idpUsername === "")}
              type="submit"
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
