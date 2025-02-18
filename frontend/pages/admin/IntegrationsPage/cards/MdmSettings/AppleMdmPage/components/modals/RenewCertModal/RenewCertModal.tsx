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
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    (e: unknown) => {
      const msg = getErrorReason(e);

      if (msg.toLowerCase().includes("email address is not valid")) {
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
        <ol className={`${baseClass}__setup-instructions-list`}>
          <li>
            <p>
              1. Download a certificate signing request (CSR) for Apple Push
              Notification service (APNs).
            </p>
            <DownloadCSR baseClass={baseClass} onError={onDownloadError} />
          </li>
          <li>
            <p>
              2. Sign in to{" "}
              <CustomLink
                url="https://identity.apple.com/pushcert/"
                text="Apple Push Certificates Portal"
                newTab
              />
            </p>
          </li>
          <li>
            <p>
              3. In Apple Push Certificates Portal, select <b>Renew</b> next to
              your certificate (make sure that the certificate&apos;s{" "}
              <b>Common Name (CN)</b> matches the one presented in Fleet).
            </p>
          </li>
          <li>
            <p>4. Upload your CSR and download new APNs certificate.</p>
          </li>
          <li>
            <p>
              5. Upload APNs certificate (.pem file) below.
              <FileUploader
                className={`${baseClass}__file-uploader`}
                accept=".pem"
                buttonMessage="Choose file"
                buttonType="link"
                graphicName="file-pem"
                message="APNs certificate (.pem)"
                onFileUpload={onSelectFile}
                fileDetails={certFile ? { name: certFile.name } : undefined}
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
