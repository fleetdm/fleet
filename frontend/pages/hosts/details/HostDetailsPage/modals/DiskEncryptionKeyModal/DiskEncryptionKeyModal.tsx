import React from "react";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";
import CustomLink from "components/CustomLink";
import Button from "components/buttons/Button";
import { noop } from "lodash";

const baseClass = "disk-encryption-key-modal";

interface IDiskEncryptionKeyModal {
  onCancel: () => void;
}

const DiskEncryptionKeyModal = ({ onCancel }: IDiskEncryptionKeyModal) => {
  return (
    <Modal title="Disk encryption key" onExit={onCancel} className={baseClass}>
      <>
        <InputField
          inputWrapperClass={`${baseClass}__secret-input`}
          name="osqueryd-secret"
          label={"Secret"}
          type={"text"}
          value={"test"}
          onChange={noop}
          hint={"Must contain at least 32 characters."}
        />
        <p>
          The disk encryption key refers to the FileVault recovery key for
          macOS.
        </p>
        <p>
          Use this key to log in to the host if you forgot the password.{" "}
          <CustomLink text="View recovery instructions" url="test" newTab />
        </p>
        <div className="modal-cta-wrap">
          <Button onClick={onCancel}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default DiskEncryptionKeyModal;
