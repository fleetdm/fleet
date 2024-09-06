// Used in AddPackageModal.tsx and EditSoftwareModal.tsx
import React, { useContext, useState } from "react";

import { NotificationContext } from "context/notification";
import { getFileDetails } from "utilities/file/fileUtils";
import deepDifference from "utilities/deep_difference";
import getDefaultInstallScript from "utilities/software_install_scripts";
import getDefaultUninstallScript from "utilities/software_uninstall_scripts";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import { FileUploader } from "components/FileUploader/FileUploader";
import Spinner from "components/Spinner";
import TooltipWrapper from "components/TooltipWrapper";

import AddPackageAdvancedOptions from "../AddPackageAdvancedOptions";

import { generateFormValidation } from "./helpers";

export const baseClass = "add-package-form";

const UploadingSoftware = () => {
  return (
    <div className={`${baseClass}__uploading-message`}>
      <Spinner centered={false} />
      <p>Adding software. This may take a few minutes to finish.</p>
    </div>
  );
};

export interface IAddPackageFormData {
  software: File | null;
  preInstallQuery?: string;
  installScript: string;
  postInstallScript?: string;
  uninstallScript?: string;
  selfService: boolean;
}

export interface IFormValidation {
  isValid: boolean;
  software: { isValid: boolean };
  preInstallQuery?: { isValid: boolean; message?: string };
  postInstallScript?: { isValid: boolean; message?: string };
  uninstallScript?: { isValid: boolean; message?: string };
  selfService?: { isValid: boolean };
}

interface IAddPackageFormProps {
  isUploading: boolean;
  onCancel: () => void;
  onSubmit: (formData: IAddPackageFormData) => void;
  isEditingSoftware?: boolean;
  defaultSoftware?: any; // TODO
  defaultInstallScript?: string;
  defaultPreInstallQuery?: string;
  defaultPostInstallScript?: string;
  defaultUninstallScript?: string;
  defaultSelfService?: boolean;
  toggleSaveChangesForEditModal?: () => void;
}

const ACCEPTED_EXTENSIONS = ".pkg,.msi,.exe,.deb";

const AddPackageForm = ({
  isUploading,
  onCancel,
  onSubmit,
  isEditingSoftware = false,
  defaultSoftware,
  defaultInstallScript,
  defaultPreInstallQuery,
  defaultPostInstallScript,
  defaultUninstallScript,
  defaultSelfService,
  toggleSaveChangesForEditModal,
}: IAddPackageFormProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const initialFormData = {
    software: defaultSoftware || null,
    installScript: defaultInstallScript || "",
    preInstallQuery: defaultPreInstallQuery || undefined,
    postInstallScript: defaultPostInstallScript || undefined,
    uninstallScript: defaultUninstallScript || undefined,
    selfService: defaultSelfService || false,
  };
  const [formData, setFormData] = useState<IAddPackageFormData>(
    initialFormData
  );
  const [formValidation, setFormValidation] = useState<IFormValidation>({
    isValid: false,
    software: { isValid: false },
  });

  const onFileSelect = (files: FileList | null) => {
    if (files && files.length > 0) {
      const file = files[0];

      let defaultInstallScript: string;
      try {
        defaultInstallScript = getDefaultInstallScript(file.name);
      } catch (e) {
        renderFlash("error", `${e}`);
        return;
      }

      let defaultUninstallScript: string;
      try {
        defaultUninstallScript = getDefaultUninstallScript(file.name);
      } catch (e) {
        renderFlash("error", `${e}`);
        return;
      }

      const newData = {
        ...formData,
        software: file,
        installScript: defaultInstallScript,
        uninstallScript: defaultUninstallScript,
      };
      setFormData(newData);
      setFormValidation(generateFormValidation(newData));
    }
  };

  const onFormSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
    // When editing software, we prompt a save changes modal for all changes except to self-service
    const updates = deepDifference(initialFormData, formData);
    const onlySelfServiceUpdated =
      Object.keys(updates).length === 1 && "selfService" in updates;

    const promptSaveChangesForEditModal =
      isEditingSoftware && !onlySelfServiceUpdated;

    if (promptSaveChangesForEditModal && !!toggleSaveChangesForEditModal) {
      evt.preventDefault();
      toggleSaveChangesForEditModal();
    } else {
      evt.preventDefault();

      onSubmit(formData);
    }
  };

  const onChangeInstallScript = (value: string) => {
    setFormData({ ...formData, installScript: value });
  };

  const onChangePreInstallQuery = (value?: string) => {
    const newData = { ...formData, preInstallQuery: value };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const onChangePostInstallScript = (value?: string) => {
    const newData = { ...formData, postInstallScript: value };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const onChangeUninstallScript = (value?: string) => {
    const newData = { ...formData, uninstallScript: value };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const onToggleSelfServiceCheckbox = (value: boolean) => {
    const newData = { ...formData, selfService: value };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const isSubmitDisabled = !formValidation.isValid;

  return (
    <div className={baseClass}>
      {isUploading ? (
        <UploadingSoftware />
      ) : (
        <form className={`${baseClass}__form`} onSubmit={onFormSubmit}>
          <FileUploader
            canEdit={isEditingSoftware}
            graphicName={"file-pkg"}
            accept={ACCEPTED_EXTENSIONS}
            message=".pkg, .msi, .exe, or .deb"
            onFileUpload={onFileSelect}
            buttonMessage="Choose file"
            buttonType="link"
            className={`${baseClass}__file-uploader`}
            fileDetails={
              formData.software ? getFileDetails(formData.software) : undefined
            }
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
            selectedPackage={formData.software}
            errors={{
              preInstallQuery: formValidation.preInstallQuery?.message,
              postInstallScript: formValidation.postInstallScript?.message,
            }}
            preInstallQuery={formData.preInstallQuery}
            installScript={formData.installScript}
            postInstallScript={formData.postInstallScript}
            uninstallScript={formData.uninstallScript}
            onChangePreInstallQuery={onChangePreInstallQuery}
            onChangeInstallScript={onChangeInstallScript}
            onChangePostInstallScript={onChangePostInstallScript}
            onChangeUninstallScript={onChangeUninstallScript}
          />
          <div className="modal-cta-wrap">
            <Button type="submit" variant="brand" disabled={isSubmitDisabled}>
              {isEditingSoftware ? "Save" : "Add software"}
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

// Allows form not to re-render as long as it's props don't change
export default React.memo(AddPackageForm);
