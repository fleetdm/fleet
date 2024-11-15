import React from "react";
import { useQuery } from "react-query";

import { IHostEncrpytionKeyResponse } from "interfaces/host";
import hostAPI from "services/entities/hosts";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InputFieldHiddenContent from "components/forms/fields/InputFieldHiddenContent";
import DataError from "components/DataError";

const baseClass = "disk-encryption-key-modal";

interface IDiskEncryptionKeyModal {
  platform: string;
  hostId: number;
  onCancel: () => void;
}

const DiskEncryptionKeyModal = ({
  platform,
  hostId,
  onCancel,
}: IDiskEncryptionKeyModal) => {
  const { data: encryptionKey, error: encryptionKeyError } = useQuery<
    IHostEncrpytionKeyResponse,
    unknown,
    string
  >("hostEncrpytionKey", () => hostAPI.getEncryptionKey(hostId), {
    refetchOnMount: false,
    refetchOnReconnect: false,
    refetchOnWindowFocus: false,
    retry: false,
    select: (data) => data.encryption_key.key,
  });

  let descriptionText = null;
  let recoveryText = "Use this key to unlock the encrypted drive.";
  if (platform === "darwin") {
    [descriptionText, recoveryText] = [
      "The disk encryption key refers to the FileVault recovery key for macOS.",
      "Use this key to log in to the host if you forgot the password.",
    ];
  } else if (platform === "windows") {
    recoveryText =
      "The disk encryption key refers to the BitLocker recovery key for Windows.";
  }

  return (
    <Modal
      title="Disk encryption key"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      {encryptionKeyError ? (
        <DataError />
      ) : (
        <>
          <InputFieldHiddenContent value={encryptionKey ?? ""} />
          <p>{descriptionText}</p>
          <p>{recoveryText} </p>
          <div className="modal-cta-wrap">
            <Button onClick={onCancel}>Done</Button>
          </div>
        </>
      )}
    </Modal>
  );
};

export default DiskEncryptionKeyModal;
