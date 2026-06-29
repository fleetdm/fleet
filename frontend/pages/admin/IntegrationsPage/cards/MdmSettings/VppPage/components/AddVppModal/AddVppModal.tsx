import React, { useCallback, useState } from "react";

import mdmAppleAPI from "services/entities/mdm_apple";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import FileUploader from "components/FileUploader";
import { notify } from "components/ToastNotification";

import { getErrorMessage } from "./helpers";

const baseClass = "add-vpp-modal";

interface IAddVppModalProps {
  onCancel: () => void;
  onAdded: () => void;
}

const AddVppModal = ({ onCancel, onAdded }: IAddVppModalProps) => {
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
      notify.error("No token selected.");
      return;
    }

    try {
      await mdmAppleAPI.uploadVppToken(tokenFile);
      notify.success("Added successfully.");
      onAdded();
    } catch (e) {
      notify.error(getErrorMessage(e), { response: e });
      onCancel();
    } finally {
      setIsUploading(false);
    }
  }, [tokenFile, onAdded, onCancel]);

  return (
    <Modal
      className={baseClass}
      title="Add VPP"
      onExit={onCancel}
      width="large"
    >
      <p className={`${baseClass}__description`}>
        Follow the step-by-step guide to add VPP.{" "}
        <CustomLink
          url="https://fleetdm.com/learn-more-about/add-vpp"
          text="Learn how"
          newTab
        />
      </p>
      <FileUploader
        className={`${baseClass}__file-uploader ${
          isUploading ? `${baseClass}__file-uploader--loading` : ""
        }`}
        accept=".vpptoken"
        message="Content token (.vpptoken)"
        graphicName="file-vpp"
        buttonType="brand-inverse-icon"
        buttonMessage={isUploading ? "Uploading..." : "Upload"}
        fileDetails={tokenFile ? { name: tokenFile.name } : undefined}
        onFileUpload={onSelectFile}
      />
      <div className="modal-cta-wrap">
        <Button
          onClick={uploadVppToken}
          isLoading={isUploading}
          disabled={!tokenFile || isUploading}
        >
          Add VPP
        </Button>
      </div>
    </Modal>
  );
};

export default AddVppModal;
