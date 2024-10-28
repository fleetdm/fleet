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
    <Modal
      className={baseClass}
      title="Add ABM"
      onExit={onCancel}
      width="large"
    >
      <>
        <ol className={`${baseClass}__setup-list`}>
          <li>
            <span>1.</span>
            <p>
              Download your public key. <DownloadABMKey baseClass={baseClass} />
            </p>
          </li>
          <li>
            <span>2.</span>
            <span>
              <span>
                Sign in to{" "}
                <CustomLink
                  newTab
                  text="Apple Business Manager"
                  url="https://business.apple.com"
                />
                <br />
                If your organization doesn&apos;t have an account, select{" "}
                <b>Enroll now</b>.
              </span>
            </span>
          </li>
          <li>
            <span>3.</span>
            <span>
              Select your <b>account name</b> at the bottom left of the screen,
              then select <b>Preferences</b>.
            </span>
          </li>
          <li>
            <span>4.</span>
            <span>
              In the <b>Your MDM Servers</b> section, select <b>Add</b>.
            </span>
          </li>
          <li>
            <span>5.</span>
            <span>Enter a name for the server such as “Fleet”.</span>
          </li>
          <li>
            <span>6.</span>
            <span>
              Under <b>MDM Server Settings</b>, upload the public key downloaded
              in the first step and select <b>Save</b>.
            </span>
          </li>
          <li>
            <span>7.</span>
            <span>
              In the <b>Default Device Assignment</b> section, select{" "}
              <b>Change</b>, then assign the newly created server as the default
              for your Macs, iPhones, and iPads. Then select <b>Done</b>.
            </span>
          </li>
          <li>
            <span>8.</span>
            <span>
              Select newly created server in the sidebar, then select{" "}
              <b>Download MDM Server Token</b> on the top.
            </span>
          </li>
          <li>
            <span>9.</span>
            <span>Upload the downloaded token (.p7m file).</span>
          </li>
        </ol>
        <FileUploader
          className={`${baseClass}__file-uploader ${
            isUploading ? `${baseClass}__file-uploader--loading` : ""
          }`}
          accept=".p7m"
          message="ABM token (.p7m)"
          graphicName={"file-p7m"}
          buttonType="link"
          buttonMessage={isUploading ? "Uploading..." : "Upload"}
          fileDetails={tokenFile ? { name: tokenFile.name } : undefined}
          onFileUpload={onSelectFile}
        />
        <div className="modal-cta-wrap">
          <Button
            variant="brand"
            onClick={uploadAbmToken}
            isLoading={isUploading}
            disabled={!tokenFile || isUploading}
          >
            Add ABM
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default AddAbmModal;
