// Used in AddPackageModal.tsx and EditSoftwareModal.tsx
import React, { useState, useEffect, useCallback } from "react";
import classnames from "classnames";

import useGitOpsMode from "hooks/useGitOpsMode";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import {
  getExtensionFromFileName,
  getFileDetails,
} from "utilities/file/fileUtils";
import getDefaultInstallScript from "utilities/software_install_scripts";
import getDefaultUninstallScript from "utilities/software_uninstall_scripts";
import { ILabelSummary } from "interfaces/label";

import { SoftwareCategory } from "interfaces/software";
import { isScriptOnlyPackageType } from "interfaces/package_type";

import { notify } from "components/ToastNotification";
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
import { DropdownTargetLabelSelector } from "components/TargetLabelSelector";
import SoftwareOptionsSelector from "pages/SoftwarePage/components/forms/SoftwareOptionsSelector";
import InfoBanner from "components/InfoBanner";
import CustomLink from "components/CustomLink";

import PackageAdvancedOptions from "../PackageAdvancedOptions";
import { createTooltipContent, generateFormValidation } from "./helpers";
import SoftwareDeploySlider from "../SoftwareDeploySelector";

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

const getGraphicName = (ext: string) => {
  if (ext === "sh") {
    return "file-sh";
  } else if (ext === "ps1") {
    return "file-ps1";
  }
  return "file-pkg";
};

const renderSoftwareDeployWarningBanner = () => (
  <InfoBanner
    color="yellow"
    className={`${baseClass}__deploy-warning`}
    cta={
      <CustomLink
        url={`${LEARN_MORE_ABOUT_BASE_LINK}/query-templates-for-automatic-install-software`}
        text="Learn more"
        newTab
      />
    }
  >
    Installing software over existing installations might cause issues.
    Fleet&apos;s policy may not detect these existing installations. Please
    create a test fleet in Fleet to verify a smooth installation.
  </InfoBanner>
);

