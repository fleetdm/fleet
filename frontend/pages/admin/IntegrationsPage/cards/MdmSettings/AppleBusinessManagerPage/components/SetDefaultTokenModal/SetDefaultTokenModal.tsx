import React from "react";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";

import configureAppleBusinessServices from "../../../../../../../../../assets/images/default-for-sign-in-modal.png";

const baseClass = "set-default-token-modal";

interface ISetDefaultTokenModalProps {
  onExit: () => void;
  onSetDefault: () => void;
}

const SetDefaultTokenModal = ({
  onExit,
  onSetDefault,
}: ISetDefaultTokenModalProps) => {
  return (
    <Modal onExit={onExit} title="Set as default for sign-in">
      <p>
        To restrict Managed Apple Account sign-in to managed devices, you also
        need to turn on the restriction in Apple Business. Fleet can&apos;t turn
        it on for you.
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
            <Button onClick={onSetDefault}>Set as default</Button>
          </>
        }
      />
    </Modal>
  );
};

export default SetDefaultTokenModal;
