import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";

import mdmAppleAPI from "services/entities/mdm_apple";
import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";

import PATHS from "router/paths";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import FileUploader from "components/FileUploader";
import { FileDetails } from "components/FileUploader/FileUploader";

import VppSetupSteps from "../VppSetupSteps";

const baseClass = "renew-vpp-token-modal";

interface IRenewVppTokenModalProps {
  onExit: () => void;
  router: InjectedRouter;
}

const RenewVppTokenModal = ({ onExit, router }: IRenewVppTokenModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isUploading, setIsUploading] = useState(false);

  const [tokenFile, setTokenFile] = useState<File | null>(null);

  const onSelectFile = (files: FileList | null) => {
    const file = files?.[0];
    if (file) {
      setTokenFile(file);
    }
  };

  const onRenewToken = async (files: FileList | null) => {
    setIsUploading(true);

    const token = files?.[0];
    if (!token) {
      setIsUploading(false);
      renderFlash("error", "No token selected.");
      return;
    }

    try {
      await mdmAppleAPI.uploadVppToken(token);
      renderFlash(
        "success",
        "Volume Purchasing Program (VPP) integration enabled successfully."
      );
      router.push(PATHS.ADMIN_INTEGRATIONS_VPP);
    } catch (e) {
      const msg = getErrorReason(e, { reasonIncludes: "valid token" });
      if (msg) {
        renderFlash("error", msg);
      } else {
        renderFlash("error", "Couldn't Upload. Please try again.");
      }
    }
    onExit();
    setIsUploading(false);
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
          buttonMessage={isUploading ? "Uploading..." : "Upload"}
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
            isLoading={isUploading}
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
