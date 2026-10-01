import classnames from "classnames";
import React, { useState } from "react";
import { useQuery, useQueryClient } from "react-query";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import InputFieldHiddenContent from "components/forms/fields/InputFieldHiddenContent";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import TooltipWrapper from "components/TooltipWrapper";
import { getErrorReason } from "interfaces/errors";
import { IHostEncrpytionKeyResponse } from "interfaces/host";
import { HostPlatform } from "interfaces/platform";
import hostAPI from "services/entities/hosts";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

const baseClass = "disk-encryption-key-modal";

export const ROTATION_PENDING_TOOLTIP =
  "The key will rotate once the host acknowledges the request.";
export const ESCROW_OFF_TOOLTIP =
  "Escrow is not turned on for this host's fleet.";

interface IDiskEncryptionKeyModal {
  platform: HostPlatform;
  hostId: number;
  /** The viewer may rotate this host's key, and the host has a verified key and isn't personally enrolled. */
  canRotateKey: boolean;
  isEscrowEnabled: boolean;
  onCancel: () => void;
}

const DiskEncryptionKeyModal = ({
  platform,
  hostId,
  canRotateKey,
  isEscrowEnabled,
  onCancel,
}: IDiskEncryptionKeyModal) => {
  const [isRotating, setIsRotating] = useState(false);
  const queryClient = useQueryClient();
  const queryKey = ["hostEncrpytionKey", hostId];

  // Every fetch logs a "viewed disk encryption key" activity, so the key is not
  // refetched on reopen; a rotation started here is written into the cache instead.
  const { data: encryptionKey, error: encryptionKeyError } = useQuery<
    IHostEncrpytionKeyResponse,
    unknown,
    IHostEncrpytionKeyResponse["encryption_key"]
  >(queryKey, () => hostAPI.getEncryptionKey(hostId), {
    refetchOnMount: false,
    refetchOnReconnect: false,
    refetchOnWindowFocus: false,
    retry: false,
    select: (data) => data.encryption_key,
  });

  const markRotationPending = () =>
    queryClient.setQueryData<IHostEncrpytionKeyResponse | undefined>(
      queryKey,
      (prev) =>
        prev && {
          ...prev,
          encryption_key: { ...prev.encryption_key, rotation_pending: true },
        }
    );

  const onRotateKey = async () => {
    setIsRotating(true);
    try {
      await hostAPI.rotateDiskEncryptionKey(hostId);
      markRotationPending();
      notify.success(
        "Successfully sent request to rotate disk encryption key."
      );
      onCancel();
    } catch (e) {
      const reason = getErrorReason(e);
      let msg =
        "Couldn't send request to rotate disk encryption key. Please try again.";
      if (reason.includes("already in progress")) {
        msg =
          "Disk encryption key rotation is already in progress for this host.";
        markRotationPending();
      } else if (reason.includes("not decryptable")) {
        msg =
          "Couldn't rotate disk encryption key. The current key is not decryptable.";
      }
      notify.error(msg, { response: e });
    }
    setIsRotating(false);
  };

  const renderRotateButton = () => {
    // hidden until the key loads, so a rotation can't race the initial fetch
    if (platform !== "darwin" || !canRotateKey || !encryptionKey) {
      return null;
    }

    const isPending = isRotating || !!encryptionKey?.rotation_pending;
    let tipContent: string | undefined;
    if (isPending) {
      tipContent = ROTATION_PENDING_TOOLTIP;
    } else if (!isEscrowEnabled) {
      tipContent = ESCROW_OFF_TOOLTIP;
    }

    return (
      <TooltipWrapper
        tipContent={tipContent}
        disableTooltip={!tipContent}
        underline={false}
        position="top"
        showArrow
      >
        <Button
          variant="secondary"
          onClick={onRotateKey}
          disabled={isPending || !isEscrowEnabled}
          className={classnames(`${baseClass}__rotate-button`, {
            [`${baseClass}__rotating`]: isPending,
          })}
          icon="refresh"
        >
          {isPending ? "Rotating..." : "Rotate key"}
        </Button>
      </TooltipWrapper>
    );
  };

  const recoveryText =
    platform === "darwin"
      ? "Use this key to log in to the host if you forgot the password."
      : "Use this key to unlock the encrypted drive.";

  return (
    <Modal title="Disk encryption key" onExit={onCancel} className={baseClass}>
      {encryptionKeyError ? (
        <DataError
          description={getErrorReason(encryptionKeyError) || undefined}
        />
      ) : (
        <>
          <InputFieldHiddenContent value={encryptionKey?.key ?? ""} />
          <p>
            {recoveryText}{" "}
            <CustomLink
              newTab
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/disk-encryption-key`}
              text="Learn more"
            />
          </p>
          <div className="modal-cta-wrap">
            <Button onClick={onCancel}>Close</Button>
            {renderRotateButton()}
          </div>
        </>
      )}
    </Modal>
  );
};

export default DiskEncryptionKeyModal;