const renderFileTypeMessage = () => {
  return (
    <>
      macOS (.pkg,{" "}
      <TooltipWrapper tipContent="Script-only package">.sh</TooltipWrapper>),
      iOS/iPadOS (.ipa),
      <br />
      Windows (.msi, .exe,{" "}
      <TooltipWrapper tipContent="Script-only package">.ps1</TooltipWrapper>),
      or Linux (.deb, .rpm, .tar.gz,{" "}
      <TooltipWrapper tipContent="Script-only package">.sh</TooltipWrapper>)
    </>
  );
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
  /** When provided, the categories list is fetched dynamically for this fleet. */
  teamId?: number;
  /** Set when this form is mounted inside the multi-package add modal
   * (#48400). Renders a contextual banner just under the file chooser —
   * GitOps copy when GitOps mode is on, the first-added-wins copy otherwise.
   * Other call sites (single-package add page, edit modal) leave it false. */
  multiPackageContext?: boolean;
  /** Restricts the file picker to a specific platform/file type when set —
   * used by the multi-package add modal so a second .pkg upload can't slip
   * onto a Linux title. Falls back to PackageForm's full all-platforms accept
   * + message when omitted. */
  restrictedFileAccept?: string;
  restrictedFileTypeLabel?: React.ReactNode;
  /** Overrides the initial `targetType` for new (non-editing) forms. The
   * multi-package add modal preselects `"Custom"` per Figma. */
  initialTargetType?: string;
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
  teamId,
  multiPackageContext = false,
  restrictedFileAccept,
  restrictedFileTypeLabel,
  initialTargetType,
}: IPackageFormProps) => {
  const { gitOpsModeEnabled, repoURL } = useGitOpsMode("software");

  const initialFormData: IPackageFormData = {
    software: defaultSoftware || null,
    installScript: defaultInstallScript || "",
    preInstallQuery: defaultPreInstallQuery || "",
    postInstallScript: defaultPostInstallScript || "",
    uninstallScript: defaultUninstallScript || "",
    selfService: defaultSelfService || false,
    targetType: initialTargetType ?? getTargetType(defaultSoftware),
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
          notify.error(`${e}`, { response: e });
          return;
        }

        let newDefaultUninstallScript: string;
        try {
          newDefaultUninstallScript = getDefaultUninstallScript(file.name);
        } catch (e) {
          notify.error(`${e}`, { response: e });
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
  const isScriptPackage = isScriptOnlyPackageType(ext);
  const isIpaPackage = ext === "ipa";
  // We currently don't support replacing a tarball package, and FMAs use the
  // page-level Versions modal to switch versions rather than a file replace.
  const canEditFile =
    isEditingSoftware && !isTarballPackage && !isFleetMaintainedApp;

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

  // Show advanced options for any selected package except .ipa (includes script packages).
  const showAdvancedOptions = formData.software && !isIpaPackage;

  const showDeploySoftwareSlider =
    !!formData.software && // show after selection
    !gitOpsModeEnabled && // hide in gitOps mode
    !isEditingSoftware && // show only on add, not edit
    !multiPackageContext && // hide in the multi-package add modal — per Figma 2:130 the modal omits the deploy slider
    // automatic install is not supported for ipa packages, exe, tarball, or script packages
    !isIpaPackage &&
    !isExePackage &&
    !isTarballPackage &&
    !isScriptPackage;

  // 4.83+ Show deploy slider on add if the package type supports it.
  // Hide from gitOps mode
  const renderSoftwareDeploySlider = () => (
    <>
      <SoftwareDeploySlider
        deploySoftware={formData.automaticInstall}
        onToggleDeploySoftware={onToggleAutomaticInstall}
      />
      {formData.automaticInstall && renderSoftwareDeployWarningBanner()}
    </>
  );

  // GitOps mode hides SoftwareOptionsSelector and TargetLabelSelector.
  // 4.83 removed option/targets from the (single-package) Add page; the
  // multi-package Add modal (#48400) reintroduces the targets selector only,
  // since each package on a multi-package title needs its own label scope.
  // The options selector (self-service + categories) stays edit-only.
  const showSoftwareOptionsSelector = !gitOpsModeEnabled && isEditingSoftware;
  const showTargetLabelSelector =
    !gitOpsModeEnabled && (isEditingSoftware || multiPackageContext);

  const renderSoftwareOptionsSelector = () => (
    <SoftwareOptionsSelector
      formData={formData}
      onToggleSelfService={onToggleSelfService}
      onSelectCategory={onSelectCategory}
      isEditingSoftware={isEditingSoftware}
      onClickPreviewEndUserExperience={() =>
        onClickPreviewEndUserExperience(isIpaPackage)
      }
      teamId={teamId}
    />
  );

  const renderTargetLabelSelector = () => (
    <DropdownTargetLabelSelector
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

  return (
    <div className={classNames}>
      <form className={`${baseClass}__form`} onSubmit={onFormSubmit}>
        <FileUploader
          canEdit={canEditFile}
          graphicName={getGraphicName(ext || "")}
          accept={restrictedFileAccept ?? ACCEPTED_EXTENSIONS}
          message={restrictedFileTypeLabel ?? renderFileTypeMessage()}
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
        {multiPackageContext &&
          (gitOpsModeEnabled ? (
            // Preserves the apostrophe typo "it's" verbatim from Figma page
            // 2:130 so copy stays in sync with design.
            <InfoBanner
              icon="info"
              className={`${baseClass}__multi-package-banner`}
              borderRadius="medium"
            >
              Add custom packages in GitOps mode so Fleet can host your
              software. After adding, copy it&apos;s SHA-256 hash into your YAML
              so the next GitOps workflow doesn&apos;t delete it.{" "}
              <CustomLink
                url={`${LEARN_MORE_ABOUT_BASE_LINK}/software-yaml`}
                text="YAML docs"
                newTab
              />
            </InfoBanner>
          ) : (
            <InfoBanner
              icon="info"
              className={`${baseClass}__multi-package-banner`}
              borderRadius="medium"
            >
              If multiple packages target the same host, Fleet will install the
              one that was added first.
            </InfoBanner>
          ))}
        {(showDeploySoftwareSlider ||
          showSoftwareOptionsSelector ||
          showTargetLabelSelector) && ( // Only show container if any one component will render — avoids stray gap spacing
          <div
            // including `form` class here keeps the children fields subject to the global form
            // children styles
            className={
              gitopsCompatible && gitOpsModeEnabled
                ? `${baseClass}__form-fields--gitops-disabled form`
                : "form"
            }
          >
            {showDeploySoftwareSlider && renderSoftwareDeploySlider()}
            {(showSoftwareOptionsSelector || showTargetLabelSelector) && (
              <div className={`${baseClass}__form-frame`}>
                {showSoftwareOptionsSelector && renderSoftwareOptionsSelector()}
                {showTargetLabelSelector && renderTargetLabelSelector()}
              </div>
            )}
          </div>
        )}
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
          {(() => {
            // Single source of truth for the submit button — both the
            // tooltipped and non-tooltipped branches need identical text,
            // disabled state, and type. A previous duplication let the
            // "Save" / "Add software" copy drift between branches (#48400).
            const submitButton = (
              <Button type="submit" disabled={isSubmitDisabled}>
                {isEditingSoftware || multiPackageContext
                  ? "Save"
                  : "Add software"}
              </Button>
            );
            return submitTooltipContent ? (
              <TooltipWrapper
                tipContent={submitTooltipContent}
                underline={false}
                showArrow
                tipOffset={10}
                position="left"
              >
                {submitButton}
              </TooltipWrapper>
            ) : (
              submitButton
            );
          })()}

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
