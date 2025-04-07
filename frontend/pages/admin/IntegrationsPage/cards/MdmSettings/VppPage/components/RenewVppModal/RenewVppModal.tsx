import React, { useState, useContext, useCallback } from "react";

import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";
import mdmAppleAPI from "services/entities/mdm_apple";

import Button from "components/buttons/Button";
import { FileUploader } from "components/FileUploader/FileUploader";
import Modal from "components/Modal";
import VppSetupSteps from "../VppSetupSteps";
import { getErrorMessage } from "./helpers";

const baseClass = "modal renew-vpp-modal";

interface IRenewVppModalProps {
  tokenId: number;
  orgName: string;
  onCancel: () => void;
  onRenewedToken: () => void;
}

const RenewVppModal = ({
  tokenId,
  orgName,
  onCancel,
  onRenewedToken,
}: IRenewVppModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
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
      renderFlash("error", "No token selected.");
      return;
    }

    try {
      await mdmAppleAPI.renewVppToken(tokenId, tokenFile);
      renderFlash(
        "success",
        "Volume Purchasing Program (VPP) integration enabled successfully."
      );
      onRenewedToken();
    } catch (e) {
      renderFlash("error", getErrorMessage(e));
      onCancel();
    }
    setIsRenewing(false);
  }, [onCancel, onRenewedToken, renderFlash, tokenFile, tokenId]);

  return (
    <Modal
      title="Renew VPP"
      onExit={onCancel}
      className={baseClass}
      isContentDisabled={isRenewing}
      width="large"
    >
      <>
        <p className={`${baseClass}__description`}>
          Renew Volume Purchasing Program for <b>{orgName}</b> location.
        </p>
        <VppSetupSteps />
        <FileUploader
          className={`${baseClass}__file-uploader`}
          accept=".vpptoken"
          message="Content token (.vpptoken)"
          graphicName="file-vpp"
          buttonType="link"
          buttonMessage="Upload"
          fileDetails={tokenFile ? { name: tokenFile.name } : undefined}
          onFileUpload={onSelectFile}
        />
        <div className="modal-cta-wrap">
          <Button
            variant="brand"
            onClick={onRenewToken}
            isLoading={isRenewing}
            disabled={!tokenFile}
          >
            Renew token
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default RenewVppModal;
