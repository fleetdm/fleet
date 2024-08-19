import React, { useState, useContext, useCallback } from "react";

import { NotificationContext } from "context/notification";

import { getErrorReason } from "interfaces/errors";

import mdmAppleBmAPI from "services/entities/mdm_apple_bm";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import {
  FileUploader,
  FileDetails,
} from "components/FileUploader/FileUploader";
import Modal from "components/Modal";

const baseClass = "modal renew-token-modal";

interface IRenewCertModalProps {
  tokenId: number;
  onCancel: () => void;
  onRenew: () => void;
}

const RenewCertModal = ({
  tokenId,
  onCancel,
  onRenew,
}: IRenewCertModalProps): JSX.Element => {
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
      // this shouldn'r happen, but just in case
      renderFlash("error", "Please provide a token file.");
      return;
    }
    setIsUploading(true);
    try {
      await mdmAppleBmAPI.renewToken(tokenId, tokenFile);
      renderFlash("success", "Renewed successfully.");
      setIsUploading(false);
      onRenew();
    } catch (e) {
      // TODO: figure out what error message will be sent from API.
      const msg = getErrorReason(e);
      if (msg.toLowerCase().includes("valid token")) {
        renderFlash("error", msg);
      } else {
        renderFlash("error", "Couldn’t renew. Please try again.");
      }
      setIsUploading(false);
    }
  }, [tokenFile, renderFlash, tokenId, onRenew]);

  return (
    <Modal title="Renew token" onExit={onCancel} className={baseClass}>
      <div className={`${baseClass}__page-content ${baseClass}__setup-content`}>
        <ol className={`${baseClass}__setup-instructions-list`}>
          <li>
            <p>
              1. Sign in to{" "}
              <CustomLink
                url="https://business.apple.com/"
                text="Apple Business Manager"
                newTab
              />
            </p>
          </li>
          <li>
            <p>
              2. Select your <b>account name</b> at the bottom left of the
              screen, then select <b>Preferences</b>.
            </p>
          </li>
          <li>
            <p>
              3. In the <b>Your MDM Servers</b> section, select your Fleet
              server, then select <b>Download Token</b> at the top.
            </p>
          </li>
          <li>
            <p>
              4. Upload the downloaded token (.p7m file) below.
              <FileUploader
                className={`${baseClass}__file-uploader`}
                accept=".p7m"
                buttonMessage="Choose file"
                buttonType="link"
                graphicName="file-p7m"
                message="ABM token (.p7m)"
                onFileUpload={onSelectFile}
                filePreview={
                  tokenFile && (
                    <FileDetails
                      details={{ name: tokenFile.name }}
                      graphicName="file-p7m"
                    />
                  )
                }
              />
            </p>
          </li>
        </ol>
        <div className={`${baseClass}__button-wrap`}>
          <Button
            className={`${baseClass}__submit-button ${
              isUploading ? `uploading` : ""
            }`}
            variant="brand"
            disabled={!tokenFile || isUploading}
            isLoading={isUploading}
            type="button"
            onClick={onRenewToken}
          >
            Renew ABM
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default RenewCertModal;
