import React, { useContext } from "react";

import { AppContext } from "context/app";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

interface DeleteProfileModalProps {
  profileName: string;
  profileId: number;
  onCancel: () => void;
  onDelete: (profileId: number) => void;
}

const baseClass = "delete-profile-modal";

const generateMessageSuffix = (isPremiumTier?: boolean, teamId?: number) => {
  if (!isPremiumTier) {
    return "";
  }
  return teamId ? " assigned to this team" : " with no team";
};

const DeleteProfileModal = ({
  profileName,
  profileId,
  onCancel,
  onDelete,
}: DeleteProfileModalProps) => {
  const { isPremiumTier, currentTeam } = useContext(AppContext);

  const messageSuffix = generateMessageSuffix(isPremiumTier, currentTeam?.id);

  return (
    <Modal
      className={baseClass}
      title={"Delete configuration profile"}
      onExit={onCancel}
      onEnter={() => onDelete(profileId)}
      width="large"
    >
      <>
        <p>
          This action will delete configuration profile{" "}
          <span className={`${baseClass}__profile-name`}>{profileName}</span>{" "}
          from all macOS hosts{messageSuffix}.
        </p>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={() => onDelete(profileId)}
            variant="alert"
            className="delete-loading"
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
