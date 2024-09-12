import React from "react";
import { useQuery } from "react-query";

import { IHostEncrpytionKeyResponse } from "interfaces/host";
import hostAPI from "services/entities/hosts";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InputFieldHiddenContent from "components/forms/fields/InputFieldHiddenContent";
import DataError from "components/DataError";
import { QueryablePlatform } from "interfaces/platform";

const baseClass = "disk-encryption-key-modal";

// currently these are the only supported platforms for the disk encryption
// key modal.
export type ModalSupportedPlatform = Extract<
  QueryablePlatform,
  "darwin" | "windows"
>;

// Checks to see if the platform is supported by the modal.
export const isSupportedPlatform = (
  platform: string
): platform is ModalSupportedPlatform => {
  return ["darwin", "windows"].includes(platform);
};

interface IDiskEncryptionKeyModal {
  platform: ModalSupportedPlatform;
  hostId: number;
  onCancel: () => void;
}

const DiskEncryptionKeyModal = ({
  platform,
  hostId,
  onCancel,
}: IDiskEncryptionKeyModal) => {
  const { data: encrpytionKey, error: encryptionKeyError } = useQuery<
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

  const isMacOS = platform === "darwin";
  const descriptionText = isMacOS
    ? "The disk encryption key refers to the FileVault recovery key for macOS."
    : "The disk encryption key refers to the BitLocker recovery key for Windows.";

  const recoveryText = isMacOS
    ? "Use this key to log in to the host if you forgot the password."
    : "Use this key to unlock the encrypted drive.";

  return (
    <Modal title="Disk encryption key" onExit={onCancel} className={baseClass}>
      {encryptionKeyError ? (
        <DataError />
      ) : (
        <>
          <InputFieldHiddenContent value={encrpytionKey ?? ""} />
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
