import React from "react";
import { useQuery } from "react-query";

import { IHostEncrpytionKeyResponse } from "interfaces/host";
import hostAPI from "services/entities/hosts";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InputFieldHiddenContent from "components/forms/fields/InputFieldHiddenContent";
import DataError from "components/DataError";
import CustomLink from "components/CustomLink";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { HostPlatform } from "interfaces/platform";

const baseClass = "disk-encryption-key-modal";

interface IDiskEncryptionKeyModal {
  platform: HostPlatform;
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

  const recoveryText =
    platform === "darwin"
      ? "Use this key to log in to the host if you forgot the password."
      : "Use this key to unlock the encrypted drive.";

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
          <p>
            {recoveryText}{" "}
            <CustomLink
              newTab
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/mdm-disk-encryption`}
              text="Learn more"
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
