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
import { RequestState } from "pages/admin/IntegrationsPage/cards/MdmSettings/MacOSMdmPage/components/actions/DownloadCSR";
// import DownloadCSR, { RequestState } from "../../actions/DownloadCSR";

const baseClass = "modal renew-token-modal";

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
              1. Sign in to{" "}
              <CustomLink
                url="https://business.apple.com/"
                text="Apple Business Manager"
                newTab
              />
            </p>
          </li>
          <li>
            <p>
              2. Select your <b>account name</b> at the bottom left of the
              screen, then select <b>Preferences</b>.
            </p>
          </li>
          <li>
            <p>
              3. In the <b>Your MDM Servers</b> section, select your Fleet
              server, then select <b>Download Token</b> at the top.
            </p>
          </li>
          <li>
            <p>
              5. Upload the downloaded token (.p7m file) below.
              <FileUploader
                className={`${baseClass}__file-uploader`}
                accept=".p7m"
                buttonMessage="Choose file"
                buttonType="link"
                graphicName="file-p7m"
                message="ABM token (.p7m)"
                onFileUpload={onSelectFile}
                filePreview={
                  pemFile && (
                    <FileDetails
                      details={{ name: pemFile.name }}
                      graphicName="file-p7m"
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
