import React, { useContext } from "react";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Modal from "components/Modal";
import { AppContext } from "context/app";
import { ProfilePlatform } from "interfaces/mdm";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

interface DeleteProfileModalProps {
  profileName: string;
  profileId: string;
  platform: ProfilePlatform;
  onCancel: () => void;
  onDelete: (profileId: string) => void;
  isDeleting: boolean;
}

const baseClass = "delete-profile-modal";

const generateMessageSuffix = (isPremiumTier?: boolean, teamId?: number) => {
  if (!isPremiumTier) {
    return "";
  }
  return teamId ? "assigned to this fleet" : "that are unassigned";
};

const DeleteProfileModal = ({
  profileName,
  profileId,
  platform,
  onCancel,
  onDelete,
  isDeleting,
}: DeleteProfileModalProps) => {
  const { isPremiumTier, currentTeam } = useContext(AppContext);

  const messageSuffix = generateMessageSuffix(isPremiumTier, currentTeam?.id);
  const isWindows = platform === "windows";

  return (
    <Modal
      className={baseClass}
      title="Delete configuration profile"
      onExit={onCancel}
      // Modal's Enter handler runs wherever focus is, so it would delete instead
      // of opening the Windows "Learn more" link.
      onEnter={isWindows ? undefined : () => onDelete(profileId)}
      width="large"
    >
      <>
        <div className={`${baseClass}__content`}>
          <p>
            This action will remove the <b>{profileName}</b> configuration
            profile from all hosts {messageSuffix}.
          </p>
          <p>Pending profiles will be canceled.</p>
          {isWindows && (
            <p>
              Some Windows settings may not be cleanly removed from hosts.{" "}
              <CustomLink
                text="Learn more"
                newTab
                url={`${LEARN_MORE_ABOUT_BASE_LINK}/removal-behavior`}
              />
            </p>
          )}
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
          <Button onClick={onCancel} variant="secondary">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeleteProfileModal;
