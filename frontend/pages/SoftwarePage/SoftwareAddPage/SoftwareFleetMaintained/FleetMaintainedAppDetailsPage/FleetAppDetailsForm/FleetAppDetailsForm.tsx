/** FleetAppDetailsForm is a separate component remnant of when we had advanced options on add <4.83 */

import React, { useState } from "react";
import { useQuery } from "react-query";

import useGitOpsMode from "hooks/useGitOpsMode";

import { SoftwareCategory } from "interfaces/software";
import { ILabelSummary } from "interfaces/label";

import { getPathWithQueryParams } from "utilities/url";
import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";
import paths from "router/paths";
import labelsAPI, { getCustomLabels } from "services/entities/labels";

import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";
import CustomLink from "components/CustomLink";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import RevealButton from "components/buttons/RevealButton";
import { DropdownTargetLabelSelector } from "components/TargetLabelSelector";
import SoftwareOptionsSelector from "pages/SoftwarePage/components/forms/SoftwareOptionsSelector";
import AdvancedOptionsFields from "pages/SoftwarePage/components/forms/AdvancedOptionsFields";
import {
  PatchOption,
  SoftwareDeploySelector,
} from "pages/SoftwarePage/components/forms/SoftwareDeploySelector";
import {
  CUSTOM_TARGET_OPTIONS,
  generateHelpText,
} from "pages/SoftwarePage/helpers";

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
          fleet_id: teamId,
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
  forceInstall: boolean;
  patch: boolean;
  patchOption: PatchOption;
  installScript: string;
  preInstallQuery?: string;
  postInstallScript?: string;
  uninstallScript?: string;
  targetType: string;
  customTarget: string;
  labelTargets: Record<string, boolean>;
  categories: string[];
}

export interface IFormValidation {
  isValid: boolean;
  preInstallQuery?: { isValid: boolean; message?: string };
  customTarget?: { isValid: boolean };
}

interface IFleetAppDetailsFormProps {
  categories?: SoftwareCategory[] | null;
  defaultInstallScript: string;
  defaultPostInstallScript: string;
  defaultUninstallScript: string;
  teamId?: string;
  onCancel: () => void;
  onSubmit: (formData: IFleetMaintainedAppFormData) => void;
  softwareTitleId?: number;
}

