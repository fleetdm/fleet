import React, { useState } from "react";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Spinner from "components/Spinner";
import Button from "components/buttons/Button";
import FileUploader from "components/FileUploader";

const baseClass = "add-software-form";

const UploadingSoftware = () => {
  return (
    <div className={`${baseClass}__uploading-message`}>
      <Spinner centered={false} />
      <p>Uploading. It may take few minutes to finish.</p>
    </div>
  );
};

export interface IAddSoftwareFormData {
  software: File | null;
  installScript: string;
  preInstallQuery: string;
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
    preInstallQuery: "",
    postInstallScript: "",
  });

  const onFileUpload = (files: FileList | null) => {
    if (files && files.length) {
      setFormData({ ...formData, software: files[0] });
    }
  };

  const onFormSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();

    onSubmit(formData);
  };

  return (
    <div className={baseClass}>
      {isUploading ? (
        <UploadingSoftware />
      ) : (
        <form className={`${baseClass}__form`} onSubmit={onFormSubmit}>
          <FileUploader
            graphicName={"file-pkg"}
            message=".pkg, .msi, .exe, or .deb"
            onFileUpload={onFileUpload}
            buttonMessage="Choose file"
            buttonType="link"
            className={`${baseClass}__file-uploader`}
          />
          {formData.software && (
            <InputField
              value={formData.installScript}
              name="install script"
              label="Install script"
              tooltip="For security agents, add the script provided by the vendor."
              helpText="Fleet will run this command on hosts to install software."
            />
          )}
          <div className="modal-cta-wrap">
            <Button type="submit" variant="brand" disabled={false}>
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
