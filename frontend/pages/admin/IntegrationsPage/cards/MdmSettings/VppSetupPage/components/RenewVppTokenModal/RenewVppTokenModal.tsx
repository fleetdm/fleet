import React, { useContext, useState } from "react";

import mdmAppleAPI from "services/entities/mdm_apple";
import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import FileUploader from "components/FileUploader";
import { FileDetails } from "components/FileUploader/FileUploader";

import VppSetupSteps from "../../../VppPage/components/VppSetupSteps";

const baseClass = "renew-vpp-token-modal";

interface IRenewVppTokenModalProps {
  onExit: () => void;
  onTokenRenewed: () => void;
}

const RenewVppTokenModal = ({
  onExit,
  onTokenRenewed,
}: IRenewVppTokenModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isRenewing, setIsRenewing] = useState(false);
  const [tokenFile, setTokenFile] = useState<File | null>(null);

  const onSelectFile = (files: FileList | null) => {
    const file = files?.[0];
    if (file) {
      setTokenFile(file);
    }
  };

  const onRenewToken = async () => {
    setIsRenewing(true);

    if (!tokenFile) {
      setIsRenewing(false);
      renderFlash("error", "No token selected.");
      return;
    }

    try {
      await mdmAppleAPI.uploadVppToken(tokenFile);
      renderFlash(
        "success",
        "Volume Purchasing Program (VPP) integration enabled successfully."
      );
      onTokenRenewed();
    } catch (e) {
      const msg = getErrorReason(e, { reasonIncludes: "valid token" });
      if (msg) {
        renderFlash("error", msg);
      } else {
        renderFlash("error", "Couldn't Upload. Please try again.");
      }
    }
    onExit();
    setIsRenewing(false);
  };

  return (
    <Modal title="Renew token" className={baseClass} onExit={onExit}>
      <>
        <VppSetupSteps />
        <FileUploader
          className={`${baseClass}__file-uploader`}
          accept=".vpptoken"
          message="Content token (.vpptoken)"
          graphicName="file-vpp"
          buttonType="link"
          buttonMessage="Upload"
          filePreview={
            tokenFile && (
              <FileDetails
                details={{ name: tokenFile.name }}
                graphicName="file-vpp"
              />
            )
          }
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

export default RenewVppTokenModal;