const FleetAppDetailsForm = ({
  categories,
  defaultInstallScript,
  defaultPostInstallScript,
  defaultUninstallScript,
  teamId,
  onCancel,
  onSubmit,
  softwareTitleId,
}: IFleetAppDetailsFormProps) => {
  const [formData, setFormData] = useState<IFleetMaintainedAppFormData>({
    selfService: false,
    forceInstall: false,
    patch: false,
    patchOption: "closed",
    preInstallQuery: "",
    installScript: defaultInstallScript,
    postInstallScript: defaultPostInstallScript,
    uninstallScript: defaultUninstallScript,
    targetType: "All hosts",
    customTarget: "labelsIncludeAny",
    labelTargets: {},
    categories: categories || [],
  });

  const [formValidation, setFormValidation] = useState<IFormValidation>({
    isValid: true,
  });

  // Fetch labels for DropdownTargetLabelSelector
  const {
    data: labels,
    isLoading: isLoadingLabels,
    isError: isErrorLabels,
  } = useQuery<ILabelSummary[], Error>(
    ["custom_labels", teamId],
    () =>
      labelsAPI
        .summary(teamId ? parseInt(teamId, 10) : null)
        .then((res) => getCustomLabels(res.labels)),
    { ...DEFAULT_USE_QUERY_OPTIONS }
  );

  const { gitOpsModeEnabled } = useGitOpsMode("software");
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);

  const onToggleSelfService = () => {
    setFormData((prevData: IFleetMaintainedAppFormData) => ({
      ...prevData,
      selfService: !prevData.selfService,
    }));
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

  const onChangeInstallScript = (value: string) => {
    setFormData((prevData) => ({ ...prevData, installScript: value }));
  };

  const onChangePreInstallQuery = (value?: string) => {
    const newData = { ...formData, preInstallQuery: value };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const onChangePostInstallScript = (value?: string) => {
    setFormData((prevData) => ({ ...prevData, postInstallScript: value }));
  };

  const onChangeUninstallScript = (value?: string) => {
    setFormData((prevData) => ({ ...prevData, uninstallScript: value }));
  };

  const onSubmitForm = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit(formData);
  };

  const isSoftwareAlreadyAdded = !!softwareTitleId;
  const isSubmitDisabled = isSoftwareAlreadyAdded || !formValidation.isValid;

  return (
    <form className={baseClass} onSubmit={onSubmitForm}>
      <SoftwareOptionsSelector
        formData={formData}
        onToggleSelfService={onToggleSelfService}
        onClickPreviewEndUserExperience={() => undefined}
        onSelectCategory={() => undefined}
      />
      <GitOpsModeTooltipWrapper
        entityType="software"
        renderChildren={(disableChildren) => (
          <SoftwareDeploySelector
            forceInstall={formData.forceInstall}
            patch={formData.patch}
            patchOption={formData.patchOption}
            onToggleForceInstall={(forceInstall) =>
              setFormData((prevData) => ({ ...prevData, forceInstall }))
            }
            onTogglePatch={(patch) =>
              setFormData((prevData) => ({ ...prevData, patch }))
            }
            onSelectPatchOption={(patchOption) =>
              setFormData((prevData) => ({ ...prevData, patchOption }))
            }
            disabled={disableChildren}
          />
        )}
      />
      <GitOpsModeTooltipWrapper
        entityType="software"
        renderChildren={(disableChildren) => (
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
            isLoadingLabels={isLoadingLabels}
            isErrorLabels={isErrorLabels}
            dropdownHelpText={generateHelpText(
              formData.forceInstall,
              formData.customTarget
            )}
            disableOptions={disableChildren}
          />
        )}
      />
      <div className={`${baseClass}__advanced-options`}>
        <RevealButton
          isShowing={showAdvancedOptions}
          showText="Advanced options"
          hideText="Advanced options"
          caretPosition="after"
          onClick={() => setShowAdvancedOptions(!showAdvancedOptions)}
        />
        {showAdvancedOptions && (
          <AdvancedOptionsFields
            showSchemaButton={false}
            installScriptHelpText={
              <>
                Use the $INSTALLER_PATH variable to point to the installer.{" "}
                <CustomLink
                  url={`${LEARN_MORE_ABOUT_BASE_LINK}/install-scripts`}
                  text="Learn more about install scripts"
                  newTab
                />
              </>
            }
            postInstallScriptHelpText=""
            uninstallScriptHelpText={
              <>
                $PACKAGE_ID will be populated after the software is added.{" "}
                <CustomLink
                  url={`${LEARN_MORE_ABOUT_BASE_LINK}/uninstall-scripts`}
                  text="Learn more about uninstall scripts"
                  newTab
                />
              </>
            }
            errors={{
              preInstallQuery: formValidation.preInstallQuery?.message,
            }}
            preInstallQuery={formData.preInstallQuery}
            installScript={formData.installScript}
            postInstallScript={formData.postInstallScript}
            uninstallScript={formData.uninstallScript}
            onClickShowSchema={() => undefined}
            onChangePreInstallQuery={onChangePreInstallQuery}
            onChangeInstallScript={onChangeInstallScript}
            onChangePostInstallScript={onChangePostInstallScript}
            onChangeUninstallScript={onChangeUninstallScript}
            gitopsCompatible
            gitOpsModeEnabled={gitOpsModeEnabled}
            patchWhenClosed={
              formData.patch && formData.patchOption === "closed"
            }
          />
        )}
      </div>
      <div className={`${baseClass}__action-buttons`}>
        <GitOpsModeTooltipWrapper
          entityType="software"
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
        <Button onClick={onCancel} variant="secondary">
          Cancel
        </Button>
      </div>
    </form>
  );
};

export default FleetAppDetailsForm;
