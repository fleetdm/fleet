import React, { useContext, useState } from "react";

import { NotificationContext } from "context/notification";
import { getFileDetails } from "utilities/file/fileUtils";
import getInstallScript from "utilities/software_install_scripts";

import { ILabel, ILabelIdentifier } from "interfaces/label";
import { InstallType } from "interfaces/software";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import Editor from "components/Editor";
import {
  FileUploader,
  FileDetails,
} from "components/FileUploader/FileUploader";
import Spinner from "components/Spinner";
import TooltipWrapper from "components/TooltipWrapper";

import AddPackageAdvancedOptions from "../AddPackageAdvancedOptions";

import { generateFormValidation, INSTALL_TYPE_OPTIONS } from "./helpers";

export const baseClass = "add-package-form";

const UploadingSoftware = () => {
  return (
    <div className={`${baseClass}__uploading-message`}>
      <Spinner centered={false} />
      <p>Uploading. It may take a few minutes to finish.</p>
    </div>
  );
};

export interface IAddPackageFormData {
  software: File | null;
  installScript: string;
  preInstallCondition?: string;
  postInstallScript?: string;
  selfService: boolean;
  installType: InstallType; // (TODO: confirm naming, current design is “install”, in conversation: https://github.com/fleetdm/fleet/pull/20872#discussion_r1710108284 )
  labelsIncludeAny: ILabelIdentifier[];
  labelsExcludeAny: ILabelIdentifier[];
  // - “All hosts” - omit both of above (tbc w Noah: https://github.com/fleetdm/fleet/pull/20872#discussion_r1710119518)
}

export interface IFormValidation {
  isValid: boolean;
  software: { isValid: boolean };
  preInstallCondition?: { isValid: boolean; message?: string };
  postInstallScript?: { isValid: boolean; message?: string };
  selfService?: { isValid: boolean };
  // TODO - confirm
  installType?: { isValid: boolean };
  // TODO - confirm
  labelsIncludeAny?: { isValid: boolean };
  // TODO - confirm
  labelsExcludeAny?: { isValid: boolean };
}

interface IAddPackageFormProps {
  isUploading: boolean;
  onCancel: () => void;
  onSubmit: (formData: IAddPackageFormData) => void;
  customLabels?: ILabel[];
}

const AddPackageForm = ({
  isUploading,
  onCancel,
  onSubmit,
}: IAddPackageFormProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const [showPreInstallCondition, setShowPreInstallCondition] = useState(false);
  const [showPostInstallScript, setShowPostInstallScript] = useState(false);
  const [formData, setFormData] = useState<IAddPackageFormData>({
    software: null,
    installScript: "",
    preInstallCondition: undefined,
    postInstallScript: undefined,
    selfService: false,
    installType: "manual",
    labelsIncludeAny: [],
    labelsExcludeAny: [],
  });
  const [formValidation, setFormValidation] = useState<IFormValidation>({
    isValid: false,
    software: { isValid: false },
  });

  const onFileUpload = (files: FileList | null) => {
    if (files && files.length > 0) {
      const file = files[0];

      let installScript: string;
      try {
        installScript = getInstallScript(file.name);
      } catch (e) {
        renderFlash("error", `${e}`);
        return;
      }

      const newData = {
        ...formData,
        software: file,
        installScript,
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

  const onChangeInstallType = (value: InstallType) => {
    const newData = { ...formData, installType: value };
    setFormData(newData);
  };
  const onToggleSelfServiceCheckbox = (value: boolean) => {
    const newData = { ...formData, selfService: value };
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
            // this is the Install script section - TODO(jacob) move into advanced section
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
          <Dropdown
            value={formData.installType}
            options={INSTALL_TYPE_OPTIONS}
            searchable={false}
            onChange={onChangeInstallType}
          />
          <Checkbox
            value={formData.selfService}
            onChange={onToggleSelfServiceCheckbox}
          >
            <TooltipWrapper
              tipContent={
                <>
                  End users can install from{" "}
                  <b>Fleet Desktop {">"} Self-service</b>.
                </>
              }
            >
              Self-service
            </TooltipWrapper>
          </Checkbox>
          <AddPackageAdvancedOptions
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

export default AddPackageForm;
