// Used in AddPackageModal.tsx and EditSoftwareModal.tsx
import React, { useContext, useState, useEffect, useCallback } from "react";
import classnames from "classnames";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import {
  getExtensionFromFileName,
  getFileDetails,
} from "utilities/file/fileUtils";
import { compareVersions } from "utilities/helpers";
import getDefaultInstallScript from "utilities/software_install_scripts";
import getDefaultUninstallScript from "utilities/software_uninstall_scripts";
import { ILabelSummary } from "interfaces/label";

import { ISoftwareVersion, SoftwareCategory } from "interfaces/software";

import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";

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
import Card from "components/Card";
import SoftwareOptionsSelector from "pages/SoftwarePage/components/forms/SoftwareOptionsSelector";

import PackageAdvancedOptions from "../PackageAdvancedOptions";
import { createTooltipContent, generateFormValidation } from "./helpers";
import PackageVersionSelector from "../PackageVersionSelector";

export const baseClass = "package-form";

export interface IPackageFormData {
  software: File | null;
  version?: string;
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

const getGraphicName = (ext: string) => {
  if (ext === "sh") {
    return "file-sh";
  } else if (ext === "ps1") {
    return "file-ps1";
  }
  return "file-pkg";
};

const renderFileTypeMessage = () => {
  return (
    <>
      macOS (.pkg), iOS/iPadOS (.ipa),
      <br />
      Windows (.msi, .exe.,{" "}
      <TooltipWrapper tipContent="Script-only package">.ps1</TooltipWrapper>),
      or Linux (.deb, .rpm,{" "}
      <TooltipWrapper tipContent="Script-only package">.sh</TooltipWrapper>)
    </>
  );
};

/** Returns the version value to use as the dropdown's default:
/ 1) If a previously selected version is still present in the options, reuse it.
/ 2) Otherwise, fall back to the first option, which is assumed to be the latest.
/ 3) Safe fallback if no options exist which should never happen */
const getDefaultVersion = (
  versionOptions: CustomOptionType[],
  selectedVersion?: string
) => {
  // This shouldn't happen
  if (!versionOptions.length) {
    return "";
  }

  // If we already have a selected version and it exists in options, keep it
  if (selectedVersion) {
    const match = versionOptions.find((opt) => opt.value === selectedVersion);
    if (match) {
      return match.value;
    }
  }

  // Otherwise, default to the first option (which should be latest)
  return versionOptions[0].value;
};

interface IPackageFormProps {
  labels: ILabelSummary[];
  showSchemaButton?: boolean;
  onCancel: () => void;
  onSubmit: (formData: IPackageFormData) => void;
  onClickShowSchema?: () => void;
  onClickPreviewEndUserExperience: (isIosOrIpadosApp: boolean) => void;
  isEditingSoftware?: boolean;
  isFleetMaintainedApp?: boolean;
  defaultSoftware?: any; // TODO
  defaultInstallScript?: string;
  defaultPreInstallQuery?: string;
  defaultPostInstallScript?: string;
  defaultUninstallScript?: string;
  defaultSelfService?: boolean;
  defaultCategories?: SoftwareCategory[] | null;
  className?: string;
  /** Indicates that this PackageForm deals with an entity that can be managed by GitOps, and so should be disabled when gitops mode is enabled */
  gitopsCompatible?: boolean;
}
// application/gzip is used for .tar.gz files because browsers can't handle double-extensions correctly
const ACCEPTED_EXTENSIONS =
  ".pkg,.msi,.exe,.deb,.rpm,application/gzip,.tgz,.sh,.ps1,.ipa";

const PackageForm = ({
  labels,
  showSchemaButton = false,
  onClickShowSchema,
  onCancel,
  onSubmit,
  onClickPreviewEndUserExperience,
  isEditingSoftware = false,
  isFleetMaintainedApp = false,
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
  const { gitops_mode_enabled: gitOpsModeEnabled, repository_url: repoURL } =
    useContext(AppContext).config?.gitops || {};

  const initialFormData: IPackageFormData = {
    software: defaultSoftware || null,
    version: defaultSoftware?.version || "",
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

  const onToggleAutomaticInstall = useCallback(
    (value?: boolean) => {
      const automaticInstall =
        typeof value === "boolean" ? value : !formData.automaticInstall;

      setFormData({ ...formData, automaticInstall });
    },
    [formData]
  );

  const onToggleSelfService = () => {
    const newData = { ...formData, selfService: !formData.selfService };
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

  const onSelectVersion = (version: string) => {
    // For now we can only update version in GitOps
    // Selection is currently disabled in the UI
    const newData = {
      ...formData,
      version,
    };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const disableFieldsForGitOps = gitopsCompatible && gitOpsModeEnabled;
  const isSubmitDisabled = !formValidation.isValid || disableFieldsForGitOps;
  const submitTooltipContent = createTooltipContent(
    formValidation,
    repoURL,
    disableFieldsForGitOps
  );

  const classNames = classnames(baseClass, className);

  const ext = getExtensionFromFileName(formData?.software?.name || "");
  const isExePackage = ext === "exe";
  const isTarballPackage = ext === "tar.gz";
  const isScriptPackage = ext === "sh" || ext === "ps1";
  const isIpaPackage = ext === "ipa";
  // We currently don't support replacing a tarball package
  const canEditFile = isEditingSoftware && !isTarballPackage;

  // If a user preselects automatic install and then uploads a:
  // exe, tarball, script, or ipa which automatic install is not supported,
  // the form will default back to manual install
  useEffect(() => {
    if (
      (isExePackage || isTarballPackage || isScriptPackage || isIpaPackage) &&
      formData.automaticInstall
    ) {
      onToggleAutomaticInstall(false);
    }
  }, [
    formData.automaticInstall,
    isExePackage,
    isTarballPackage,
    isScriptPackage,
    isIpaPackage,
    onToggleAutomaticInstall,
  ]);

  // Show advanced options when a package is selected that's not a script or ipa
  const showAdvancedOptions =
    formData.software && !isScriptPackage && !isIpaPackage;

  // GitOps mode hides SoftwareOptionsSelector and TargetLabelSelector
  const showOptionsTargetsSelectors = !gitOpsModeEnabled;

  const renderSoftwareOptionsSelector = () => (
    <SoftwareOptionsSelector
      formData={formData}
      onToggleAutomaticInstall={onToggleAutomaticInstall}
      onToggleSelfService={onToggleSelfService}
      onSelectCategory={onSelectCategory}
      isCustomPackage
      isEditingSoftware={isEditingSoftware}
      isExePackage={isExePackage}
      isTarballPackage={isTarballPackage}
      isScriptPackage={isScriptPackage}
      isIpaPackage={isIpaPackage}
      onClickPreviewEndUserExperience={() =>
        onClickPreviewEndUserExperience(isIpaPackage)
      }
    />
  );

  const renderTargetLabelSelector = () => (
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
        generateHelpText(formData.automaticInstall, formData.customTarget)
      }
    />
  );

  const renderCustomEditor = () => {
    if (isEditingSoftware && !isFleetMaintainedApp) {
      return null;
    }

    const fmaVersions = defaultSoftware.fleet_maintained_versions || [];

    /** Ensures latest version is first, compatible with sorting n-number of versions */
    fmaVersions.sort((a: ISoftwareVersion, b: ISoftwareVersion) => {
      const v1 = a.version ?? "";
      const v2 = b.version ?? "";
      return compareVersions(v2, v1);
    });

    const hasMultipleVersions = fmaVersions.length > 1;

    const versionOptions = fmaVersions.map(
      (v: ISoftwareVersion, index: number) => {
        // If multiple versions, only adds "Latest" label to the first option
        const isLatest = hasMultipleVersions && index === 0;

        return {
          label: isLatest ? `Latest (${v.version})` : `${v.version}`,
          value: v.version,
        };
      }
    );

    return (
      <PackageVersionSelector
        selectedVersion={getDefaultVersion(versionOptions, formData.version)}
        versionOptions={versionOptions}
        onSelectVersion={onSelectVersion}
        className={`${baseClass}__version-selector`}
      />
    );
  };

  return (
    <div className={classNames}>
      <form className={`${baseClass}__form`} onSubmit={onFormSubmit}>
        <FileUploader
          canEdit={canEditFile}
          customEditor={renderCustomEditor}
          graphicName={getGraphicName(ext || "")}
          accept={ACCEPTED_EXTENSIONS}
          message={renderFileTypeMessage()}
          onFileUpload={onFileSelect}
          buttonMessage="Choose file"
          buttonType="brand-inverse-icon"
          className={`${baseClass}__file-uploader`}
          fileDetails={
            formData.software ? getFileDetails(formData.software) : undefined
          }
          gitopsCompatible={gitopsCompatible}
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
              {isEditingSoftware ? (
                renderSoftwareOptionsSelector()
              ) : (
                <Card
                  paddingSize="medium"
                  borderRadiusSize={isEditingSoftware ? "medium" : "large"}
                >
                  {renderSoftwareOptionsSelector()}
                </Card>
              )}
              {isEditingSoftware ? (
                renderTargetLabelSelector()
              ) : (
                <Card
                  paddingSize="medium"
                  borderRadiusSize={isEditingSoftware ? "medium" : "large"}
                >
                  {renderTargetLabelSelector()}
                </Card>
              )}
            </div>
          )}
        </div>
        {showAdvancedOptions && (
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
            gitopsCompatible={gitopsCompatible}
            gitOpsModeEnabled={gitOpsModeEnabled}
          />
        )}
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
