// Used in AddPackageModal.tsx and EditSoftwareModal.tsx
import React, { useContext, useState } from "react";

import { NotificationContext } from "context/notification";
import { getFileDetails } from "utilities/file/fileUtils";
import getDefaultInstallScript from "utilities/software_install_scripts";
import getDefaultUninstallScript from "utilities/software_uninstall_scripts";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import FileUploader from "components/FileUploader";
import Spinner from "components/Spinner";
import TooltipWrapper from "components/TooltipWrapper";

import PackageAdvancedOptions from "../PackageAdvancedOptions";

import { generateFormValidation } from "./helpers";

export const baseClass = "package-form";

const UploadingSoftware = () => {
  return (
    <div className={`${baseClass}__uploading-message`}>
      <Spinner centered={false} />
      <p>Uploading software. This may take a few minutes to finish.</p>
    </div>
  );
};

export interface IPackageFormData {
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

interface IPackageFormProps {
  isUploading: boolean;
  onCancel: () => void;
  onSubmit: (formData: IPackageFormData) => void;
  isEditingSoftware?: boolean;
  defaultSoftware?: any; // TODO
  defaultInstallScript?: string;
  defaultPreInstallQuery?: string;
  defaultPostInstallScript?: string;
  defaultUninstallScript?: string;
  defaultSelfService?: boolean;
}

const ACCEPTED_EXTENSIONS = ".pkg,.msi,.exe,.deb";

const PackageForm = ({
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
}: IPackageFormProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const initialFormData = {
    software: defaultSoftware || null,
    installScript: defaultInstallScript || "",
    preInstallQuery: defaultPreInstallQuery || "",
    postInstallScript: defaultPostInstallScript || "",
    uninstallScript: defaultUninstallScript || "",
    selfService: defaultSelfService || false,
  };
  const [formData, setFormData] = useState<IPackageFormData>(initialFormData);
  const [formValidation, setFormValidation] = useState<IFormValidation>({
    isValid: false,
    software: { isValid: false },
  });

  const onFileSelect = (files: FileList | null) => {
    if (files && files.length > 0) {
      const file = files[0];

      // Only populate default install/uninstall scripts when adding (but not editing) software
      if (isEditingSoftware) {
        const newData = { ...formData, software: file };
        setFormData(newData);
        setFormValidation(generateFormValidation(newData));
      } else {
        let newDefaultInstallScript: string;
        try {
          newDefaultInstallScript = getDefaultInstallScript(file.name);
        } catch (e) {
          renderFlash("error", `${e}`);
          return;
        }

        let newDefaultUninstallScript: string;
        try {
          newDefaultUninstallScript = getDefaultUninstallScript(file.name);
        } catch (e) {
          renderFlash("error", `${e}`);
          return;
        }

        const newData = {
          ...formData,
          software: file,
          installScript: newDefaultInstallScript || "",
          uninstallScript: newDefaultUninstallScript || "",
        };
        setFormData(newData);
        setFormValidation(generateFormValidation(newData));
      }
    }
  };

  const onFormSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit(formData);
  };

  const onChangeInstallScript = (value: string) => {
    const newData = { ...formData, installScript: value };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
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
        <UploadingSoftware /> // Note: Sarah is replacing uploading state as subsequent 4.57 feature
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
          <PackageAdvancedOptions
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

// Allows form not to re-render as long as its props don't change
export default React.memo(PackageForm);
