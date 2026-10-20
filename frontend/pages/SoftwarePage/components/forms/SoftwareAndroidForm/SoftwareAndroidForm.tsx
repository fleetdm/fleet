import classnames from "classnames";
import React, { useState } from "react";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import InputField from "components/forms/fields/InputField";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import InfoBanner from "components/InfoBanner/InfoBanner";
import { DropdownTargetLabelSelector } from "components/TargetLabelSelector";
import useGitOpsMode from "hooks/useGitOpsMode";
import { IInputFieldParseTarget } from "interfaces/form_field";
import { ILabelSummary } from "interfaces/label";
import { IAppStoreApp } from "interfaces/software";
import {
  CUSTOM_TARGET_OPTIONS,
  generateHelpText,
  generateSelectedLabels,
  getCustomTarget,
  getTargetType,
  isAndroidWebApp,
} from "pages/SoftwarePage/helpers";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import { AndroidOptionsDescription } from "../SoftwareOptionsSelector/SoftwareOptionsSelector";

import generateFormValidation from "./helpers";

const baseClass = "software-android-form";

export interface ISoftwareAndroidFormData {
  selfService: boolean;
  automaticInstall: boolean;
  targetType: string;
  customTarget: string;
  labelTargets: Record<string, boolean>;
  applicationID: string;
  categories: string[];
  platform: "android";
}

export interface IFormValidation {
  isValid: boolean;
}

interface ISoftwareAndroidFormProps {
  labels: ILabelSummary[] | null;
  softwareAndroidForEdit?: IAppStoreApp; // 4.77 Currently no edit Android functionality
  onSubmit: (formData: ISoftwareAndroidFormData) => void;
  isLoading?: boolean;
  onCancel: () => void;
}

const SoftwareAndroidForm = ({
  labels,
  softwareAndroidForEdit,
  onSubmit,
  isLoading = false,
  onCancel,
}: ISoftwareAndroidFormProps) => {
  const { gitOpsModeEnabled } = useGitOpsMode("software");

  const [formData, setFormData] = useState<ISoftwareAndroidFormData>(
    softwareAndroidForEdit
      ? {
          applicationID: softwareAndroidForEdit.app_store_id || "",
          selfService: softwareAndroidForEdit.self_service || false, // 4.77 Currently unavailable to change
          automaticInstall: softwareAndroidForEdit.automatic_install || false, // 4.77 Currently unavailable for Android apps
          targetType: getTargetType(softwareAndroidForEdit),
          customTarget: getCustomTarget(softwareAndroidForEdit),
          labelTargets: generateSelectedLabels(softwareAndroidForEdit),
          categories: softwareAndroidForEdit.categories || [],
          platform: "android",
        }
      : {
          applicationID: "",
          selfService: true, // Default to true for new Android apps
          automaticInstall: false, // 4.77 Currently not available for Android apps
          targetType: "All hosts",
          customTarget: "labelsIncludeAny",
          labelTargets: {},
          categories: [],
          platform: "android",
        }
  );

  const [formValidation, setFormValidation] = useState<IFormValidation>({
    isValid: !!softwareAndroidForEdit, // Disables submit before Android application ID is entered
  });

  const onFormSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit(formData);
  };

  const onInputChange = ({ name, value }: IInputFieldParseTarget) => {
    const newFormData = { ...formData, [name]: value };
    setFormData(newFormData);
    setFormValidation(generateFormValidation(newFormData));
  };

  const onSelectTargetType = (value: string) => {
    const newData = { ...formData, targetType: value };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const onSelectCustomTargetOption = (value: string) => {
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

  const renderContent = () => {
    // Edit Android does not exist on 4.77
    if (softwareAndroidForEdit) {
      return null;
    }

    // Add Android form
    return (
      <>
        <div className={`${baseClass}__form-fields`}>
          <InputField
            autofocus
            label="Application ID"
            placeholder="com.android.chrome"
            helpText={
              <>
                The ID at the end of the app&apos;s{" "}
                <CustomLink
                  text="Google Play URL"
                  url={`${LEARN_MORE_ABOUT_BASE_LINK}/google-play-store`}
                  newTab
                />{" "}
                E.g. &quot;com.android.chrome&quot; from
                &quot;https://play.google.com/store/apps/details?id=com.android.chrome&quot;
              </>
            }
            onChange={onInputChange}
            name="applicationID"
            value={formData.applicationID}
            parseTarget
            disabled={gitOpsModeEnabled} // TODO: Confirm GitOps behavior
          />
        </div>
        {isAndroidWebApp(formData.applicationID) && (
          <InfoBanner
            color="yellow"
            cta={
              <CustomLink
                url={`${LEARN_MORE_ABOUT_BASE_LINK}/android-web-apps-chrome-required`}
                text="Learn more"
                newTab
              />
            }
          >
            This is an Android web app and it requires Google Chrome to work.
            Please make sure you add Google Chrome to this fleet.
          </InfoBanner>
        )}
        <div>
          <AndroidOptionsDescription />
        </div>
        <DropdownTargetLabelSelector
          selectedTargetType={formData.targetType}
          selectedCustomTarget={formData.customTarget}
          selectedLabels={formData.labelTargets}
          customTargetOptions={CUSTOM_TARGET_OPTIONS}
          className={`${baseClass}__target`}
          onSelectTargetType={onSelectTargetType}
          onSelectCustomTarget={onSelectCustomTargetOption}
          onSelectLabel={onSelectLabel}
          labels={labels || []}
          dropdownHelpText={
            generateHelpText(false, formData.customTarget) // maps to !automaticInstall help text
          }
          disableOptions={gitOpsModeEnabled}
        />
      </>
    );
  };

  const contentWrapperClasses = classnames(`${baseClass}__content-wrapper`, {
    [`${baseClass}__content-disabled`]: isLoading,
  });

  const formContentClasses = classnames(`${baseClass}__form-content`, {
    [`${baseClass}__form-content--disabled`]: gitOpsModeEnabled,
  });

  return (
    <form className={baseClass} onSubmit={onFormSubmit}>
      {isLoading && <div className={`${baseClass}__overlay`} />}
      <div className={contentWrapperClasses}>
        <div className={formContentClasses}>
          <>{renderContent()}</>
        </div>
        <div className={`${baseClass}__action-buttons`}>
          <GitOpsModeTooltipWrapper
            entityType="software"
            position="top"
            tipOffset={8}
            renderChildren={(disableChildren) => (
              <Button
                type="submit"
                disabled={disableChildren || isSubmitDisabled}
                isLoading={isLoading}
                className={`${baseClass}__add-software-btn`}
              >
                {softwareAndroidForEdit ? "Save" : "Add software"}
              </Button>
            )}
          />
          <Button onClick={onCancel} variant="secondary">
            Cancel
          </Button>
        </div>
      </div>
    </form>
  );
};

export default SoftwareAndroidForm;
