import React, { useState } from "react";

import { ILabelSummary } from "interfaces/label";
import { PackageType } from "interfaces/package_type";

import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";
import RevealButton from "components/buttons/RevealButton";
import Button from "components/buttons/Button";
import Radio from "components/forms/fields/Radio";
import TargetLabelSelector from "components/TargetLabelSelector";
import InfoBanner from "components/InfoBanner";
import CustomLink from "components/CustomLink";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import AdvancedOptionsFields from "pages/SoftwarePage/components/AdvancedOptionsFields";

import {
  CUSTOM_TARGET_OPTIONS,
  generateFormValidation,
  generateHelpText,
} from "./helpers";

const baseClass = "fleet-app-details-form";

export type InstallType = "manual" | "automatic";
export interface IFleetMaintainedAppFormData {
  selfService: boolean;
  installScript: string;
  preInstallQuery?: string;
  postInstallScript?: string;
  uninstallScript?: string;
  installType: InstallType;
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
  showSchemaButton: boolean;
  onClickShowSchema: () => void;
  onCancel: () => void;
  onSubmit: (formData: IFleetMaintainedAppFormData) => void;
}

interface IInstallTypeSection {
  installType: InstallType;
  onChangeInstallType: (value: string) => void;
  isCustomPackage?: boolean;
  isExeCustomPackage?: boolean;
}

// Also used in custom package form (PackageForm.tsx)
export const InstallTypeSection = ({
  installType,
  onChangeInstallType,
  isCustomPackage = false,
  isExeCustomPackage = false,
}: IInstallTypeSection) => {
  const isAutomaticDisabled = isExeCustomPackage;
  const AUTOMATIC_DISABLED_TOOLTIP = (
    <>
      Fleet can&apos;t create a policy to detect existing installations for
      <br /> .exe packages. To automatically install an .exe, add a custom
      <br /> policy and enable the install software automation on the
      <br /> <b>Policies</b> page.
    </>
  );

  return (
    <fieldset>
      <legend>Install</legend>
      <div className={`${baseClass}__radio-input`}>
        <Radio
          checked={installType === "manual"}
          id="manual-radio-btn"
          value="manual"
          name="install-type"
          label="Manual"
          onChange={onChangeInstallType}
          helpText={
            <>
              Manually install on <b>Host details</b> page for each host.
            </>
          }
        />
        <Radio
          checked={installType === "automatic"}
          id="automatic-radio-btn"
          value="automatic"
          name="install-type"
          label="Automatic"
          disabled={isAutomaticDisabled}
          tooltip={isAutomaticDisabled && AUTOMATIC_DISABLED_TOOLTIP}
          onChange={onChangeInstallType}
          helpText={
            <>
              Automatically install on each host that&apos;s{" "}
              <TooltipWrapper
                tipContent={
                  <>
                    If the host already has any version of this
                    <br /> software, it won&apos;t be installed.
                  </>
                }
              >
                missing this software
              </TooltipWrapper>
              . Policy that triggers install can be customized after software is
              added.
            </>
          }
        />
      </div>
      {installType === "automatic" && isCustomPackage && (
        <InfoBanner
          color="yellow"
          cta={
            <CustomLink
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/automatic-software-install-policies`}
              text="Learn more"
              newTab
            />
          }
        >
          Installing software over existing installations might cause issues.
          Fleet&apos;s policy may not detect these existing installations.
          Please create a test team in Fleet to verify a smooth installation.
        </InfoBanner>
      )}
    </fieldset>
  );
};

const FleetAppDetailsForm = ({
  labels,
  name: appName,
  defaultInstallScript,
  defaultPostInstallScript,
  defaultUninstallScript,
  showSchemaButton,
  onClickShowSchema,
  onCancel,
  onSubmit,
}: IFleetAppDetailsFormProps) => {
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);

  const [formData, setFormData] = useState<IFleetMaintainedAppFormData>({
    selfService: false,
    preInstallQuery: undefined,
    installScript: defaultInstallScript,
    postInstallScript: defaultPostInstallScript,
    uninstallScript: defaultUninstallScript,
    installType: "manual",
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

  const onChangeInstallType = (value: string) => {
    const installType = value as InstallType;
    const newData = { ...formData, installType };
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

  const isSubmitDisabled = !formValidation.isValid;

  return (
    <form className={baseClass} onSubmit={onSubmitForm}>
      <InstallTypeSection
        installType={formData.installType}
        onChangeInstallType={onChangeInstallType}
      />
      <TargetLabelSelector
        selectedTargetType={formData.targetType}
        selectedCustomTarget={formData.customTarget}
        selectedLabels={formData.labelTargets}
        customTargetOptions={CUSTOM_TARGET_OPTIONS}
        className={`${baseClass}__target`}
        dropdownHelpText={
          formData.targetType === "Custom" &&
          generateHelpText(formData.installType, formData.customTarget)
        }
        onSelectTargetType={onSelectTargetType}
        onSelectCustomTarget={onSelectCustomTargetOption}
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
              End users can install from <b>Fleet Desktop {">"} Self-service</b>
              .
            </>
          }
        >
          Self-service
        </TooltipWrapper>
      </Checkbox>
      <div className={`${baseClass}__advanced-options-section`}>
        <RevealButton
          className={`${baseClass}__accordion-title`}
          isShowing={showAdvancedOptions}
          showText="Advanced options"
          hideText="Advanced options"
          caretPosition="after"
          onClick={() => setShowAdvancedOptions(!showAdvancedOptions)}
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
      <div className={`${baseClass}__form-buttons`}>
        <Button type="submit" variant="brand" disabled={isSubmitDisabled}>
          Add software
        </Button>
        <Button onClick={onCancel} variant="inverse">
          Cancel
        </Button>
      </div>
    </form>
  );
};

export default FleetAppDetailsForm;
