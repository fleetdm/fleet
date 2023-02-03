import React from "react";

import { IEnrollSecret } from "interfaces/enroll_secret";

import Modal from "components/Modal";
import CustomLink from "components/CustomLink";
import Button from "components/buttons/Button";
import InputFieldHiddenContent from "components/forms/fields/InputFieldHiddenContent";

const baseClass = "disk-encryption-key-modal";

interface IDiskEncryptionKeyModal {
  secret: IEnrollSecret;
  onCancel: () => void;
}

const DiskEncryptionKeyModal = ({
  secret,
  onCancel,
}: IDiskEncryptionKeyModal) => {
  return (
    <Modal title="Disk encryption key" onExit={onCancel} className={baseClass}>
      <>
        <InputFieldHiddenContent value={"test-secret-key"} />
        <p>
          The disk encryption key refers to the FileVault recovery key for
          macOS.
        </p>
        <p>
          Use this key to log in to the host if you forgot the password.{" "}
          <CustomLink
            text="View recovery instructions"
            url="https://fleetdm.com/docs/using-fleet/mobile-device-management#unlock-a-device-using-the-disk-encryption-key"
            newTab
          />
        </p>
        <div className="modal-cta-wrap">
          <Button onClick={onCancel}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default DiskEncryptionKeyModal;
