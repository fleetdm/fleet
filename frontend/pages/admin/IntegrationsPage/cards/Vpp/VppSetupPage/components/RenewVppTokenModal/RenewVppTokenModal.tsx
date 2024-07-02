import React, { useState } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import FileUploader from "components/FileUploader";
import { FileDetails } from "components/FileUploader/FileUploader";

import VppSetupSteps from "../VppSetupSteps";

const baseClass = "renew-vpp-token-modal";

interface IRenewVppTokenModalProps {
  onExit: () => void;
}

const RenewVppTokenModal = ({ onExit }: IRenewVppTokenModalProps) => {
  const [isRenewing, setIsRenewing] = useState(false);

  const [tokenFile, setTokenFile] = useState<File | null>(null);

  const onSelectFile = (files: FileList | null) => {
    const file = files?.[0];
    if (file) {
      setTokenFile(file);
    }
  };

  const onRenewToken = () => {
    // TODO: API integration
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
