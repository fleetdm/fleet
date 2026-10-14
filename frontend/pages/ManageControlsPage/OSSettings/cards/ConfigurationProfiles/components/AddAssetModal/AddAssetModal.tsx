import React, { useRef, useState } from "react";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import FileUploader from "components/FileUploader";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import { getErrorReason } from "interfaces/errors";
import mdmAPI from "services/entities/mdm";

const baseClass = "add-asset-modal";

const LEARN_MORE_URL =
  "https://fleetdm.com/learn-more-about/configuration-profile-assets";

const DEFAULT_ERROR_MESSAGE = "Couldn't add asset. Please try again.";

const splitFileName = (fileName: string): { name: string; ext: string } => {
  const lastDot = fileName.lastIndexOf(".");
  const name = lastDot > 0 ? fileName.slice(0, lastDot) : fileName;
  const ext = lastDot > 0 ? fileName.slice(lastDot + 1) : "";
  return { name, ext };
};

interface IAddAssetModalProps {
  currentTeamId: number;
  onUpload: () => void;
  closeModal: () => void;
}

const AddAssetModal = ({
  currentTeamId,
  onUpload,
  closeModal,
}: IAddAssetModalProps) => {
  const [isLoading, setIsLoading] = useState(false);
  const [fileName, setFileName] = useState<string | null>(null);

  const fileRef = useRef<File | null>(null);

  const onDone = () => {
    fileRef.current = null;
    setFileName(null);
    closeModal();
  };

  const onFileOpen = (files: FileList | null) => {
    if (!files || files.length === 0) {
      return;
    }
    const file = files[0];
    fileRef.current = file;
    setFileName(file.name);
  };

  const onAddAsset = async () => {
    if (!fileRef.current) {
      notify.error(DEFAULT_ERROR_MESSAGE);
      return;
    }

    setIsLoading(true);
    try {
      await mdmAPI.uploadAsset({
        file: fileRef.current,
        teamId: currentTeamId,
      });
      notify.success("Successfully added.");
      onUpload();
    } catch (e) {
      notify.error(getErrorReason(e) || DEFAULT_ERROR_MESSAGE, { response: e });
    } finally {
      setIsLoading(false);
      onDone();
    }
  };

  let fileDetails;
  if (fileName) {
    const { name, ext } = splitFileName(fileName);
    fileDetails = { name, description: ext ? `.${ext}` : undefined };
  }

  return (
    <Modal className={baseClass} title="Add asset" onExit={onDone}>
      <div className={`${baseClass}__modal-content-wrap`}>
        <FileUploader
          graphicName="file-json"
          title="Upload asset"
          message={
            <>
              Only JSON files with com.apple.asset.* are supported. Referenced
              data (Reference.DataURL) must be self-hosted.{" "}
              <CustomLink newTab text="Learn more" url={LEARN_MORE_URL} />
            </>
          }
          accept=".json"
          buttonType="secondary"
          buttonMessage="Choose file"
          onFileUpload={onFileOpen}
          fileDetails={fileDetails}
        />
        <div className="modal-cta-wrap">
          <Button
            onClick={onAddAsset}
            isLoading={isLoading}
            disabled={!fileName}
          >
            Add asset
          </Button>
          <Button variant="secondary" onClick={onDone}>
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default AddAssetModal;
