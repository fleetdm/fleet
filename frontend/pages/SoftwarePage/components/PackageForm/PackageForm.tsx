// Used in AddPackageModal.tsx and EditSoftwareModal.tsx
import React, { useContext, useState, useEffect } from "react";
import classnames from "classnames";

import { NotificationContext } from "context/notification";
import { getFileDetails } from "utilities/file/fileUtils";
import getDefaultInstallScript from "utilities/software_install_scripts";
import getDefaultUninstallScript from "utilities/software_uninstall_scripts";
import { ILabelSummary } from "interfaces/label";
import { PackageType } from "interfaces/package_type";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import FileUploader from "components/FileUploader";
import TooltipWrapper from "components/TooltipWrapper";
import {
  InstallType,
  InstallTypeSection,
} from "pages/SoftwarePage/SoftwareAddPage/SoftwareFleetMaintained/FleetMaintainedAppDetailsPage/FleetAppDetailsForm/FleetAppDetailsForm";
import TargetLabelSelector from "components/TargetLabelSelector";

import PackageAdvancedOptions from "../PackageAdvancedOptions";

import {
  CUSTOM_TARGET_OPTIONS,
  generateFormValidation,
  generateSelectedLabels,
  getCustomTarget,
  getTargetType,
} from "./helpers";

export const baseClass = "package-form";

export interface IPackageFormData {
  software: File | null;
  preInstallQuery?: string;
  installScript: string;
  postInstallScript?: string;
  uninstallScript?: string;
  selfService: boolean;
  targetType: string;
  customTarget: string;
  labelTargets: Record<string, boolean>;
  installType: InstallType; // Used on add but not edit
}

export interface IPackageFormValidation {
  isValid: boolean;
  software: { isValid: boolean };
  preInstallQuery?: { isValid: boolean; message?: string };
  customTarget?: { isValid: boolean };
}

interface IPackageFormProps {
  labels: ILabelSummary[];
  showSchemaButton?: boolean;
  onCancel: () => void;
  onSubmit: (formData: IPackageFormData) => void;
  onClickShowSchema?: () => void;
  isEditingSoftware?: boolean;
  defaultSoftware?: any; // TODO
  defaultInstallScript?: string;
  defaultPreInstallQuery?: string;
  defaultPostInstallScript?: string;
  defaultUninstallScript?: string;
  defaultSelfService?: boolean;
  className?: string;
}

const ACCEPTED_EXTENSIONS = ".pkg,.msi,.exe,.deb,.rpm";

const PackageForm = ({
  labels,
  showSchemaButton = false,
  onClickShowSchema,
  onCancel,
  onSubmit,
  isEditingSoftware = false,
  defaultSoftware,
  defaultInstallScript,
  defaultPreInstallQuery,
  defaultPostInstallScript,
  defaultUninstallScript,
  defaultSelfService,
  className,
}: IPackageFormProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const initialFormData: IPackageFormData = {
    software: defaultSoftware || null,
    installScript: defaultInstallScript || "",
    preInstallQuery: defaultPreInstallQuery || "",
    postInstallScript: defaultPostInstallScript || "",
    uninstallScript: defaultUninstallScript || "",
    selfService: defaultSelfService || false,
    targetType: getTargetType(defaultSoftware),
    customTarget: getCustomTarget(defaultSoftware),
    labelTargets: generateSelectedLabels(defaultSoftware),
    installType: "manual",
  };

  const [formData, setFormData] = useState<IPackageFormData>(initialFormData);
  const [formValidation, setFormValidation] = useState<IPackageFormValidation>({
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

  const onChangeInstallType = (value: string) => {
    const installType = value as InstallType;
    const newData = { ...formData, installType };
    setFormData(newData);
  };

  const onToggleSelfServiceCheckbox = (value: boolean) => {
    const newData = { ...formData, selfService: value };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const onSelectTargetType = (value: string) => {
    const newData = { ...formData, targetType: value };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const onSelectCustomTarget = (value: string) => {
    const newData = { ...formData, customTarget: value };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const onSelectLabel = ({ name, value }: { name: string; value: boolean }) => {
    const newData = {
      ...formData,
      labelTargets: { ...formData.labelTargets, [name]: value },
    };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const isSubmitDisabled = !formValidation.isValid;

  const classNames = classnames(baseClass, className);

  const ext = formData?.software?.name.split(".").pop() as PackageType;
  const isExePackage = ext === "exe";

  // If a user preselects automatic install and then uploads a .exe
  // which automatic install is not supported, the form will default
  // back to manual install
  useEffect(() => {
    if (isExePackage && formData.installType === "automatic") {
      onChangeInstallType("manual");
    }
  }, [isExePackage]);

  return (
    <div className={classNames}>
      <form className={`${baseClass}__form`} onSubmit={onFormSubmit}>
        <FileUploader
          canEdit={isEditingSoftware}
          graphicName={"file-pkg"}
          accept={ACCEPTED_EXTENSIONS}
          message=".pkg, .msi, .exe, .deb, or .rpm"
          onFileUpload={onFileSelect}
          buttonMessage="Choose file"
          buttonType="link"
          className={`${baseClass}__file-uploader`}
          fileDetails={
            formData.software ? getFileDetails(formData.software) : undefined
          }
        />
        <InstallTypeSection
          isExeFleetApp={isExePackage}
          installType={formData.installType}
          onChangeInstallType={onChangeInstallType}
        />
        <TargetLabelSelector
          selectedTargetType={formData.targetType}
          selectedCustomTarget={formData.customTarget}
          selectedLabels={formData.labelTargets}
          customTargetOptions={CUSTOM_TARGET_OPTIONS}
          className={`${baseClass}__target`}
          onSelectTargetType={onSelectTargetType}
          onSelectCustomTarget={onSelectCustomTarget}
          onSelectLabel={onSelectLabel}
          labels={labels || []}
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
          showSchemaButton={showSchemaButton}
          selectedPackage={formData.software}
          errors={{
            preInstallQuery: formValidation.preInstallQuery?.message,
          }}
          preInstallQuery={formData.preInstallQuery}
          installScript={formData.installScript}
          postInstallScript={formData.postInstallScript}
          uninstallScript={formData.uninstallScript}
          onClickShowSchema={onClickShowSchema}
          onChangePreInstallQuery={onChangePreInstallQuery}
          onChangeInstallScript={onChangeInstallScript}
          onChangePostInstallScript={onChangePostInstallScript}
          onChangeUninstallScript={onChangeUninstallScript}
        />
        <div className="form-buttons">
          <Button type="submit" variant="brand" disabled={isSubmitDisabled}>
            {isEditingSoftware ? "Save" : "Add software"}
          </Button>
          <Button onClick={onCancel} variant="inverse">
            Cancel
          </Button>
        </div>
      </form>
    </div>
  );
};

// Allows form not to re-render as long as its props don't change
export default React.memo(PackageForm);
