import React, { useCallback, useContext, useState } from "react";
import { noop } from "lodash";

import { NotificationContext } from "context/notification";
import pkiAPI from "services/entities/pki";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import FileUploader from "components/FileUploader";
import CustomLink from "components/CustomLink";
import TooltipWrapper from "components/TooltipWrapper";

import DownloadCSR from "pages/admin/components/DownloadFileButtons/DownloadCSR";

import { getErrorMessage } from "./helpers";

const baseClass = "add-pki-modal";

interface IAddPkiModalProps {
  onCancel: () => void;
  onAdded: () => void;
}

const AddPkiModal = ({ onCancel, onAdded }: IAddPkiModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const [pkiName, setPkiName] = useState("");
  const [pkiCert, setPkiCert] = useState<File | null>(null);
  const [isUploading, setIsUploading] = useState(false);

  const onSelectFile = useCallback((files: FileList | null) => {
    const file = files?.[0];
    if (file) {
      setPkiCert(file);
    }
  }, []);

  const uploadPkiCert = useCallback(async () => {
    setIsUploading(true);
    if (!pkiCert) {
      setIsUploading(false);
      renderFlash("error", "No file selected.");
      return;
    }

    try {
      await pkiAPI.uploadCert(pkiName, pkiCert);
      renderFlash("success", "Added successfully.");
      onAdded();
    } catch (e) {
      renderFlash("error", getErrorMessage(e));
      onCancel();
    } finally {
      setIsUploading(false);
    }
  }, [pkiName, pkiCert, renderFlash, onAdded, onCancel]);

  const onInputChangeName = useCallback(
    (value: string) => {
      setPkiName(value);
    },
    [setPkiName]
  );

  return (
    <Modal
      className={baseClass}
      title="Add PKI"
      onExit={onCancel}
      width="large"
    >
      <>
        <div className={`${baseClass}__setup`}>
          <p>To help your end users connect to Wi-Fi, you can add your PKI.</p>
          <p>Fleet currently supports DigiCert PKI.</p>
          <InputField
            label="Name"
            onChange={onInputChangeName}
            name="pkiName"
            value={pkiName}
          />
          <ol className={`${baseClass}__setup-list`}>
            <li>
              <span>1.</span>
              <p>
                Download a certificate signing request (CSR) for DigiCert.
                <DownloadCSR
                  baseClass={baseClass}
                  onError={noop}
                  onSuccess={noop}
                  pkiName={pkiName}
                />
              </p>
            </li>
            <li>
              <span>2.</span>
              <span>
                <span>
                  Go to{" "}
                  <CustomLink
                    newTab
                    text="DigiCert PKI platform"
                    url="https:/fleetdm.com/sign-in-to/digicert-pki"
                  />
                  <br />
                </span>
              </span>
            </li>
            <li>
              <span>3.</span>
              <span>
                In DigiCert, select <b>Settings {">"} Get an RA certificate</b>,
                upload your CSR, and download your registration authority (RA)
                certificate.
              </span>
            </li>
            <li>
              <span>4.</span>
              <span>Upload your RA certificate (.p7b file) below.</span>
            </li>
          </ol>
        </div>
        <FileUploader
          className={`${baseClass}__file-uploader ${
            isUploading ? `${baseClass}__file-uploader--loading` : ""
          }`}
          accept=".p7b"
          message="RA certificate (.p7b)"
          graphicName={"file-crt"}
          buttonType="link"
          buttonMessage={isUploading ? "Uploading..." : "Upload"}
          fileDetails={pkiCert ? { name: pkiCert.name } : undefined}
          onFileUpload={onSelectFile}
        />
        <div className="modal-cta-wrap">
          <TooltipWrapper
            tipContent="Complete all fields to save"
            position="top"
            showArrow
            underline={false}
            tipOffset={8}
            disableTooltip={(!!pkiCert && !!pkiName) || isUploading}
          >
            <Button
              variant="brand"
              onClick={uploadPkiCert}
              isLoading={isUploading}
              disabled={!pkiCert || !pkiName || isUploading}
            >
              Save
            </Button>
          </TooltipWrapper>
          <Button variant="inverse" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default AddPkiModal;
