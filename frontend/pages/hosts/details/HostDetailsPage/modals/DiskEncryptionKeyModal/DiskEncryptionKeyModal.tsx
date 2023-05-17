import React from "react";
import { useQuery } from "react-query";

import { IHostEncrpytionKeyResponse } from "interfaces/host";
import hostAPI from "services/entities/hosts";

import Modal from "components/Modal";
import CustomLink from "components/CustomLink";
import Button from "components/buttons/Button";
import InputFieldHiddenContent from "components/forms/fields/InputFieldHiddenContent";
import DataError from "components/DataError";

const baseClass = "disk-encryption-key-modal";

interface IDiskEncryptionKeyModal {
  hostId: number;
  onCancel: () => void;
}

const DiskEncryptionKeyModal = ({
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

  return (
    <Modal title="Disk encryption key" onExit={onCancel} className={baseClass}>
      {encryptionKeyError ? (
        <DataError />
      ) : (
        <>
          <InputFieldHiddenContent value={encrpytionKey ?? ""} />
          <p>
            The disk encryption key refers to the FileVault recovery key for
            macOS.
          </p>
          <p>
            Use this key to log in to the host if you forgot the password.{" "}
            <CustomLink
              text="View recovery instructions"
              url="https://fleetdm.com/docs/using-fleet/mdm-disk-encryption#reset-a-macos-hosts-password-using-the-disk-encryption-key"
              newTab
            />
          </p>
          <div className="modal-cta-wrap">
            <Button onClick={onCancel}>Done</Button>
          </div>
        </>
      )}
    </Modal>
  );
};

export default DiskEncryptionKeyModal;
