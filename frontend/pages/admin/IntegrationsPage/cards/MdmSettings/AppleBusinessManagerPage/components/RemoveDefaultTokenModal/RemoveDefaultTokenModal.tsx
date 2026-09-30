import React from "react";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";

import configureAppleBusinessServices from "../../../../../../../../../assets/images/default-for-sign-in-modal.png";

const baseClass = "remove-default-token-modal";

interface IRemoveDefaultTokenModalProps {
  onExit: () => void;
  onRemoveDefault: () => void;
}

const RemoveDefaultTokenModal = ({
  onExit,
  onRemoveDefault,
}: IRemoveDefaultTokenModalProps) => {
  return (
    <Modal onExit={onExit} title="Remove default for sign-in">
      <p>
        If Allow Managed Apple Account on is set, hosts that manually enroll
        won&apos;t be able to sign in to Apple Services (e.g. iCloud) with
        Managed Apple Accounts.
      </p>
      <img
        className={`${baseClass}__image`}
        src={configureAppleBusinessServices}
        alt="Apple Business - Access Management - Services"
      />
      <p>
        <CustomLink
          newTab
          url="https://business.apple.com/main/preferences/accessmanagement/services"
          text="Configure Apple Services"
        />
      </p>
      <ModalFooter
        primaryButtons={
          <>
            <Button onClick={onExit} variant="secondary">
              Cancel
            </Button>
            <Button onClick={onRemoveDefault} variant="alert">
              Remove
            </Button>
          </>
        }
      />
    </Modal>
  );
};

export default RemoveDefaultTokenModal;
