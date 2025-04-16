import React, { useCallback, useContext, useState } from "react";

import { NotificationContext } from "context/notification";
import mdmAppleAPI from "services/entities/mdm_apple";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import FileUploader from "components/FileUploader";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import VppSetupSteps from "../VppSetupSteps";
import { getErrorMessage } from "./helpers";

const baseClass = "add-vpp-modal";

interface IAddVppModalProps {
  onCancel: () => void;
  onAdded: () => void;
}

const AddVppModal = ({ onCancel, onAdded }: IAddVppModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const [tokenFile, setTokenFile] = useState<File | null>(null);
  const [isUploading, setIsUploading] = useState(false);

  const onSelectFile = useCallback((files: FileList | null) => {
    const file = files?.[0];
    if (file) {
      setTokenFile(file);
    }
  }, []);

  const uploadVppToken = useCallback(async () => {
    setIsUploading(true);
    if (!tokenFile) {
      setIsUploading(false);
      renderFlash("error", "No token selected.");
      return;
    }

    try {
      await mdmAppleAPI.uploadVppToken(tokenFile);
      renderFlash("success", "Added successfully.");
      onAdded();
    } catch (e) {
      renderFlash("error", getErrorMessage(e));
      onCancel();
    } finally {
      setIsUploading(false);
    }
  }, [tokenFile, renderFlash, onAdded, onCancel]);

  return (
    <Modal
      className={baseClass}
      title="Add VPP"
      onExit={onCancel}
      width="large"
    >
      <>
        <VppSetupSteps extendendSteps />
        <FileUploader
          className={`${baseClass}__file-uploader ${
            isUploading ? `${baseClass}__file-uploader--loading` : ""
          }`}
          accept=".vpptoken"
          message="Content token (.vpptoken)"
          graphicName="file-vpp"
          buttonType="link"
          buttonMessage={isUploading ? "Uploading..." : "Upload"}
          fileDetails={tokenFile ? { name: tokenFile.name } : undefined}
          onFileUpload={onSelectFile}
          gitopsCompatible
        />
        <div className="modal-cta-wrap">
          <GitOpsModeTooltipWrapper
            tipOffset={8}
            renderChildren={(disableChildren) => (
              <Button
                onClick={uploadVppToken}
                isLoading={isUploading}
                disabled={!tokenFile || isUploading || disableChildren}
              >
                Add VPP
              </Button>
            )}
          />
        </div>
      </>
    </Modal>
  );
};

export default AddVppModal;
