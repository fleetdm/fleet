import React, { useContext } from "react";

import { AppContext } from "context/app";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

interface DeleteProfileModalProps {
  profileName: string;
  profileId: string;
  onCancel: () => void;
  onDelete: (profileId: string) => void;
  isDeleting: boolean;
}

const baseClass = "delete-profile-modal";

const generateMessageSuffix = (isPremiumTier?: boolean, teamId?: number) => {
  if (!isPremiumTier) {
    return "";
  }
  return teamId ? "assigned to this team" : "with no team";
};

const DeleteProfileModal = ({
  profileName,
  profileId,
  onCancel,
  onDelete,
  isDeleting,
}: DeleteProfileModalProps) => {
  const { isPremiumTier, currentTeam } = useContext(AppContext);

  const messageSuffix = generateMessageSuffix(isPremiumTier, currentTeam?.id);

  return (
    <Modal
      className={baseClass}
      title="Delete configuration profile"
      onExit={onCancel}
      onEnter={() => onDelete(profileId)}
      width="large"
    >
      <>
        <div className={`${baseClass}__content`}>
          <p>
            This action will remove the <b>{profileName}</b> configuration
            profile from all hosts {messageSuffix}.
          </p>
          <p>Pending profiles will be canceled.</p>
        </div>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={() => onDelete(profileId)}
            variant="alert"
            className="delete-loading"
            isLoading={isDeleting}
          >
            Delete
          </Button>
          <Button onClick={onCancel} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeleteProfileModal;
