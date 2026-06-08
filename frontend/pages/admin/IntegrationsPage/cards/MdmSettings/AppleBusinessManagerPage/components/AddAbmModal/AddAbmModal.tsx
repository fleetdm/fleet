import React, { useCallback, useContext, useState } from "react";

import { NotificationContext } from "context/notification";
import mdmAbmAPI from "services/entities/mdm_apple_bm";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import FileUploader from "components/FileUploader";
import CustomLink from "components/CustomLink";
import DownloadABMKey from "pages/admin/components/DownloadFileButtons/DownloadABMKey";
import { getErrorMessage } from "./helpers";

const baseClass = "add-abm-modal";

interface IAddAbmModalProps {
  onCancel: () => void;
  onAdded: () => void;
}

const AddAbmModal = ({ onCancel, onAdded }: IAddAbmModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const [tokenFile, setTokenFile] = useState<File | null>(null);
  const [isUploading, setIsUploading] = useState(false);

  const onSelectFile = useCallback((files: FileList | null) => {
    const file = files?.[0];
    if (file) {
      setTokenFile(file);
    }
  }, []);

  const uploadAbmToken = useCallback(async () => {
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
      renderFlash("error", getErrorMessage(e));
      onCancel();
    } finally {
      setIsUploading(false);
    }
  }, [tokenFile, renderFlash, onAdded, onCancel]);

  return (
    <Modal className={baseClass} title="Add AB" onExit={onCancel} width="large">
      <p>
        Follow the step-by-step guide to connect Fleet to Apple Business.{" "}
        <CustomLink
          url="https://fleetdm.com/learn-more-about/setup-abm"
          text="Learn how"
          newTab
        />
      </p>
      <FileUploader
        className={`${baseClass}__file-uploader ${
          isUploading ? `${baseClass}__file-uploader--loading` : ""
        }`}
        accept=".p7m"
        message="AB token (.p7m)"
        graphicName="file-p7m"
        buttonType="brand-inverse-icon"
        buttonMessage={isUploading ? "Uploading..." : "Upload"}
        fileDetails={tokenFile ? { name: tokenFile.name } : undefined}
        onFileUpload={onSelectFile}
      />
      <div className="modal-cta-wrap">
        <Button
          onClick={uploadAbmToken}
          isLoading={isUploading}
          disabled={!tokenFile || isUploading}
        >
          Add AB
        </Button>
        <DownloadABMKey baseClass={baseClass} />
      </div>
    </Modal>
  );
};

export default AddAbmModal;
