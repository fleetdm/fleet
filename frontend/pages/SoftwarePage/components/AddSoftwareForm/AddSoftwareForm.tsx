import React, { useState } from "react";

import { getPlatformDisplayName } from "utilities/file/fileUtils";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Spinner from "components/Spinner";
import Button from "components/buttons/Button";
import FileUploader from "components/FileUploader";

import ProfileGraphic from "pages/ManageControlsPage/OSSettings/cards/CustomSettings/components/ProfileUploader/components/AddProfileGraphic";

import AddSoftwareAdvancedOptions from "../AddSoftwareAdvancedOptions/AddSoftwareAdvancedOptions";

const baseClass = "add-software-form";

const getFileDetails = (file: File) => {
  return {
    name: file.name,
    platform: getPlatformDisplayName(file),
  };
};

const UploadingSoftware = () => {
  return (
    <div className={`${baseClass}__uploading-message`}>
      <Spinner centered={false} />
      <p>Uploading. It may take few minutes to finish.</p>
    </div>
  );
};

// TODO: if we reuse this one more time, we should consider moving this
// into FileUploader as a default preview. Currently we have this in
// AddProfileModal.tsx and here.
const FileDetails = ({
  details: { name, platform },
}: {
  details: {
    name: string;
    platform: string;
  };
}) => (
  <div className={`${baseClass}__selected-file`}>
    <ProfileGraphic baseClass={baseClass} />
    <div className={`${baseClass}__selected-file--details`}>
      <div className={`${baseClass}__selected-file--details--name`}>{name}</div>
      <div className={`${baseClass}__selected-file--details--platform`}>
        {platform}
      </div>
    </div>
  </div>
);

export interface IAddSoftwareFormData {
  software: File | null;
  installScript: string;
  preInstallCondition: string;
  postInstallScript: string;
}

interface IAddSoftwareFormProps {
  isUploading: boolean;
  onCancel: () => void;
  onSubmit: (formData: IAddSoftwareFormData) => void;
}

const AddSoftwareForm = ({
  isUploading,
  onCancel,
  onSubmit,
}: IAddSoftwareFormProps) => {
  const [formData, setFormData] = useState<IAddSoftwareFormData>({
    software: null,
    installScript: "",
    preInstallCondition: "",
    postInstallScript: "",
  });

  const onFileUpload = (files: FileList | null) => {
    if (files && files.length > 0) {
      setFormData({ ...formData, software: files[0] });
    }
  };

  const onFormSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit(formData);
  };

  const onChangeInstallScript = (value: string) => {
    setFormData({ ...formData, installScript: value });
  };

  const onChangePreInstallCondition = (value: string) => {
    setFormData({ ...formData, preInstallCondition: value });
  };

  const onChangePostInstallScript = (value: string) => {
    setFormData({ ...formData, postInstallScript: value });
  };

  return (
    <div className={baseClass}>
      {isUploading ? (
        <UploadingSoftware />
      ) : (
        <form className={`${baseClass}__form`} onSubmit={onFormSubmit}>
          <FileUploader
            graphicName={"file-pkg"}
            accept=".pkg,.msi,.exe,.deb"
            message=".pkg, .msi, .exe, or .deb"
            onFileUpload={onFileUpload}
            buttonMessage="Choose file"
            buttonType="link"
            className={`${baseClass}__file-uploader`}
            filePreview={
              formData.software && (
                <FileDetails details={getFileDetails(formData.software)} />
              )
            }
          />
          {formData.software && (
            <InputField
              value={formData.installScript}
              onChange={onChangeInstallScript}
              name="install script"
              label="Install script"
              tooltip="For security agents, add the script provided by the vendor."
              helpText="Fleet will run this command on hosts to install software."
            />
          )}
          <AddSoftwareAdvancedOptions
            preInstallCondition={formData.preInstallCondition}
            postInstallScript={formData.postInstallScript}
            onChangePreInstallCondition={onChangePreInstallCondition}
            onChangePostInstallScript={onChangePostInstallScript}
          />
          <div className="modal-cta-wrap">
            <Button
              type="submit"
              variant="brand"
              disabled={Boolean(!formData.software)}
            >
              Add software
            </Button>
            <Button onClick={onCancel} variant="inverse">
              Cancel
            </Button>
          </div>
        </form>
      )}
    </div>
  );
};

export default AddSoftwareForm;
