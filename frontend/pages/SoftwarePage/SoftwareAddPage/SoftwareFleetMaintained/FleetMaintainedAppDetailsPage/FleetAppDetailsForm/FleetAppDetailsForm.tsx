import React, { useContext, useState } from "react";
import { AppContext } from "context/app";

import { ILabelSummary } from "interfaces/label";
import {
  CUSTOM_TARGET_OPTIONS,
  generateHelpText,
} from "pages/SoftwarePage/helpers";
import { getPathWithQueryParams } from "utilities/url";
import paths from "router/paths";

import RevealButton from "components/buttons/RevealButton";
import Button from "components/buttons/Button";
import Card from "components/Card";
import SoftwareOptionsSelector from "components/SoftwareOptionsSelector";
import TargetLabelSelector from "components/TargetLabelSelector";
import TooltipWrapper from "components/TooltipWrapper";
import CustomLink from "components/CustomLink";
import AdvancedOptionsFields from "pages/SoftwarePage/components/AdvancedOptionsFields";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import { generateFormValidation } from "./helpers";

const baseClass = "fleet-app-details-form";

export const softwareAlreadyAddedTipContent = (
  softwareTitleId?: number,
  teamId?: string
) => {
  const pathToSoftwareTitles = softwareTitleId
    ? getPathWithQueryParams(
        paths.SOFTWARE_TITLE_DETAILS(softwareTitleId.toString()),
        {
          team_id: teamId,
        }
      )
    : "";
  return (
    <>
      You already added this software.
      <br />
      <CustomLink
        url={pathToSoftwareTitles}
        text="View software"
        variant="tooltip-link"
      />
    </>
  );
};
export interface IFleetMaintainedAppFormData {
  selfService: boolean;
  automaticInstall: boolean;
  installScript: string;
  preInstallQuery?: string;
  postInstallScript?: string;
  uninstallScript?: string;
  targetType: string;
  customTarget: string;
  labelTargets: Record<string, boolean>;
}

export interface IFormValidation {
  isValid: boolean;
  preInstallQuery?: { isValid: boolean; message?: string };
  customTarget?: { isValid: boolean };
}

interface IFleetAppDetailsFormProps {
  labels: ILabelSummary[] | null;
  name: string;
  defaultInstallScript: string;
  defaultPostInstallScript: string;
  defaultUninstallScript: string;
  teamId?: string;
  showSchemaButton: boolean;
  onClickShowSchema: () => void;
  onCancel: () => void;
  onSubmit: (formData: IFleetMaintainedAppFormData) => void;
  softwareTitleId?: number;
}

const FleetAppDetailsForm = ({
  labels,
  name: appName,
  defaultInstallScript,
  defaultPostInstallScript,
  defaultUninstallScript,
  teamId,
  showSchemaButton,
  onClickShowSchema,
  onCancel,
  onSubmit,
  softwareTitleId,
}: IFleetAppDetailsFormProps) => {
  const gitOpsModeEnabled = useContext(AppContext).config?.gitops
    .gitops_mode_enabled;

  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);

  const [formData, setFormData] = useState<IFleetMaintainedAppFormData>({
    selfService: false,
    automaticInstall: false,
    preInstallQuery: undefined,
    installScript: defaultInstallScript,
    postInstallScript: defaultPostInstallScript,
    uninstallScript: defaultUninstallScript,
    targetType: "All hosts",
    customTarget: "labelsIncludeAny",
    labelTargets: {},
  });
  const [formValidation, setFormValidation] = useState<IFormValidation>({
    isValid: true,
    preInstallQuery: { isValid: false },
  });

  const onChangePreInstallQuery = (value?: string) => {
    const newData = { ...formData, preInstallQuery: value };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const onChangeInstallScript = (value: string) => {
    const newData = { ...formData, installScript: value };
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

  const onToggleAutomaticInstallCheckbox = (value: boolean) => {
    const newData = { ...formData, automaticInstall: value };
    setFormData(newData);
  };

  const onSelectTargetType = (value: string) => {
    const newData = { ...formData, targetType: value };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const onSelectCustomTargetOption = (value: string) => {
    const newData = { ...formData, customTarget: value };
    setFormData(newData);
  };

  const onSelectLabel = ({ name, value }: { name: string; value: boolean }) => {
    const newData = {
      ...formData,
      labelTargets: { ...formData.labelTargets, [name]: value },
    };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const onSubmitForm = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit(formData);
  };

  const gitOpsModeDisabledClass = gitOpsModeEnabled
    ? "form-fields--disabled"
    : "";
  const isSoftwareAlreadyAdded = !!softwareTitleId;
  const isSubmitDisabled = !formValidation.isValid || isSoftwareAlreadyAdded;

  return (
    <form className={`${baseClass}`} onSubmit={onSubmitForm}>
      <div className={`${baseClass}__form-frame ${gitOpsModeDisabledClass}`}>
        <Card paddingSize="medium" borderRadiusSize="large">
          <SoftwareOptionsSelector
            formData={formData}
            onToggleAutomaticInstall={onToggleAutomaticInstallCheckbox}
            onToggleSelfService={onToggleSelfServiceCheckbox}
            disableOptions={isSoftwareAlreadyAdded}
          />
        </Card>
        <Card paddingSize="medium" borderRadiusSize="large">
          <TargetLabelSelector
            selectedTargetType={formData.targetType}
            selectedCustomTarget={formData.customTarget}
            selectedLabels={formData.labelTargets}
            customTargetOptions={CUSTOM_TARGET_OPTIONS}
            className={`${baseClass}__target`}
            dropdownHelpText={
              formData.targetType === "Custom" &&
              generateHelpText(formData.automaticInstall, formData.customTarget)
            }
            onSelectTargetType={onSelectTargetType}
            onSelectCustomTarget={onSelectCustomTargetOption}
            onSelectLabel={onSelectLabel}
            labels={labels || []}
            disableOptions={isSoftwareAlreadyAdded}
          />
        </Card>
      </div>
      <div
        className={`${baseClass}__advanced-options-section ${gitOpsModeDisabledClass}`}
      >
        <RevealButton
          className={`${baseClass}__accordion-title`}
          isShowing={showAdvancedOptions}
          showText="Advanced options"
          hideText="Advanced options"
          caretPosition="after"
          onClick={() => setShowAdvancedOptions(!showAdvancedOptions)}
          disabled={isSoftwareAlreadyAdded}
        />
        {showAdvancedOptions && (
          <AdvancedOptionsFields
            className={`${baseClass}__advanced-options-fields`}
            showSchemaButton={showSchemaButton}
            installScriptHelpText="Use the $INSTALLER_PATH variable to point to the installer. Currently, shell scripts are supported."
            postInstallScriptHelpText="Currently, shell scripts are supported."
            uninstallScriptHelpText="Currently, shell scripts are supported."
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
        )}
      </div>
      <div className={`${baseClass}__action-buttons`}>
        <GitOpsModeTooltipWrapper
          renderChildren={(disableChildren) => (
            <TooltipWrapper
              tipContent={softwareAlreadyAddedTipContent(
                softwareTitleId,
                teamId
              )}
              disableTooltip={!isSoftwareAlreadyAdded}
              position="left"
              showArrow
              underline={false}
              tipOffset={10}
            >
              <Button
                type="submit"
                disabled={disableChildren || isSubmitDisabled}
              >
                Add software
              </Button>
            </TooltipWrapper>
          )}
        />
        <Button onClick={onCancel} variant="inverse">
          Cancel
        </Button>
      </div>
    </form>
  );
};

export default FleetAppDetailsForm;
