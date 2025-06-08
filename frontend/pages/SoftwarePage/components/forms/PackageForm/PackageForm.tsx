// Used in AddPackageModal.tsx and EditSoftwareModal.tsx
import React, { useContext, useState, useEffect, useCallback } from "react";
import classnames from "classnames";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import {
  getExtensionFromFileName,
  getFileDetails,
} from "utilities/file/fileUtils";
import getDefaultInstallScript from "utilities/software_install_scripts";
import getDefaultUninstallScript from "utilities/software_uninstall_scripts";
import { ILabelSummary } from "interfaces/label";
import { PackageType } from "interfaces/package_type";
import { SoftwareCategory } from "interfaces/software";

import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";
import FileUploader from "components/FileUploader";
import {
  CUSTOM_TARGET_OPTIONS,
  generateHelpText,
  generateSelectedLabels,
  getCustomTarget,
  getTargetType,
} from "pages/SoftwarePage/helpers";
import TargetLabelSelector from "components/TargetLabelSelector";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Card from "components/Card";
import SoftwareOptionsSelector from "pages/SoftwarePage/components/forms/SoftwareOptionsSelector";

import PackageAdvancedOptions from "../PackageAdvancedOptions";
import { createTooltipContent, generateFormValidation } from "./helpers";

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
  automaticInstall: boolean; // Used on add but not edit
  categories: string[];
}

export interface IPackageFormValidation {
  isValid: boolean;
  software: { isValid: boolean };
  preInstallQuery?: { isValid: boolean; message?: string };
  installScript?: { isValid: boolean; message?: string };
  uninstallScript?: { isValid: boolean; message?: string };
  customTarget?: { isValid: boolean };
}

interface IPackageFormProps {
  labels: ILabelSummary[];
  showSchemaButton?: boolean;
  onCancel: () => void;
  onSubmit: (formData: IPackageFormData) => void;
  onClickShowSchema?: () => void;
  onClickPreviewEndUserExperience: () => void;
  isEditingSoftware?: boolean;
  defaultSoftware?: any; // TODO
  defaultInstallScript?: string;
  defaultPreInstallQuery?: string;
  defaultPostInstallScript?: string;
  defaultUninstallScript?: string;
  defaultSelfService?: boolean;
  defaultCategories?: SoftwareCategory[];
  className?: string;
  /** Indicates that this PackageForm deals with an entity that can be managed by GitOps, and so should be disabled when gitops mode is enabled */
  gitopsCompatible?: boolean;
}
// application/gzip is used for .tar.gz files because browsers can't handle double-extensions correctly
const ACCEPTED_EXTENSIONS = ".pkg,.msi,.exe,.deb,.rpm,application/gzip,.tgz";

