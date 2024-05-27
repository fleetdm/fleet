import React, { useState, useContext, useEffect } from "react";

import { NotificationContext } from "context/notification";

import MdmAPI from "services/entities/mdm";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import {
  FileUploader,
  FileDetails,
} from "components/FileUploader/FileUploader";
import Modal from "components/Modal";
import DownloadCSR, { RequestState } from "../../actions/DownloadCSR";

const baseClass = "modal renew-cert-modal";

interface IRenewCertModalProps {
  onCancel: () => void;
}

const RenewCertModal = ({ onCancel }: IRenewCertModalProps): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);

  const [uploadState, setUploadState] = useState<RequestState>(undefined);
  const [pemFile, setPemFile] = useState<File | null>(null);

  const onSelectFile = (files: FileList | null) => {
    const file = files?.[0];
    if (file) {
      setPemFile(file);
    }
    console.log("File uploaded", files?.length);
  };

  const onRenewCertClick = async () => {
    if (!pemFile) {
      return;
    }

    setUploadState("loading");

    try {
      // const formData = new FormData();
      // formData.append("cert", pemFile);

      // await MdmAPI.uploadAPNsCert(formData);
      // setRequestState("success");
      console.log("Renew cert clicked");
    } catch (e) {
      console.error(e);
      // TODO: error handling per Figma
      setUploadState("error");
    }
  };

  useEffect(() => {
    if (uploadState === "loading") {
      setTimeout(() => {
        setUploadState("error");
      }, 5000);
    }
    if (uploadState === "success") {
      renderFlash("success", "Certificate renewed successfully");
      onCancel();
    }
    if (uploadState === "error") {
      renderFlash("error", "Failed to renew certificate");
      onCancel();
    }
  }, [uploadState, onCancel, renderFlash]);

  return (
    <Modal title="Renew certificate" onExit={onCancel} className={baseClass}>
      <div className={`${baseClass}__page-content ${baseClass}__setup-content`}>
        <ol className={`${baseClass}__setup-instructions-list`}>
          <li>
            <p>
              1. Download a certificate signing request (CSR) for Apple Push
              Notification service (APNs).
            </p>
            <DownloadCSR baseClass={baseClass} />
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
                // accept=".pem" // TODO: uncomment this
                buttonMessage="Choose file"
                buttonType="link"
                graphicName="file-pem"
                message="APNs certificate (.pem)"
                onFileUpload={onSelectFile}
                filePreview={
                  pemFile && (
                    <FileDetails
                      details={{ name: pemFile.name }}
                      graphicName="file-pem"
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
              uploadState === "loading" ? `uploading` : ""
            }`}
            variant="brand"
            disabled={!pemFile || uploadState === "loading"}
            isLoading={uploadState === "loading"}
            type="button"
            onClick={onRenewCertClick}
          >
            Renew certificate
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default RenewCertModal;
