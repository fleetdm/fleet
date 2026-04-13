import React, { useState, useContext, useCallback } from "react";

import { NotificationContext } from "context/notification";
import mdmAppleBmAPI from "services/entities/mdm_apple_bm";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import { FileUploader } from "components/FileUploader/FileUploader";
import Modal from "components/Modal";

import { getErrorMessage } from "./helpers";

const baseClass = "renew-abm-modal";

interface IRenewAbmModalProps {
  tokenId: number;
  orgName: string;
  onCancel: () => void;
  onRenewedToken: () => void;
}

const RenewAbmModal = ({
  tokenId,
  orgName,
  onCancel,
  onRenewedToken,
}: IRenewAbmModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const [isUploading, setIsUploading] = useState(false);
  const [tokenFile, setTokenFile] = useState<File | null>(null);

  const onSelectFile = useCallback((files: FileList | null) => {
    const file = files?.[0];
    if (file) {
      setTokenFile(file);
    }
  }, []);

  const onRenewToken = useCallback(async () => {
    if (!tokenFile) {
      // this shouldn't happen, but just in case
      renderFlash("error", "Please provide a token file.");
      return;
    }
    setIsUploading(true);
    try {
      await mdmAppleBmAPI.renewToken(tokenId, tokenFile);
      renderFlash("success", "Renewed successfully.");
      setIsUploading(false);
      onRenewedToken();
    } catch (e) {
      renderFlash("error", getErrorMessage(e));
      onCancel();
      setIsUploading(false);
    }
  }, [tokenFile, renderFlash, tokenId, onRenewedToken, onCancel]);

  return (
    <Modal
      title="Renew token"
      onExit={onCancel}
      className={baseClass}
      isContentDisabled={isUploading}
      width="large"
    >
      <div className={`${baseClass}__page-content ${baseClass}__setup-content`}>
        <p className={`${baseClass}__description`}>
          Renew Apple Business for <b>{orgName}</b> by uploading your AB token.{" "}
          <CustomLink
            newTab
            text="Learn more"
            url="https://fleetdm.com/guides/apple-mdm-setup#to-renew-an-abm-token"
          />
        </p>
        <ul className={`${baseClass}__setup-instructions-list`}>
          <li>
            <p>
              Upload the downloaded token (.p7m file) below.
              <FileUploader
                className={`${baseClass}__file-uploader`}
                accept=".p7m"
                buttonMessage="Choose file"
                buttonType="brand-inverse-icon"
                graphicName="file-p7m"
                message="AB token (.p7m)"
                onFileUpload={onSelectFile}
                fileDetails={tokenFile ? { name: tokenFile.name } : undefined}
              />
            </p>
          </li>
        </ul>
        <div className="modal-cta-wrap">
          <Button
            className={`${baseClass}__submit-button ${
              isUploading ? `uploading` : ""
            }`}
            disabled={!tokenFile || isUploading}
            isLoading={isUploading}
            type="button"
            onClick={onRenewToken}
          >
            Renew AB
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default RenewAbmModal;