const PackageForm = ({
  labels,
  showSchemaButton = false,
  onClickShowSchema,
  onCancel,
  onSubmit,
  onClickPreviewEndUserExperience,
  isEditingSoftware = false,
  defaultSoftware,
  defaultInstallScript,
  defaultPreInstallQuery,
  defaultPostInstallScript,
  defaultUninstallScript,
  defaultSelfService,
  defaultCategories,
  className,
  gitopsCompatible = false,
}: IPackageFormProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const gitOpsModeEnabled = useContext(AppContext).config?.gitops
    .gitops_mode_enabled;

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
    automaticInstall: false,
    categories: defaultCategories || [],
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

  const onToggleAutomaticInstallCheckbox = useCallback(
    (value: boolean) => {
      const newData = { ...formData, automaticInstall: value };
      setFormData(newData);
    },
    [formData]
  );

  const onToggleSelfServiceCheckbox = (value: boolean) => {
    const newData = { ...formData, selfService: value };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const onSelectCategory = ({
    name,
    value,
  }: {
    name: string;
    value: boolean;
  }) => {
    let newCategories: string[];

    if (value) {
      // Add the name if not already present
      newCategories = formData.categories.includes(name)
        ? formData.categories
        : [...formData.categories, name];
    } else {
      // Remove the name if present
      newCategories = formData.categories.filter((cat) => cat !== name);
    }

    const newData = {
      ...formData,
      categories: newCategories,
    };

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
  const submitTooltipContent = createTooltipContent(formValidation);

  const classNames = classnames(baseClass, className);

  const ext = getExtensionFromFileName(formData?.software?.name || "");
  const isExePackage = ext === "exe";
  const isTarballPackage = ext === "tar.gz";
  // We currently don't support replacing a tarball package
  const canEditFile = isEditingSoftware && !isTarballPackage;

  // If a user preselects automatic install and then uploads a .exe
  // which automatic install is not supported, the form will default
  // back to manual install
  useEffect(() => {
    if ((isExePackage || isTarballPackage) && formData.automaticInstall) {
      onToggleAutomaticInstallCheckbox(false);
    }
  }, [
    formData.automaticInstall,
    isExePackage,
    isTarballPackage,
    onToggleAutomaticInstallCheckbox,
  ]);

  // GitOps mode hides SoftwareOptionsSelector and TargetLabelSelector
  const showOptionsTargetsSelectors = !gitOpsModeEnabled;

  return (
    <div className={classNames}>
      <form className={`${baseClass}__form`} onSubmit={onFormSubmit}>
        <FileUploader
          canEdit={canEditFile}
          graphicName="file-pkg"
          accept={ACCEPTED_EXTENSIONS}
          message=".pkg, .msi, .exe, .deb, .rpm, or .tar.gz"
          onFileUpload={onFileSelect}
          buttonMessage="Choose file"
          buttonType="link"
          className={`${baseClass}__file-uploader`}
          fileDetails={
            formData.software ? getFileDetails(formData.software) : undefined
          }
          gitopsCompatible={false}
          gitOpsModeEnabled={gitOpsModeEnabled}
        />
        <div
          // including `form` class here keeps the children fields subject to the global form
          // children styles
          className={
            gitopsCompatible && gitOpsModeEnabled
              ? `${baseClass}__form-fields--gitops-disabled form`
              : "form"
          }
        >
          {showOptionsTargetsSelectors && (
            <div className={`${baseClass}__form-frame`}>
              <Card
                paddingSize="medium"
                borderRadiusSize={isEditingSoftware ? "medium" : "large"}
              >
                <SoftwareOptionsSelector
                  formData={formData}
                  onToggleAutomaticInstall={onToggleAutomaticInstallCheckbox}
                  onToggleSelfService={onToggleSelfServiceCheckbox}
                  onSelectCategory={onSelectCategory}
                  isCustomPackage
                  isEditingSoftware={isEditingSoftware}
                  isExePackage={isExePackage}
                  isTarballPackage={isTarballPackage}
                  onClickPreviewEndUserExperience={
                    onClickPreviewEndUserExperience
                  }
                />
              </Card>
              <Card
                paddingSize="medium"
                borderRadiusSize={isEditingSoftware ? "medium" : "large"}
              >
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
                  dropdownHelpText={
                    formData.targetType === "Custom" &&
                    generateHelpText(
                      formData.automaticInstall,
                      formData.customTarget
                    )
                  }
                />
              </Card>
            </div>
          )}
        </div>
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
        <div className={`${baseClass}__action-buttons`}>
          {submitTooltipContent ? (
            <TooltipWrapper
              tipContent={submitTooltipContent}
              underline={false}
              showArrow
              tipOffset={10}
              position="left"
            >
              <Button type="submit" disabled={isSubmitDisabled}>
                {isEditingSoftware ? "Save" : "Add software"}
              </Button>
            </TooltipWrapper>
          ) : (
            <Button type="submit" disabled={isSubmitDisabled}>
              {isEditingSoftware ? "Save" : "Add software"}
            </Button>
          )}

          <Button variant="inverse" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </form>
    </div>
  );
};

// Allows form not to re-render as long as its props don't change
export default React.memo(PackageForm);
