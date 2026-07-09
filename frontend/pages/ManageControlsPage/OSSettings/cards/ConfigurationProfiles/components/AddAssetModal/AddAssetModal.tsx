import React, { useRef, useState } from "react";

import { notify } from "components/ToastNotification";

import { getErrorReason } from "interfaces/errors";
import mdmAPI from "services/entities/mdm";

import Button from "components/buttons/Button";
import Card from "components/Card";
import CustomLink from "components/CustomLink";
import Graphic from "components/Graphic";
import Icon from "components/Icon";
import Modal from "components/Modal";

const baseClass = "add-asset-modal";

const LEARN_MORE_URL =
  "https://fleetdm.com/learn-more-about/configuration-profile-assets";

const DEFAULT_ERROR_MESSAGE = "Couldn't add asset. Please try again.";

interface IFileChooserProps {
  isLoading: boolean;
  onFileOpen: (files: FileList | null) => void;
}

const FileChooser = ({ isLoading, onFileOpen }: IFileChooserProps) => {
  const inputRef = useRef<HTMLInputElement>(null);

  return (
    <div className={`${baseClass}__file-chooser`}>
      <Graphic name="file-json" className={`${baseClass}__graphic`} />
      <span className={`${baseClass}__file-chooser--title`}>Upload asset</span>
      <span className={`${baseClass}__file-chooser--message`}>
        Currently, only asset declarations (com.apple.asset) are supported. All
        other assets must be user hosted.{" "}
        <CustomLink newTab text="Learn more" url={LEARN_MORE_URL} />
      </span>
      <Button
        className={`${baseClass}__upload-button`}
        variant="brand-inverse-icon"
        isLoading={isLoading}
        onClick={() => inputRef.current?.click()}
      >
        <span className={`${baseClass}__file-chooser--button-wrap`}>
          Choose file <Icon name="upload" color="core-fleet-green" />
        </span>
      </Button>
      <input
        ref={inputRef}
        accept=".json"
        id="upload-asset"
        type="file"
        hidden
        onChange={(e) => {
          onFileOpen(e.target.files);
        }}
      />
    </div>
  );
};

const FileDetails = ({ name }: { name: string }) => (
  <div className={`${baseClass}__selected-file`}>
    <Graphic name="file-json" className={`${baseClass}__graphic`} />
    <div className={`${baseClass}__selected-file--details`}>
      <div className={`${baseClass}__selected-file--details--name`}>{name}</div>
      <div className={`${baseClass}__selected-file--details--platform`}>
        .json
      </div>
    </div>
  </div>
);

interface IAddAssetModalProps {
  currentTeamId: number;
  onUpload: () => void;
  setShowModal: React.Dispatch<React.SetStateAction<boolean>>;
}

const AddAssetModal = ({
  currentTeamId,
  onUpload,
  setShowModal,
}: IAddAssetModalProps) => {
  const [isLoading, setIsLoading] = useState(false);
  const [fileName, setFileName] = useState<string | null>(null);

  const fileRef = useRef<File | null>(null);

  const onDone = () => {
    fileRef.current = null;
    setFileName(null);
    setShowModal(false);
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

  return (
    <Modal className={baseClass} title="Add asset" onExit={onDone}>
      <div className={`${baseClass}__modal-content-wrap`}>
        <Card color="grey" className={`${baseClass}__file`}>
          {!fileName ? (
            <FileChooser isLoading={isLoading} onFileOpen={onFileOpen} />
          ) : (
            <FileDetails name={fileName} />
          )}
        </Card>
        <div className="modal-cta-wrap">
          <Button
            onClick={onAddAsset}
            isLoading={isLoading}
            disabled={!fileName}
          >
            Add asset
          </Button>
          <Button variant="inverse" onClick={onDone}>
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default AddAssetModal;
