import React, { useState, useCallback } from "react";

import mdmAppleAPI from "services/entities/mdm_apple";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import { FileUploader } from "components/FileUploader/FileUploader";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import { getErrorMessage } from "./helpers";

const baseClass = "modal renew-vpp-modal";

interface IRenewVppModalProps {
  tokenId: number;
  onCancel: () => void;
  onRenewedToken: () => void;
}

const RenewVppModal = ({
  tokenId,
  onCancel,
  onRenewedToken,
}: IRenewVppModalProps) => {
  const [isRenewing, setIsRenewing] = useState(false);
  const [tokenFile, setTokenFile] = useState<File | null>(null);

  const onSelectFile = (files: FileList | null) => {
    const file = files?.[0];
    if (file) {
      setTokenFile(file);
    }
  };

  const onRenewToken = useCallback(async () => {
    setIsRenewing(true);

    if (!tokenFile) {
      setIsRenewing(false);
      notify.error("No token selected.");
      return;
    }

    try {
      await mdmAppleAPI.renewVppToken(tokenId, tokenFile);
      notify.success(
        "Volume Purchasing Program (VPP) integration enabled successfully."
      );
      onRenewedToken();
    } catch (e) {
      notify.error(getErrorMessage(e), { response: e });
      onCancel();
    }
    setIsRenewing(false);
  }, [onCancel, onRenewedToken, tokenFile, tokenId]);

  return (
    <Modal
      title="Renew VPP"
      onExit={onCancel}
      className={baseClass}
      isContentDisabled={isRenewing}
      width="large"
    >
      <p className={`${baseClass}__description`}>
        Follow the step-by-step guide to renew.{" "}
        <CustomLink
          url="https://fleetdm.com/learn-more-about/renew-vpp"
          text="Learn how"
          newTab
        />
      </p>
      <FileUploader
        className={`${baseClass}__file-uploader`}
        accept=".vpptoken"
        message="Content token (.vpptoken)"
        graphicName="file-vpp"
        buttonType="brand-inverse-icon"
        buttonMessage="Upload"
        fileDetails={tokenFile ? { name: tokenFile.name } : undefined}
        onFileUpload={onSelectFile}
      />
      <div className="modal-cta-wrap">
        <Button
          onClick={onRenewToken}
          isLoading={isRenewing}
          disabled={!tokenFile}
        >
          Renew token
        </Button>
      </div>
    </Modal>
  );
};

export default RenewVppModal;
