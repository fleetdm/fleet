import React, { useCallback, useContext, useState } from "react";

import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";
import mdmAbmAPI from "services/entities/mdm_apple_bm";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import FileUploader from "components/FileUploader";
import { FileDetails } from "components/FileUploader/FileUploader";

import VppSetupSteps from "./VppSetupSteps";

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
      await mdmAbmAPI.uploadToken(tokenFile);
      renderFlash("success", "Added successfully.");
      onAdded();
    } catch (e) {
      // TODO: ensure API is sending back the correct err messages
      const msg = getErrorReason(e);
      renderFlash("error", msg);
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
          filePreview={
            tokenFile && (
              <FileDetails
                details={{ name: tokenFile.name }}
                graphicName="file-p7m"
              />
            )
          }
          onFileUpload={onSelectFile}
        />
        <div className="modal-cta-wrap">
          <Button
            variant="brand"
            onClick={uploadVppToken}
            isLoading={isUploading}
            disabled={!tokenFile || isUploading}
          >
            Add VPP
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default AddVppModal;
