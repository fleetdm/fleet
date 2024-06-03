import React, { useState } from "react";

import getInstallScript from "utilities/software_install_scripts";

import Spinner from "components/Spinner";
import Button from "components/buttons/Button";
import Editor from "components/Editor";
import FileUploader, {
  FileDetails,
} from "components/FileUploader/FileUploader";

import { getFileDetails } from "utilities/file/fileUtils";

import AddSoftwareAdvancedOptions from "../AddSoftwareAdvancedOptions";

import { generateFormValidation } from "./helpers";

export const baseClass = "add-software-form";

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
  preInstallCondition?: string;
  postInstallScript?: string;
}

export interface IFormValidation {
  isValid: boolean;
  software: { isValid: boolean };
  preInstallCondition?: { isValid: boolean; message?: string };
  postInstallScript?: { isValid: boolean; message?: string };
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
  const [showPreInstallCondition, setShowPreInstallCondition] = useState(false);
  const [showPostInstallScript, setShowPostInstallScript] = useState(false);
  const [formData, setFormData] = useState<IAddSoftwareFormData>({
    software: null,
    installScript: "",
    preInstallCondition: undefined,
    postInstallScript: undefined,
  });
  const [formValidation, setFormValidation] = useState<IFormValidation>({
    isValid: false,
    software: { isValid: false },
  });

  const onFileUpload = (files: FileList | null) => {
    if (files && files.length > 0) {
      const file = files[0];
      const newData = {
        ...formData,
        software: file,
        installScript: getInstallScript(file.name),
      };
      setFormData(newData);
      setFormValidation(
        generateFormValidation(
          newData,
          showPreInstallCondition,
          showPostInstallScript
        )
      );
    }
  };

  const onFormSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit(formData);
  };

  const onTogglePreInstallConditionCheckbox = (value: boolean) => {
    const newData = { ...formData, preInstallCondition: undefined };
    setShowPreInstallCondition(value);
    setFormData(newData);
    setFormValidation(
      generateFormValidation(newData, value, showPostInstallScript)
    );
  };

  const onTogglePostInstallScriptCheckbox = (value: boolean) => {
    const newData = { ...formData, postInstallScript: undefined };
    setShowPostInstallScript(value);
    setFormData(newData);
    setFormValidation(
      generateFormValidation(newData, showPreInstallCondition, value)
    );
  };

  const onChangeInstallScript = (value: string) => {
    setFormData({ ...formData, installScript: value });
  };

  const onChangePreInstallCondition = (value?: string) => {
    const newData = { ...formData, preInstallCondition: value };
    setFormData(newData);
    setFormValidation(
      generateFormValidation(
        newData,
        showPreInstallCondition,
        showPostInstallScript
      )
    );
  };

  const onChangePostInstallScript = (value?: string) => {
    const newData = { ...formData, postInstallScript: value };
    setFormData(newData);
    setFormValidation(
      generateFormValidation(
        newData,
        showPreInstallCondition,
        showPostInstallScript
      )
    );
  };

  const isSubmitDisabled = !formValidation.isValid;

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
            <Editor
              wrapEnabled
              maxLines={10}
              name="install-script"
              onChange={onChangeInstallScript}
              value={formData.installScript}
              helpText="Fleet will run this command on hosts to install software."
              label="Install script"
              labelTooltip={
                <>
                  For security agents, add the script provided by the vendor.
                  <br />
                  In custom scripts, you can use the $INSTALLER_PATH variable to
                  point to the installer.
                </>
              }
            />
          )}
          <AddSoftwareAdvancedOptions
            errors={{
              preInstallCondition: formValidation.preInstallCondition?.message,
              postInstallScript: formValidation.postInstallScript?.message,
            }}
            showPreInstallCondition={showPreInstallCondition}
            showPostInstallScript={showPostInstallScript}
            preInstallCondition={formData.preInstallCondition}
            postInstallScript={formData.postInstallScript}
            onTogglePreInstallCondition={onTogglePreInstallConditionCheckbox}
            onTogglePostInstallScript={onTogglePostInstallScriptCheckbox}
            onChangePreInstallCondition={onChangePreInstallCondition}
            onChangePostInstallScript={onChangePostInstallScript}
          />
          <div className="modal-cta-wrap">
            <Button type="submit" variant="brand" disabled={isSubmitDisabled}>
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
