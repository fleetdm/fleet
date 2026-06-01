import React, { useState, useContext, useCallback } from "react";

import { NotificationContext } from "context/notification";

import mdmAppleApi from "services/entities/mdm_apple";
import { getErrorReason } from "interfaces/errors";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import { FileUploader } from "components/FileUploader/FileUploader";
import Modal from "components/Modal";
import DownloadCSR from "../../../../../../../components/DownloadFileButtons/DownloadCSR";

const baseClass = "modal renew-cert-modal";

interface IRenewCertModalProps {
  onCancel: () => void;
  onRenew: () => void;
}

const RenewCertModal = ({
  onCancel,
  onRenew,
}: IRenewCertModalProps): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);

  const [isUploading, setIsUploading] = useState(false);
  const [certFile, setCertFile] = useState<File | null>(null);

  const onSelectFile = useCallback((files: FileList | null) => {
    const file = files?.[0];
    if (file) {
      setCertFile(file);
    }
  }, []);

  const onRenewClick = useCallback(async () => {
    if (!certFile) {
      // this shouldn't happen, but just in case
      renderFlash("error", "Please provide a certificate file.");
      return;
    }
    setIsUploading(true);
    try {
      await mdmAppleApi.uploadApplePushCertificate(certFile);
      renderFlash("success", "APNs certificate renewed successfully.");
      setIsUploading(false);
      onRenew();
    } catch (e) {
      console.error(e);
      const msg = getErrorReason(e);
      if (msg.toLowerCase().includes("valid certificate")) {
        renderFlash("error", msg);
      } else {
        renderFlash("error", "Couldn’t renew. Please try again.");
      }
      setIsUploading(false);
      onCancel();
    }
  }, [certFile, renderFlash, onCancel, onRenew]);

  const onDownloadError = useCallback(
    (e: unknown) => {
      const msg = getErrorReason(e);
      if (msg.includes("is not permitted for APNS certificate signing.")) {
        renderFlash("error", msg);
      } else {
        renderFlash("error", "Something's gone wrong. Please try again.");
      }
    },
    [renderFlash]
  );

  return (
    <Modal title="Renew certificate" onExit={onCancel} className={baseClass}>
      <div className={`${baseClass}__page-content ${baseClass}__setup-content`}>
        <p>
          Follow the step-by-step guide to renew.{" "}
          <CustomLink
            url="https://fleetdm.com/learn-more-about/renew-apns"
            text="Learn how"
            newTab
          />
        </p>
        <FileUploader
          className={`${baseClass}__file-uploader`}
          accept=".pem"
          buttonMessage="Choose file"
          buttonType="brand-inverse-icon"
          graphicName="file-pem"
          message="APNs certificate (.pem)"
          onFileUpload={onSelectFile}
          fileDetails={certFile ? { name: certFile.name } : undefined}
        />
        <div className={`${baseClass}__button-wrap`}>
          <DownloadCSR baseClass={baseClass} onError={onDownloadError} />
          <Button
            className={`${baseClass}__submit-button ${
              isUploading ? `uploading` : ""
            }`}
            disabled={!certFile || isUploading}
            isLoading={isUploading}
            type="button"
            onClick={onRenewClick}
          >
            Renew certificate
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default RenewCertModal;
